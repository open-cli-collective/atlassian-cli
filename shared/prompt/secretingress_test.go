package prompt

import (
	"errors"
	"io"
	"strings"
	"testing"
)

const ingressSentinel = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAingressSentinel"

func TestReadSecretFromIngress_StdinHappy(t *testing.T) {
	t.Parallel()
	got, err := ReadSecretFromIngress(strings.NewReader("  "+ingressSentinel+"\n  "), true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ingressSentinel {
		t.Fatalf("got %q, want %q", got, ingressSentinel)
	}
}

func TestReadSecretFromIngress_EnvHappy(t *testing.T) {
	t.Setenv("INGRESS_TEST_VAR", "  "+ingressSentinel+"  ")
	got, err := ReadSecretFromIngress(nil, false, "INGRESS_TEST_VAR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ingressSentinel {
		t.Fatalf("got %q, want %q", got, ingressSentinel)
	}
}

func TestReadSecretFromIngress_BothRejected(t *testing.T) {
	t.Setenv("INGRESS_TEST_VAR", "x")
	_, err := ReadSecretFromIngress(strings.NewReader("y"), true, "INGRESS_TEST_VAR")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error must mention mutual exclusion, got %v", err)
	}
}

func TestReadSecretFromIngress_NeitherReturnsEmptyNilErr(t *testing.T) {
	t.Parallel()
	got, err := ReadSecretFromIngress(nil, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty pass-through, got %q", got)
	}
}

func TestReadSecretFromIngress_StdinWhitespaceRejected(t *testing.T) {
	t.Parallel()
	_, err := ReadSecretFromIngress(strings.NewReader("   \n  "), true, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error must mention empty, got %v", err)
	}
}

func TestReadSecretFromIngress_EnvUnsetRejected(t *testing.T) {
	t.Setenv("INGRESS_UNSET_VAR", "")
	_, err := ReadSecretFromIngress(nil, false, "INGRESS_UNSET_VAR")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "INGRESS_UNSET_VAR") {
		t.Fatalf("error must name the var, got %v", err)
	}
}

func TestReadSecretFromIngress_StdinNilRejected(t *testing.T) {
	t.Parallel()
	_, err := ReadSecretFromIngress(nil, true, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestReadSecretFromIngress_StdinReadErrSurfaces — io.ReadAll errors
// (non-EOF) must propagate without leaking the partial value.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestReadSecretFromIngress_StdinReadErrSurfaces(t *testing.T) {
	t.Parallel()
	_, err := ReadSecretFromIngress(errReader{}, true, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read API token") {
		t.Fatalf("error must wrap the read context, got %v", err)
	}
}

func TestReadSecretFromIngress_NeverEmitsSecret_AllPaths(t *testing.T) {
	cases := []struct {
		name string
		seed func(t *testing.T)
		opts struct {
			stdin    io.Reader
			useStdin bool
			fromEnv  string
		}
	}{
		{
			name: "happy-stdin",
			opts: struct {
				stdin    io.Reader
				useStdin bool
				fromEnv  string
			}{stdin: strings.NewReader(ingressSentinel), useStdin: true},
		},
		{
			name: "happy-env",
			seed: func(t *testing.T) { t.Setenv("INGRESS_LEAK_VAR", ingressSentinel) },
			opts: struct {
				stdin    io.Reader
				useStdin bool
				fromEnv  string
			}{fromEnv: "INGRESS_LEAK_VAR"},
		},
		{
			name: "fail-both-sources",
			seed: func(t *testing.T) { t.Setenv("INGRESS_LEAK_VAR", ingressSentinel) },
			opts: struct {
				stdin    io.Reader
				useStdin bool
				fromEnv  string
			}{stdin: strings.NewReader(ingressSentinel), useStdin: true, fromEnv: "INGRESS_LEAK_VAR"},
		},
		{
			name: "fail-empty-stdin",
			opts: struct {
				stdin    io.Reader
				useStdin bool
				fromEnv  string
			}{stdin: strings.NewReader("    "), useStdin: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.seed != nil {
				tc.seed(t)
			}
			got, err := ReadSecretFromIngress(tc.opts.stdin, tc.opts.useStdin, tc.opts.fromEnv)
			// On failure paths, returned err must not leak the secret.
			if err != nil && strings.Contains(err.Error(), ingressSentinel) {
				t.Fatalf("error leaked sentinel: %v", err)
			}
			// On the happy path got holds the secret intentionally; no leak
			// assertion against got is meaningful there.
			_ = got
		})
	}
}
