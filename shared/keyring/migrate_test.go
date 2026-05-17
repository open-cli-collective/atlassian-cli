package keyring

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credstore"
)

// planWrites is the pure §1.8 resolver. The matrix below is the explicit
// list from the plan: idempotent equals, fail-loud differs, overwrite.
func TestPlanWrites_Matrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		want      map[string]string
		current   map[string]string
		overwrite bool
		wantWrite map[string]string
		wantConf  []string
	}{
		{
			name:      "empty",
			want:      map[string]string{},
			current:   map[string]string{},
			wantWrite: map[string]string{},
		},
		{
			name:      "fresh write (absent in keyring)",
			want:      map[string]string{KeyAPIToken: "t"},
			current:   map[string]string{},
			wantWrite: map[string]string{KeyAPIToken: "t"},
		},
		{
			name:      "equal value is idempotent no-op",
			want:      map[string]string{KeyAPIToken: "t"},
			current:   map[string]string{KeyAPIToken: "t"},
			wantWrite: map[string]string{},
		},
		{
			name:     "different value is a fail-loud conflict",
			want:     map[string]string{KeyAPIToken: "new"},
			current:  map[string]string{KeyAPIToken: "old"},
			wantConf: []string{KeyAPIToken},
		},
		{
			name:      "overwrite forces a differing value",
			want:      map[string]string{KeyAPIToken: "new"},
			current:   map[string]string{KeyAPIToken: "old"},
			overwrite: true,
			wantWrite: map[string]string{KeyAPIToken: "new"},
		},
		{
			name:      "multi-key: one fresh, one equal, one conflict",
			want:      map[string]string{KeyAPIToken: "a", KeyCFLAPIToken: "b", KeyJTKAPIToken: "c"},
			current:   map[string]string{KeyCFLAPIToken: "b", KeyJTKAPIToken: "x"},
			wantWrite: map[string]string{KeyAPIToken: "a"},
			wantConf:  []string{KeyJTKAPIToken},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotWrite, gotConf := planWrites(tc.want, tc.current, tc.overwrite)
			if len(gotWrite) != len(tc.wantWrite) {
				t.Fatalf("toWrite=%v want %v", gotWrite, tc.wantWrite)
			}
			for k, v := range tc.wantWrite {
				if gotWrite[k] != v {
					t.Fatalf("toWrite[%s]=%q want %q", k, gotWrite[k], v)
				}
			}
			if strings.Join(gotConf, ",") != strings.Join(tc.wantConf, ",") {
				t.Fatalf("conflicts=%v want %v", gotConf, tc.wantConf)
			}
		})
	}
}

func lc(path, tok string) *credstore.LegacyCreds {
	return &credstore.LegacyCreds{Path: path, APIToken: tok}
}

// gatherEffective maps today's persisted precedence to per-scope keys.
func TestGatherEffective_Mapping(t *testing.T) {
	t.Parallel()

	t.Run("empty everywhere", func(t *testing.T) {
		t.Parallel()
		want, _, any := gatherEffective(&credstore.Store{}, nil, nil)
		if len(want) != 0 || any {
			t.Fatalf("want empty, anyPlaintext=false; got %v %v", want, any)
		}
	})

	t.Run("shared default only", func(t *testing.T) {
		t.Parallel()
		st := &credstore.Store{Default: credstore.Section{APIToken: "D"}}
		want, _, any := gatherEffective(st, nil, nil)
		if want[KeyAPIToken] != "D" || len(want) != 1 || !any {
			t.Fatalf("got %v any=%v", want, any)
		}
	})

	t.Run("cfl legacy file only -> cfl override (no default shadows it)", func(t *testing.T) {
		t.Parallel()
		want, _, any := gatherEffective(&credstore.Store{}, lc("/cfl", "C"), nil)
		if want[KeyCFLAPIToken] != "C" || !any {
			t.Fatalf("got %v any=%v", want, any)
		}
	})

	t.Run("jtk legacy file only -> jtk override", func(t *testing.T) {
		t.Parallel()
		want, _, _ := gatherEffective(&credstore.Store{}, nil, lc("/jtk", "J"))
		if want[KeyJTKAPIToken] != "J" {
			t.Fatalf("got %v", want)
		}
	})

	t.Run("shared Store.CFL override differing from default is a real override", func(t *testing.T) {
		t.Parallel()
		st := &credstore.Store{
			Default: credstore.Section{APIToken: "D"},
			CFL:     credstore.ToolSection{Section: credstore.Section{APIToken: "C"}},
		}
		want, _, _ := gatherEffective(st, nil, nil)
		if want[KeyAPIToken] != "D" || want[KeyCFLAPIToken] != "C" {
			t.Fatalf("got %v", want)
		}
	})

	t.Run("legacy cfl file shadowed by shared default is scrub-only", func(t *testing.T) {
		t.Parallel()
		st := &credstore.Store{Default: credstore.Section{APIToken: "D"}}
		want, _, any := gatherEffective(st, lc("/cfl", "shadowed"), nil)
		if _, ok := want[KeyCFLAPIToken]; ok {
			t.Fatalf("shadowed legacy must not be written: %v", want)
		}
		if want[KeyAPIToken] != "D" || !any {
			t.Fatalf("got %v any=%v (anyPlaintext must be true so the dead file is scrubbed)", want, any)
		}
	})

	t.Run("tool value equal to effective default is cleanup-only", func(t *testing.T) {
		t.Parallel()
		st := &credstore.Store{
			Default: credstore.Section{APIToken: "D"},
			CFL:     credstore.ToolSection{Section: credstore.Section{APIToken: "D"}},
		}
		want, _, _ := gatherEffective(st, nil, nil)
		if _, ok := want[KeyCFLAPIToken]; ok {
			t.Fatalf("equal-to-default must not pin an override: %v", want)
		}
	})
}

// conflictError must name keys + non-secret locations + ref, never the
// secret value (§1.12).
func TestConflictError_NoSecretLeak(t *testing.T) {
	t.Parallel()
	err := conflictError(
		[]string{KeyAPIToken},
		//nolint:gosec // G101: a non-secret location label, not a credential
		map[string]string{KeyAPIToken: "shared config default"},
		Ref,
	)
	msg := err.Error()
	if strings.Contains(msg, "SUPER-SECRET") {
		t.Fatalf("conflict error leaked a value: %s", msg)
	}
	if !strings.Contains(msg, KeyAPIToken) || !strings.Contains(msg, Ref) {
		t.Fatalf("conflict error missing key/ref: %s", msg)
	}
}
