package credstore

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func nc(label, section, path string, c ConnProfile) NamedConn {
	return NamedConn{Label: label, Section: section, Path: path, Conn: c}
}

// §2.2/MON-5328 detector semantics: field-wise union, conflict iff a
// canonical field has ≥2 distinct non-empty values; usable-basic
// materializes implicit basic; fragmentary sources give no auth opinion;
// partial default is compatible with a fuller source.
func TestDetectConnDivergence(t *testing.T) {
	t.Parallel()
	basic := "basic"
	bearer := "bearer"

	t.Run("no sources -> empty chosen, basic default", func(t *testing.T) {
		t.Parallel()
		got, conf := DetectConnDivergence(nil)
		testutil.Equal(t, 0, len(conf))
		testutil.Equal(t, "", got.URL)
		testutil.Equal(t, basic, got.AuthMethod)
	})

	t.Run("all equal -> no conflict", func(t *testing.T) {
		t.Parallel()
		c := ConnProfile{URL: "https://acme.atlassian.net", Email: "u@e", AuthMethod: basic}
		got, conf := DetectConnDivergence([]NamedConn{
			nc("shared config", "default", "/c.yml", c),
			nc("legacy cfl config", "", "/cfl.yml", c),
		})
		testutil.Equal(t, 0, len(conf))
		testutil.Equal(t, "https://acme.atlassian.net", got.URL)
	})

	t.Run("canonically equal but textually different -> no conflict", func(t *testing.T) {
		t.Parallel()
		got, conf := DetectConnDivergence([]NamedConn{
			nc("a", "default", "/c.yml", ConnProfile{URL: "https://acme.atlassian.net/wiki/", Email: "u@e"}),
			nc("b", "", "/cfl.yml", ConnProfile{URL: "https://acme.atlassian.net", Email: " u@e "}),
		})
		testutil.Equal(t, 0, len(conf))
		testutil.Equal(t, "https://acme.atlassian.net", got.URL)
		testutil.Equal(t, "u@e", got.Email)
	})

	t.Run("partial default compatible with fuller legacy -> union, no conflict", func(t *testing.T) {
		t.Parallel()
		got, conf := DetectConnDivergence([]NamedConn{
			nc("shared config", "default", "/c.yml", ConnProfile{URL: "https://acme.atlassian.net"}),
			nc("legacy cfl config", "", "/cfl.yml", ConnProfile{URL: "https://acme.atlassian.net", Email: "u@e"}),
		})
		testutil.Equal(t, 0, len(conf))
		testutil.Equal(t, "https://acme.atlassian.net", got.URL)
		testutil.Equal(t, "u@e", got.Email) // unioned in from the fuller source
	})

	t.Run("divergent url -> conflict, no value leak", func(t *testing.T) {
		t.Parallel()
		got, conf := DetectConnDivergence([]NamedConn{
			nc("shared config", "default", "/c.yml", ConnProfile{URL: "https://a.atlassian.net", Email: "u@e"}),
			nc("legacy cfl config", "", "/cfl.yml", ConnProfile{URL: "https://b.atlassian.net", Email: "u@e"}),
		})
		testutil.Equal(t, ConnProfile{}, got)
		testutil.Equal(t, 1, len(conf))
		testutil.Equal(t, "url", conf[0].Field)
		joined := strings.Join(conf[0].Sources, " ")
		if strings.Contains(joined, "a.atlassian.net") || strings.Contains(joined, "b.atlassian.net") {
			t.Fatalf("conflict leaked a value: %v", conf[0].Sources)
		}
		if !strings.Contains(joined, "shared config default.url (/c.yml)") ||
			!strings.Contains(joined, "legacy cfl config url (/cfl.yml)") {
			t.Fatalf("conflict must name source-label section.field (path): %v", conf[0].Sources)
		}
	})

	t.Run("implicit-basic default vs explicit-bearer legacy -> conflict", func(t *testing.T) {
		t.Parallel()
		_, conf := DetectConnDivergence([]NamedConn{
			nc("shared config", "default", "/c.yml", ConnProfile{URL: "https://acme.atlassian.net", Email: "u@e"}),
			nc("legacy jtk config", "", "/jtk.json", ConnProfile{
				URL: "https://acme.atlassian.net", CloudID: "cid", AuthMethod: bearer,
			}),
		})
		var gotAuth bool
		for _, c := range conf {
			if c.Field == "auth_method" {
				gotAuth = true
			}
		}
		if !gotAuth {
			t.Fatalf("implicit-basic default must conflict with explicit bearer; conflicts=%v", conf)
		}
	})

	t.Run("fragmentary source (cloud_id only) gives no auth opinion", func(t *testing.T) {
		t.Parallel()
		got, conf := DetectConnDivergence([]NamedConn{
			nc("shared config", "default", "/c.yml", ConnProfile{URL: "https://acme.atlassian.net", Email: "u@e"}),
			nc("legacy cfl config", "", "/cfl.yml", ConnProfile{CloudID: "cid"}),
		})
		testutil.Equal(t, 0, len(conf)) // cloud_id-only contributes no auth_method, no conflict
		testutil.Equal(t, basic, got.AuthMethod)
		testutil.Equal(t, "cid", got.CloudID)
	})

	t.Run("explicit basic + implicit basic agree -> no conflict", func(t *testing.T) {
		t.Parallel()
		// usableBasic must not only PREVENT silent bearer-union; an
		// explicit `basic` and an implicit-basic (url+email, no auth)
		// source must canonically AGREE (the positive direction).
		got, conf := DetectConnDivergence([]NamedConn{
			nc("shared config", "default", "/c.yml", ConnProfile{URL: "https://acme.atlassian.net", Email: "u@e", AuthMethod: "basic"}),
			nc("legacy cfl config", "", "/cfl.yml", ConnProfile{URL: "https://acme.atlassian.net", Email: "u@e"}),
		})
		testutil.Equal(t, 0, len(conf))
		testutil.Equal(t, basic, got.AuthMethod)
	})

	t.Run("all-empty source ignored", func(t *testing.T) {
		t.Parallel()
		got, conf := DetectConnDivergence([]NamedConn{
			nc("shared config", "default", "/c.yml", ConnProfile{URL: "https://acme.atlassian.net", Email: "u@e"}),
			nc("shared config", "cfl", "/c.yml", ConnProfile{}),
		})
		testutil.Equal(t, 0, len(conf))
		testutil.Equal(t, "https://acme.atlassian.net", got.URL)
	})
}
