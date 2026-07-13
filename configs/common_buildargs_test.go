package configs

import "testing"

func TestParseBuildArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "single pair",
			args: []string{"NODE_ENV=production"},
			want: map[string]string{"NODE_ENV": "production"},
		},
		{
			name: "comma separated in one flag",
			args: []string{"NODE_ENV=production,API_URL=https://example.com"},
			want: map[string]string{
				"NODE_ENV": "production",
				"API_URL":  "https://example.com",
			},
		},
		{
			name: "repeated flags",
			args: []string{"FOO=bar", "BAZ=qux"},
			want: map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name: "later flag overrides earlier",
			args: []string{"FOO=bar", "FOO=baz"},
			want: map[string]string{"FOO": "baz"},
		},
		{
			name: "value with comma stays intact",
			args: []string{"MESSAGE=hello,world"},
			want: map[string]string{"MESSAGE": "hello,world"},
		},
		{
			name:    "invalid entry",
			args:    []string{"not-a-build-arg"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseBuildArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for key, wantVal := range tt.want {
				if gotVal, ok := got[key]; !ok || gotVal != wantVal {
					t.Fatalf("key %q: got %q, want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}
