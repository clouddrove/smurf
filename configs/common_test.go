package configs

import (
	"strings"
	"testing"
)

func TestParseImage(t *testing.T) {
	cases := []struct {
		name     string
		image    string
		wantName string
		wantTag  string
	}{
		{"name and tag", "nginx:1.25", "nginx", "1.25"},
		{"no tag defaults to latest", "nginx", "nginx", "latest"},
		{"empty defaults to latest", "", "", "latest"},
		{"registry port splits at first colon", "myregistry:5000/img", "myregistry", "5000/img"},
		{"repo path with tag", "clouddrove/smurf:v2", "clouddrove/smurf", "v2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			name, tag, err := ParseImage(c.image)
			if err != nil {
				t.Fatalf("ParseImage(%q) unexpected error: %v", c.image, err)
			}
			if name != c.wantName {
				t.Errorf("ParseImage(%q) name = %q, want %q", c.image, name, c.wantName)
			}
			if tag != c.wantTag {
				t.Errorf("ParseImage(%q) tag = %q, want %q", c.image, tag, c.wantTag)
			}
		})
	}
}

func TestSplitKeyValue(t *testing.T) {
	cases := []struct {
		name      string
		arg       string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		{"key and value", "key=value", "key", "value", true},
		{"empty value", "key=", "key", "", true},
		{"empty key", "=value", "", "value", true},
		{"no equals", "noequals", "noequals", "", false},
		{"multiple equals splits once", "a=b=c", "a", "b=c", true},
		{"empty string", "", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			key, value, ok := SplitKeyValue(c.arg)
			if key != c.wantKey || value != c.wantValue || ok != c.wantOK {
				t.Errorf("SplitKeyValue(%q) = (%q, %q, %v), want (%q, %q, %v)",
					c.arg, key, value, ok, c.wantKey, c.wantValue, c.wantOK)
			}
		})
	}
}

func TestParseEcrImageRef(t *testing.T) {
	t.Run("valid reference", func(t *testing.T) {
		ref := "123456789012.dkr.ecr.us-east-1.amazonaws.com/myrepo:v1"
		accountID, region, repo, tag, err := ParseEcrImageRef(ref)
		if err != nil {
			t.Fatalf("ParseEcrImageRef(%q) unexpected error: %v", ref, err)
		}
		if accountID != "123456789012" {
			t.Errorf("accountID = %q, want %q", accountID, "123456789012")
		}
		if region != "us-east-1" {
			t.Errorf("region = %q, want %q", region, "us-east-1")
		}
		if repo != "myrepo" {
			t.Errorf("repository = %q, want %q", repo, "myrepo")
		}
		if tag != "v1" {
			t.Errorf("tag = %q, want %q", tag, "v1")
		}
	})

	t.Run("nested repository path", func(t *testing.T) {
		ref := "123456789012.dkr.ecr.eu-west-2.amazonaws.com/team/myrepo:latest"
		accountID, region, repo, tag, err := ParseEcrImageRef(ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if accountID != "123456789012" || region != "eu-west-2" || repo != "team/myrepo" || tag != "latest" {
			t.Errorf("got (%q, %q, %q, %q), want (123456789012, eu-west-2, team/myrepo, latest)",
				accountID, region, repo, tag)
		}
	})

	errCases := []struct {
		name       string
		ref        string
		wantSubstr string
	}{
		{"missing tag", "123456789012.dkr.ecr.us-east-1.amazonaws.com/myrepo", "must be domain/repository:tag"},
		{"missing repository name", "somedomain:v1", "missing repository name"},
		{"too few domain parts", "a.b.c/myrepo:v1", "too few parts"},
		{"empty account id", ".dkr.ecr.us-east-1.amazonaws.com/myrepo:v1", "missing required ECR parameters"},
	}
	for _, c := range errCases {
		t.Run(c.name, func(t *testing.T) {
			_, _, _, _, err := ParseEcrImageRef(c.ref)
			if err == nil {
				t.Fatalf("ParseEcrImageRef(%q) expected error, got nil", c.ref)
			}
			if !strings.Contains(err.Error(), c.wantSubstr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), c.wantSubstr)
			}
		})
	}
}

func TestParseGhcrImageRef(t *testing.T) {
	t.Run("full reference with tag", func(t *testing.T) {
		ns, repo, tag, err := ParseGhcrImageRef("ghcr.io/clouddrove/smurf:v2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ns != "clouddrove" || repo != "smurf" || tag != "v2" {
			t.Errorf("got (%q, %q, %q), want (clouddrove, smurf, v2)", ns, repo, tag)
		}
	})

	t.Run("no tag defaults to latest", func(t *testing.T) {
		ns, repo, tag, err := ParseGhcrImageRef("ghcr.io/clouddrove/smurf")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ns != "clouddrove" || repo != "smurf" || tag != "latest" {
			t.Errorf("got (%q, %q, %q), want (clouddrove, smurf, latest)", ns, repo, tag)
		}
	})

	t.Run("nested repository with tag", func(t *testing.T) {
		ns, repo, tag, err := ParseGhcrImageRef("ghcr.io/clouddrove/team/smurf:v3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ns != "clouddrove" || repo != "team/smurf" || tag != "v3" {
			t.Errorf("got (%q, %q, %q), want (clouddrove, team/smurf, v3)", ns, repo, tag)
		}
	})

	errCases := []struct {
		name       string
		ref        string
		wantSubstr string
	}{
		{"ghcr prefix but too few parts", "ghcr.io/onlyone", "invalid GHCR image reference"},
		{"not a ghcr reference", "docker.io/library/nginx:1.25", "not a GHCR image reference"},
		{"plain local image", "myimage:latest", "not a GHCR image reference"},
	}
	for _, c := range errCases {
		t.Run(c.name, func(t *testing.T) {
			_, _, _, err := ParseGhcrImageRef(c.ref)
			if err == nil {
				t.Fatalf("ParseGhcrImageRef(%q) expected error, got nil", c.ref)
			}
			if !strings.Contains(err.Error(), c.wantSubstr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), c.wantSubstr)
			}
		})
	}
}
