package terraform

import (
	"encoding/json"
	"reflect"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
)

// TestGetAllResources verifies the pure address-collecting logic that feeds
// state-list's -o json output: managed resources are kept, data sources are
// skipped, and child modules are walked recursively.
func TestGetAllResources(t *testing.T) {
	root := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{Address: "aws_s3_bucket.this", Type: "aws_s3_bucket", Mode: tfjson.ManagedResourceMode},
			// In real state a data source's Type carries no "data." prefix;
			// only Address and Mode identify it.
			{Address: "data.aws_caller_identity.current", Type: "aws_caller_identity", Mode: tfjson.DataResourceMode},
		},
		ChildModules: []*tfjson.StateModule{
			{
				Resources: []*tfjson.StateResource{
					{Address: "module.child.aws_iam_role.this", Type: "aws_iam_role", Mode: tfjson.ManagedResourceMode},
				},
			},
		},
	}

	got := getAllResources(root)
	want := []string{"aws_s3_bucket.this", "module.child.aws_iam_role.this"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("getAllResources() = %v, want %v", got, want)
	}
}

func TestGetAllResources_NilModule(t *testing.T) {
	if got := getAllResources(nil); got != nil {
		t.Errorf("getAllResources(nil) = %v, want nil", got)
	}
}

// TestResourceAddressesForJSON verifies the empty-state case marshals to an
// empty JSON array ("[]") rather than "null", which would confuse a
// pipeline consuming smurf stf state-list -o json.
func TestResourceAddressesForJSON(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  string
	}{
		{name: "nil slice", input: nil, want: "[]"},
		{name: "empty slice", input: []string{}, want: "[]"},
		{name: "populated slice", input: []string{"aws_s3_bucket.this", "aws_iam_role.this"}, want: `["aws_s3_bucket.this","aws_iam_role.this"]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(resourceAddressesForJSON(tc.input))
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if string(data) != tc.want {
				t.Errorf("json.Marshal(resourceAddressesForJSON(%v)) = %s, want %s", tc.input, data, tc.want)
			}
		})
	}
}
