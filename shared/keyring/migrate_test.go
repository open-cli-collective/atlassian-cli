package keyring

import (
	"errors"
	"sort"
	"strings"
	"testing"
)

// src builds the {value -> location-set} map planMigration consumes.
func src(pairs ...[2]string) map[string]map[string]struct{} {
	m := map[string]map[string]struct{}{}
	for _, p := range pairs {
		val, loc := p[0], p[1]
		if m[val] == nil {
			m[val] = map[string]struct{}{}
		}
		m[val][loc] = struct{}{}
	}
	return m
}

func TestPlanMigration(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		curAPI    string
		srcLoc    map[string]map[string]struct{}
		overwrite bool
		wantWrite bool
		wantVal   string
		wantConf  bool
	}{
		{name: "nothing to migrate", srcLoc: src()},
		{name: "single source, api_token absent -> write",
			srcLoc: src([2]string{"T", "legacy cfl"}), wantWrite: true, wantVal: "T"},
		{name: "single source equals existing api_token -> no-op (cleanup only)",
			curAPI: "T", srcLoc: src([2]string{"T", "legacy cfl"})},
		{name: "single source differs from api_token -> conflict",
			curAPI: "OLD", srcLoc: src([2]string{"NEW", "legacy cfl"}), wantConf: true},
		{name: "single source differs, --overwrite resolves",
			curAPI: "OLD", srcLoc: src([2]string{"NEW", "legacy cfl"}), overwrite: true,
			wantWrite: true, wantVal: "NEW"},
		{name: "two distinct sources -> hard conflict",
			srcLoc:   src([2]string{"A", "legacy cfl"}, [2]string{"B", "legacy jtk"}),
			wantConf: true},
		{name: "two distinct sources, --overwrite still conflicts (never picks)",
			srcLoc:    src([2]string{"A", "legacy cfl"}, [2]string{"B", "keyring deprecated key cfl_api_token"}),
			overwrite: true, wantConf: true},
		{name: "same value from many sources collapses to one -> write",
			srcLoc:    src([2]string{"S", "shared default"}, [2]string{"S", "legacy jtk"}),
			wantWrite: true, wantVal: "S"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan, conflicts := planMigration(tc.curAPI, tc.srcLoc, tc.overwrite)
			if (len(conflicts) > 0) != tc.wantConf {
				t.Fatalf("conflict=%v, want %v (locs=%v)", len(conflicts) > 0, tc.wantConf, conflicts)
			}
			if plan.write != tc.wantWrite {
				t.Fatalf("write=%v, want %v", plan.write, tc.wantWrite)
			}
			if plan.write && plan.value != tc.wantVal {
				t.Fatalf("value=%q, want %q", plan.value, tc.wantVal)
			}
			// Plan-then-apply invariant: a conflict must never also
			// request a write (no mutation when the picture is ambiguous).
			if tc.wantConf && plan.write {
				t.Fatal("conflict must not also request a write")
			}
		})
	}
}

// conflictError names every source location and never the secret value.
func TestConflictError_NoSecretLeak(t *testing.T) {
	t.Parallel()
	const secretVal = "TOK-do-not-leak" //nolint:gosec // G101: test fixture string, not a real credential
	locs := []string{"legacy cfl config (/x/cfl.yml)", "keyring deprecated key jtk_api_token (atlassian-cli/default)"}
	sort.Strings(locs)
	err := conflictError(locs, "atlassian-cli/default")
	if !errors.Is(err, ErrMigrationConflict) {
		t.Fatalf("want ErrMigrationConflict, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "legacy cfl config (/x/cfl.yml)") || !strings.Contains(msg, "jtk_api_token") {
		t.Fatalf("conflict error must name every source location: %q", msg)
	}
	if strings.Contains(msg, secretVal) {
		t.Fatal("conflict error leaked the secret value")
	}
}
