package terraform

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

func TestFormatFailureMessage(t *testing.T) {
	got := formatFailureMessage([]string{"aws_s3_bucket.a", "aws_iam_role.b"})
	for _, want := range []string{"Failed to remove resources", "aws_s3_bucket.a", "aws_iam_role.b", "insufficient permissions"} {
		if !strings.Contains(got, want) {
			t.Errorf("message missing %q; got: %s", want, got)
		}
	}
}

func TestGetBasicStateInfoFromData(t *testing.T) {
	t.Run("serial and managed resource count", func(t *testing.T) {
		data := []byte(`{"serial": 42, "resources": [{"mode":"managed"},{"mode":"data"},{"mode":"managed"}]}`)
		info, err := getBasicStateInfoFromData(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The deliberately simple parser slices from just after the "serial"
		// key, so the extracted value still carries the ": " delimiter.
		if !strings.Contains(info["serial"], "42") {
			t.Errorf("serial = %q, want it to contain 42", info["serial"])
		}
		// Only "managed" resources are counted (the data source is ignored).
		if info["resources"] != "2" {
			t.Errorf("resources = %q, want 2", info["resources"])
		}
	})

	t.Run("empty state defaults", func(t *testing.T) {
		info, err := getBasicStateInfoFromData([]byte(`{}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info["serial"] != "0" {
			t.Errorf("serial = %q, want 0", info["serial"])
		}
		if info["resources"] != "0" {
			t.Errorf("resources = %q, want 0", info["resources"])
		}
	})
}

func TestFormatPushErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"locked", errors.New("state is locked by lock id"), "State is locked"},
		{"newer serial", errors.New("serial is newer than expected"), "higher serial number"},
		{"access denied", errors.New("access denied to bucket"), "Permission denied"},
		{"default", errors.New("some other failure"), "Failed to push state"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatPushErrorMessage(c.err); !strings.Contains(got, c.want) {
				t.Errorf("formatPushErrorMessage(%v) missing %q; got: %s", c.err, c.want, got)
			}
		})
	}
}

func TestFormatPullErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"no state", errors.New("no state file was found"), "No remote state found"},
		{"permission", errors.New("permission denied"), "Permission denied"},
		{"timeout", errors.New("context deadline exceeded"), "Timeout"},
		{"no host", errors.New("dial tcp: no such host"), "resolve backend hostname"},
		{"default", errors.New("weird error"), "Failed to pull remote state"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatPullErrorMessage(c.err); !strings.Contains(got, c.want) {
				t.Errorf("formatPullErrorMessage(%v) missing %q; got: %s", c.err, c.want, got)
			}
		})
	}
}

func TestSearchResourceInModules(t *testing.T) {
	modules := []*tfjson.StateModule{
		{
			Resources: []*tfjson.StateResource{{Address: "aws_s3_bucket.a"}},
			ChildModules: []*tfjson.StateModule{
				{Resources: []*tfjson.StateResource{{Address: "module.child.aws_iam_role.b"}}},
			},
		},
	}
	if got := searchResourceInModules(modules, "aws_s3_bucket.a"); got == nil || got.Address != "aws_s3_bucket.a" {
		t.Errorf("expected to find root resource, got %v", got)
	}
	if got := searchResourceInModules(modules, "module.child.aws_iam_role.b"); got == nil || got.Address != "module.child.aws_iam_role.b" {
		t.Errorf("expected to find child resource, got %v", got)
	}
	if got := searchResourceInModules(modules, "does.not.exist"); got != nil {
		t.Errorf("expected nil for missing address, got %v", got)
	}
}

func TestContainsAction(t *testing.T) {
	actions := []tfjson.Action{tfjson.ActionCreate, tfjson.ActionUpdate}
	if !containsAction(actions, tfjson.ActionCreate) {
		t.Error("expected create to be present")
	}
	if containsAction(actions, tfjson.ActionDelete) {
		t.Error("did not expect delete")
	}
	if containsAction(nil, tfjson.ActionRead) {
		t.Error("empty actions should contain nothing")
	}
}

func TestFormatActions(t *testing.T) {
	got := formatActions([]tfjson.Action{tfjson.ActionCreate, tfjson.ActionUpdate, tfjson.ActionDelete, tfjson.ActionRead})
	for _, want := range []string{"create", "update", "delete", "read"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatActions missing %q; got %q", want, got)
		}
	}
	// verbs are comma-joined
	if strings.Count(got, ",") != 3 {
		t.Errorf("expected 3 separators, got %q", got)
	}
}

func TestGetAllRefreshResources(t *testing.T) {
	root := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{{Address: "a"}},
		ChildModules: []*tfjson.StateModule{
			{Resources: []*tfjson.StateResource{{Address: "b"}, {Address: "c"}}},
		},
	}
	got := getAllRefreshResources(root)
	if len(got) != 3 {
		t.Errorf("expected 3 resources, got %d", len(got))
	}
	if getAllRefreshResources(nil) != nil {
		t.Error("nil module should return nil")
	}
}

func TestCountResources(t *testing.T) {
	root := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{{Address: "a"}, {Address: "b"}},
		ChildModules: []*tfjson.StateModule{
			{Resources: []*tfjson.StateResource{{Address: "c"}}},
			{Resources: []*tfjson.StateResource{{Address: "d"}}, ChildModules: []*tfjson.StateModule{
				{Resources: []*tfjson.StateResource{{Address: "e"}}},
			}},
		},
	}
	if got := countResources(root); got != 5 {
		t.Errorf("countResources = %d, want 5", got)
	}
	if got := countResources(nil); got != 0 {
		t.Errorf("countResources(nil) = %d, want 0", got)
	}
}

func TestGetResourceAddress(t *testing.T) {
	r := &tfjson.StateResource{Address: "aws_s3_bucket.this"}
	if got := getResourceAddress(r, ""); got != "aws_s3_bucket.this" {
		t.Errorf("empty prefix: got %q", got)
	}
	// With a prefix that is not already part of the address, it is prepended.
	if got := getResourceAddress(r, "module.a"); got != "module.a.aws_s3_bucket.this" {
		t.Errorf("with prefix: got %q", got)
	}
	// If the address already carries the prefix, it is not duplicated.
	r2 := &tfjson.StateResource{Address: "module.a.aws_iam_role.x"}
	if got := getResourceAddress(r2, "module.a"); got != "module.a.aws_iam_role.x" {
		t.Errorf("prefixed address: got %q", got)
	}
}

func TestGenerateDOTGraph(t *testing.T) {
	t.Run("nil state still emits a valid graph shell", func(t *testing.T) {
		got := generateDOTGraph(nil)
		if !strings.Contains(got, "digraph G {") || !strings.HasSuffix(got, "}\n") {
			t.Errorf("unexpected graph for nil state: %q", got)
		}
	})

	t.Run("resources become labelled nodes", func(t *testing.T) {
		state := &tfjson.State{
			Values: &tfjson.StateValues{
				RootModule: &tfjson.StateModule{
					Resources: []*tfjson.StateResource{
						{Address: "aws_s3_bucket.this"},
						{Address: "aws_iam_role.this", DependsOn: []string{"aws_s3_bucket.this"}},
					},
				},
			},
		}
		got := generateDOTGraph(state)
		if !strings.Contains(got, `"aws_s3_bucket.this"`) {
			t.Errorf("graph missing bucket node: %s", got)
		}
		if !strings.Contains(got, `"aws_iam_role.this" -> "aws_s3_bucket.this"`) {
			t.Errorf("graph missing dependency edge: %s", got)
		}
	})
}

func TestOutputsToJSON(t *testing.T) {
	outputs := map[string]tfexec.OutputMeta{
		"region":   {Value: json.RawMessage(`"us-east-1"`)},
		"password": {Sensitive: true, Value: json.RawMessage(`"hunter2"`)},
		"count":    {Value: json.RawMessage(`3`)},
		"broken":   {Value: json.RawMessage(`not-json`)},
	}
	got := outputsToJSON(outputs)
	if got["region"] != "us-east-1" {
		t.Errorf("region = %v, want us-east-1", got["region"])
	}
	if got["password"] != "[sensitive value hidden]" {
		t.Errorf("password = %v, want hidden marker", got["password"])
	}
	// JSON numbers decode to float64.
	if got["count"] != float64(3) {
		t.Errorf("count = %v (%T), want 3", got["count"], got["count"])
	}
	// Invalid JSON falls back to the raw string.
	if got["broken"] != "not-json" {
		t.Errorf("broken = %v, want raw string fallback", got["broken"])
	}
}

func TestColorizeNoChanges(t *testing.T) {
	plan := "Terraform will perform the following actions\nNo changes. Your infrastructure matches the configuration\ndone"
	got := colorizeNoChanges(plan)
	lines := strings.Split(got, "\n")
	// The "No changes." line is wrapped in the green ANSI escape.
	if !strings.Contains(lines[1], "\033[32m") || !strings.Contains(lines[1], "\033[0m") {
		t.Errorf("expected no-changes line to be colorized, got %q", lines[1])
	}
	// Unrelated lines are left untouched.
	if lines[0] != "Terraform will perform the following actions" || lines[2] != "done" {
		t.Errorf("unrelated lines were modified: %q", got)
	}
}

func TestBuildPlanOptions(t *testing.T) {
	t.Run("counts options and rejects a missing var file", func(t *testing.T) {
		_, err := buildPlanOptions(nil, []string{"/no/such/file.tfvars"}, nil, "")
		if err == nil {
			t.Fatal("expected an error for a missing var file")
		}
		if !strings.Contains(err.Error(), "variable file not found") {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("builds options for present inputs", func(t *testing.T) {
		dir := t.TempDir()
		vf := filepath.Join(dir, "prod.tfvars")
		if err := os.WriteFile(vf, []byte("region = \"us\"\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		opts, err := buildPlanOptions([]string{"a=1", "b=2"}, []string{vf}, []string{"aws_s3_bucket.x"}, "terraform.tfstate")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// state + 2 vars + 1 var-file + 1 target = 5 options.
		if len(opts) != 5 {
			t.Errorf("len(opts) = %d, want 5", len(opts))
		}
	})
}

func TestBuildApplyOptions(t *testing.T) {
	// lock + dirOrPlan are always present; state and each target add one more.
	opts := buildApplyOptions(true, "plan.out", "terraform.tfstate", []string{"a", "b"})
	if len(opts) != 5 {
		t.Errorf("len(opts) = %d, want 5", len(opts))
	}
	optsNoState := buildApplyOptions(false, ".", "", nil)
	if len(optsNoState) != 2 {
		t.Errorf("len(optsNoState) = %d, want 2", len(optsNoState))
	}
}

func TestAddVarsAndFiles(t *testing.T) {
	t.Run("missing var file errors", func(t *testing.T) {
		_, err := addVarsAndFiles(nil, []string{"/no/such/file.tfvars"}, nil)
		if err == nil || !strings.Contains(err.Error(), "variable file not found") {
			t.Fatalf("expected missing-file error, got %v", err)
		}
	})
	t.Run("appends vars and files", func(t *testing.T) {
		dir := t.TempDir()
		vf := filepath.Join(dir, "v.tfvars")
		if err := os.WriteFile(vf, []byte("x = 1\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		base := []tfexec.ApplyOption{tfexec.Lock(true)}
		opts, err := addVarsAndFiles([]string{"a=1"}, []string{vf}, base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// base(1) + 1 var-file + 1 var = 3.
		if len(opts) != 3 {
			t.Errorf("len(opts) = %d, want 3", len(opts))
		}
	})
}

func TestTerraformCommand(t *testing.T) {
	got := terraformCommand()
	if got == "" {
		t.Fatal("terraformCommand returned empty string")
	}
	// It either resolves to a known secure absolute path or falls back to the bare name.
	if got != "terraform" && !filepath.IsAbs(got) {
		t.Errorf("terraformCommand = %q, want an absolute path or \"terraform\"", got)
	}
}
