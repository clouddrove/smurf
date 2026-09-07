package docker

import "testing"

// These three functions decide which registry an image is pushed to and under
// what credentials. They were inlined in PushImageToECR, which cannot run
// without AWS, so nothing exercised them. Both failure modes are quiet: the
// wrong host pushes to the wrong registry, and a truncated password reads as a
// permissions problem rather than a parsing bug.

func TestSplitECRCredentials(t *testing.T) {
	user, pass, err := splitECRCredentials("AWS:secretpassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != "AWS" {
		t.Errorf("username = %q, want AWS", user)
	}
	if pass != "secretpassword" {
		t.Errorf("password = %q", pass)
	}
}

// The reason SplitN is limited to two: ECR passwords are long base64-ish
// strings that can contain a colon. Splitting on every colon would truncate
// the password and fail authentication with an error that looks like a
// permissions problem.
func TestSplitECRCredentials_PasswordMayContainColons(t *testing.T) {
	user, pass, err := splitECRCredentials("AWS:abc:def:ghi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != "AWS" {
		t.Errorf("username = %q, want AWS", user)
	}
	if pass != "abc:def:ghi" {
		t.Errorf("password = %q, want the whole remainder including colons", pass)
	}
}

func TestSplitECRCredentials_RejectsMalformed(t *testing.T) {
	for _, token := range []string{"", "nocolonhere", "AWS"} {
		if _, _, err := splitECRCredentials(token); err == nil {
			t.Errorf("splitECRCredentials(%q) should have failed", token)
		}
	}
}

// An empty password is still a well-formed token; refusing it here would turn
// a server-side problem into a confusing client-side one.
func TestSplitECRCredentials_EmptyPasswordIsWellFormed(t *testing.T) {
	user, pass, err := splitECRCredentials("AWS:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != "AWS" || pass != "" {
		t.Errorf("got %q/%q, want AWS and an empty password", user, pass)
	}
}

// An image reference cannot carry a scheme, so the endpoint AWS returns has to
// be reduced to a host first.
func TestECRRegistryHost(t *testing.T) {
	cases := map[string]string{
		"https://123456789012.dkr.ecr.us-east-1.amazonaws.com":  "123456789012.dkr.ecr.us-east-1.amazonaws.com",
		"http://123456789012.dkr.ecr.us-east-1.amazonaws.com":   "123456789012.dkr.ecr.us-east-1.amazonaws.com",
		"123456789012.dkr.ecr.us-east-1.amazonaws.com":          "123456789012.dkr.ecr.us-east-1.amazonaws.com",
		"https://123456789012.dkr.ecr.eu-west-2.amazonaws.com/": "123456789012.dkr.ecr.eu-west-2.amazonaws.com",
		"": "",
	}
	for in, want := range cases {
		if got := ecrRegistryHost(in); got != want {
			t.Errorf("ecrRegistryHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestECRImageRef(t *testing.T) {
	got := ecrImageRef("123456789012.dkr.ecr.us-east-1.amazonaws.com", "my-app", "v1.2.3")
	want := "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app:v1.2.3"
	if got != want {
		t.Errorf("ecrImageRef() = %q, want %q", got, want)
	}
}

// configs.ParseImage returns an empty tag for an untagged image. Without a
// default the reference would end in a bare colon, which the daemon rejects
// with a message that says nothing about the missing tag.
func TestECRImageRef_EmptyTagBecomesLatest(t *testing.T) {
	got := ecrImageRef("registry.example.com", "my-app", "")
	want := "registry.example.com/my-app:latest"
	if got != want {
		t.Errorf("ecrImageRef() with no tag = %q, want %q", got, want)
	}
}

// The whole point is the round trip: what AWS hands back must produce the
// reference the image is actually pushed to.
func TestECRPushTarget_EndToEnd(t *testing.T) {
	const (
		proxyEndpoint = "https://123456789012.dkr.ecr.ap-south-1.amazonaws.com"
		// Not a credential: a fabricated token shaped like what ECR returns,
		// chosen so its password half contains a colon.
		decodedToken = "AWS:eyJwYXlsb2FkIjoiYWJj:ZGVm" // #nosec G101
	)

	user, pass, err := splitECRCredentials(decodedToken)
	if err != nil {
		t.Fatalf("credential split failed: %v", err)
	}
	ref := ecrImageRef(ecrRegistryHost(proxyEndpoint), "team/api", "2026.09.1")

	if user != "AWS" {
		t.Errorf("username = %q", user)
	}
	if pass != "eyJwYXlsb2FkIjoiYWJj:ZGVm" { // #nosec G101 -- fabricated fixture, not a credential
		t.Errorf("password = %q, want the colon preserved", pass)
	}
	want := "123456789012.dkr.ecr.ap-south-1.amazonaws.com/team/api:2026.09.1"
	if ref != want {
		t.Errorf("push target = %q, want %q", ref, want)
	}
}
