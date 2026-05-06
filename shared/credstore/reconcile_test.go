package credstore

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestSectionsEqual(t *testing.T) {
	t.Parallel()
	a := Section{URL: "https://acme.atlassian.net", Email: "u@e", APIToken: "t"}
	b := Section{URL: "https://acme.atlassian.net/wiki", Email: "u@e", APIToken: "t"}
	if !SectionsEqual(a, b) {
		t.Fatalf("SectionsEqual should ignore /wiki suffix")
	}

	c := Section{URL: "https://acme.atlassian.net", Email: "u@e", APIToken: "DIFFERENT"}
	if SectionsEqual(a, c) {
		t.Fatalf("SectionsEqual should detect different tokens")
	}
}

func TestMaskToken(t *testing.T) {
	t.Parallel()
	testutil.Equal(t, "", MaskToken(""))
	testutil.Equal(t, "***", MaskToken("short"))
	testutil.Equal(t, "ATAT...wxyz", MaskToken("ATATTabcdefghijwxyz"))
}

func TestFormatSection_ShowsAllFields(t *testing.T) {
	t.Parallel()
	s := Section{
		URL:        "https://acme.atlassian.net",
		Email:      "rian@example.com",
		APIToken:   "ATATTabcdefghijwxyz",
		AuthMethod: "bearer",
		CloudID:    "cloud-123",
	}
	got := FormatSection("cfl", s)
	for _, want := range []string{
		"cfl:",
		"https://acme.atlassian.net",
		"rian@example.com",
		"ATAT...wxyz",
		"method: bearer",
		"cloud_id: cloud-123",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatSection missing %q in:\n%s", want, got)
		}
	}
}

func TestFormatSection_DefaultsToBasic(t *testing.T) {
	t.Parallel()
	s := Section{URL: "https://x", Email: "u@e", APIToken: "t"}
	got := FormatSection("default", s)
	if !strings.Contains(got, "method: basic") {
		t.Errorf("expected default to render as basic; got:\n%s", got)
	}
}

func TestFormatSection_MissingFieldsRenderUnset(t *testing.T) {
	t.Parallel()
	got := FormatSection("default", Section{})
	for _, want := range []string{"url:    (unset)", "email:  (unset)", "token:  (unset)"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in:\n%s", want, got)
		}
	}
}
