package setcredential

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

const sentinel = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAjtkSent"

// runCmd executes `jtk set-credential <args...>` against a fresh root +
// the setcredential subcommand, returning stdout, stderr, the captured
// exit error, and the root.Options used. The token never comes through
// argv; stdin comes from opts.Stdin so the command never sees the value
// in args.
func runCmd(t *testing.T, stdin string, args ...string) (string, string, *root.Options, error) {
	t.Helper()
	opts := &root.Options{
		Stdin:  strings.NewReader(stdin),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	rootCmd := &cobra.Command{Use: "jtk", SilenceErrors: true, SilenceUsage: true}
	Register(rootCmd, opts)
	rootCmd.SetArgs(append([]string{"set-credential"}, args...))
	err := rootCmd.Execute()
	return opts.Stdout.(*bytes.Buffer).String(), opts.Stderr.(*bytes.Buffer).String(), opts, err
}

func parseEnvelope(t *testing.T, line string) keyring.SetCredentialResult {
	t.Helper()
	var env keyring.SetCredentialResult
	line = strings.TrimSpace(line)
	if line == "" {
		t.Fatal("envelope: empty stdout")
	}
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		t.Fatalf("envelope: unmarshal %q: %v", line, err)
	}
	return env
}

func TestSetCredential_StdinExplicit_Success(t *testing.T) {
	credtest.Hermetic(t)
	_, stderr, _, err := runCmd(t, sentinel+"\n",
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin")
	testutil.RequireNoError(t, err)
	if !strings.HasPrefix(stderr, "wrote api_token to "+keyring.Ref+" via ") {
		t.Fatalf("stderr line shape mismatch: %q", stderr)
	}

	s, oerr := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, oerr)
	defer func() { _ = s.Close() }()
	got, ok, terr := s.Token()
	testutil.RequireNoError(t, terr)
	testutil.True(t, ok)
	testutil.Equal(t, sentinel, got)
}

func TestSetCredential_FromEnvSuccess(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("JTK_TEST_TOKEN_VAR", sentinel)

	_, _, _, err := runCmd(t, "",
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--from-env", "JTK_TEST_TOKEN_VAR")
	testutil.RequireNoError(t, err)

	s, oerr := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, oerr)
	defer func() { _ = s.Close() }()
	got, ok, _ := s.Token()
	testutil.True(t, ok)
	testutil.Equal(t, sentinel, got)
}

func TestSetCredential_KeyOmitted_PreKeyringFails(t *testing.T) {
	credtest.Hermetic(t)
	_, _, _, err := runCmd(t, sentinel,
		"--ref", keyring.Ref, "--stdin")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--key") || !strings.Contains(err.Error(), "api_token") {
		t.Fatalf("error must hint at --key api_token, got %q", err)
	}
}

func TestSetCredential_RefOmittedNoConfig_PreKeyringFails(t *testing.T) {
	credtest.Hermetic(t) // fresh install — no config.yml
	_, _, _, err := runCmd(t, sentinel,
		"--key", keyring.KeyAPIToken, "--stdin")
	if err == nil {
		t.Fatal("expected error on fresh install with no --ref")
	}
	if !strings.Contains(err.Error(), "--ref") || !strings.Contains(err.Error(), keyring.Ref) {
		t.Fatalf("error must hint at --ref %s, got %q", keyring.Ref, err)
	}
}

func TestSetCredential_RefOmittedConfigExists_DefaultsToCanonical(t *testing.T) {
	credtest.Hermetic(t)
	seedConfig(t, "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\n")

	_, _, _, err := runCmd(t, sentinel,
		"--key", keyring.KeyAPIToken, "--stdin")
	testutil.RequireNoError(t, err)

	s, oerr := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, oerr)
	defer func() { _ = s.Close() }()
	got, ok, _ := s.Token()
	testutil.True(t, ok)
	testutil.Equal(t, sentinel, got)
}

func TestSetCredential_RefOmittedCorruptConfig_PreKeyringFails(t *testing.T) {
	credtest.Hermetic(t)
	seedConfig(t, "[unclosed_array: yes\n")

	_, _, _, err := runCmd(t, sentinel,
		"--key", keyring.KeyAPIToken, "--stdin")
	if err == nil {
		t.Fatal("expected error from corrupt config probe")
	}
	if !errors.Is(err, credstore.ErrCorruptStore) {
		t.Fatalf("error must wrap ErrCorruptStore, got %v", err)
	}
}

func TestSetCredential_NoSource_Fails(t *testing.T) {
	credtest.Hermetic(t)
	_, _, _, err := runCmd(t, "",
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--stdin") || !strings.Contains(err.Error(), "--from-env") {
		t.Fatalf("error must mention both source flags, got %q", err)
	}
}

func TestSetCredential_BothSources_Fails(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("BOTHSRC_VAR", "x")
	_, _, _, err := runCmd(t, sentinel,
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken,
		"--stdin", "--from-env", "BOTHSRC_VAR")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error must mention mutual exclusion, got %q", err)
	}
}

func TestSetCredential_ExistingNoOverwrite_Fails(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "previous-token")

	_, _, _, err := runCmd(t, sentinel,
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin")
	if err == nil {
		t.Fatal("expected error on existing entry without --overwrite")
	}
	if !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("error must mention --overwrite, got %q", err)
	}

	s, oerr := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, oerr)
	defer func() { _ = s.Close() }()
	got, ok, _ := s.Token()
	testutil.True(t, ok)
	testutil.Equal(t, "previous-token", got)
}

func TestSetCredential_ExistingWithOverwrite_Succeeds(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "previous-token")

	_, _, _, err := runCmd(t, sentinel,
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin", "--overwrite")
	testutil.RequireNoError(t, err)

	s, oerr := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, oerr)
	defer func() { _ = s.Close() }()
	got, ok, _ := s.Token()
	testutil.True(t, ok)
	testutil.Equal(t, sentinel, got)
}

func TestSetCredential_JSONSuccess(t *testing.T) {
	credtest.Hermetic(t)
	stdout, stderr, _, err := runCmd(t, sentinel,
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin", "--json")
	testutil.RequireNoError(t, err)
	if stderr != "" {
		t.Fatalf("stderr must be empty under --json: %q", stderr)
	}
	env := parseEnvelope(t, stdout)
	testutil.Equal(t, keyring.Ref, env.Ref)
	testutil.Equal(t, keyring.KeyAPIToken, env.Key)
	testutil.True(t, env.Written)
	if env.Backend == "" {
		t.Fatal("Backend must be populated on success")
	}
	if env.Error != "" {
		t.Fatalf("Error must be empty on success, got %q", env.Error)
	}
}

// TestSetCredential_JSONFailure_PreKeyring_EmptyToken — empty-stdin
// rejection is technically a "pre-keyring" failure (we never Open() with
// no source). Asserts the envelope contract: backend:"", written:false,
// error populated, stderr empty.
func TestSetCredential_JSONFailure_PreKeyring_EmptyToken(t *testing.T) {
	credtest.Hermetic(t)
	stdout, stderr, _, err := runCmd(t, "", "--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin", "--json")
	if err == nil {
		t.Fatal("expected pre-keyring error")
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty under --json failure: %q", stderr)
	}
	env := parseEnvelope(t, stdout)
	testutil.Equal(t, "", env.Backend)
	testutil.False(t, env.Written)
	if env.Error == "" {
		t.Fatal("envelope error field must be populated on failure")
	}
}

// TestSetCredential_JSONFailure_PreKeyring_MissingRef — ref-validation
// failure path under --json. The envelope MUST carry backend:"" since
// the keyring was never opened. Pairs with the EmptyToken case so both
// pre-keyring routes (validation vs source-read) are covered.
func TestSetCredential_JSONFailure_PreKeyring_MissingRef(t *testing.T) {
	credtest.Hermetic(t) // no config — --ref omission is the failure
	stdout, stderr, _, err := runCmd(t, sentinel, "--key", keyring.KeyAPIToken, "--stdin", "--json")
	if err == nil {
		t.Fatal("expected pre-keyring error on missing --ref")
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty under --json failure: %q", stderr)
	}
	env := parseEnvelope(t, stdout)
	testutil.Equal(t, "", env.Backend)
	testutil.False(t, env.Written)
	if !strings.Contains(env.Error, "--ref") {
		t.Fatalf("envelope error must hint at --ref, got %q", env.Error)
	}
}

func TestSetCredential_JSONFailure_PostKeyring(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "earlier-token")

	stdout, stderr, _, err := runCmd(t, sentinel,
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin", "--json")
	if err == nil {
		t.Fatal("expected post-keyring error")
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty under --json failure: %q", stderr)
	}
	env := parseEnvelope(t, stdout)
	if env.Backend == "" {
		t.Fatal("Backend must be populated on post-keyring failure")
	}
	testutil.False(t, env.Written)
	if !strings.Contains(env.Error, "--overwrite") {
		t.Fatalf("envelope error must mention --overwrite, got %q", env.Error)
	}
}

// TestSetCredential_JSONMigratingInvocation_StderrEmpty — when the §1.8
// migration runs during set-credential's Open(), the human notice MUST
// NOT leak to stderr under --json. Mirrors the library-layer drain test
// but at the cobra wrapper so a regression in the wrapper or its main.go
// wiring is caught here.
func TestSetCredential_JSONMigratingInvocation_StderrEmpty(t *testing.T) {
	keyring.ResetMigrationNotice()
	t.Cleanup(keyring.ResetMigrationNotice)
	credtest.Hermetic(t)
	credtest.SeedDeprecatedKey(t, "jtk_api_token", "legacy-token-value")

	stdout, stderr, _, err := runCmd(t, sentinel,
		"--ref", keyring.Ref, "--key", keyring.KeyAPIToken,
		"--stdin", "--json", "--overwrite")
	testutil.RequireNoError(t, err)
	if stderr != "" {
		t.Fatalf("stderr must be empty under --json even during migration: %q", stderr)
	}
	env := parseEnvelope(t, stdout)
	testutil.True(t, env.Written)
}

func TestSetCredential_NonCanonicalRef_Fails(t *testing.T) {
	credtest.Hermetic(t)
	_, _, _, err := runCmd(t, sentinel,
		"--ref", "bogus/ref", "--key", keyring.KeyAPIToken, "--stdin")
	if err == nil {
		t.Fatal("expected error on non-canonical ref")
	}
	if !strings.Contains(err.Error(), "bogus/ref") || !strings.Contains(err.Error(), keyring.Ref) {
		t.Fatalf("error must name bad ref and only valid ref, got %q", err)
	}
}

func TestSetCredential_NonCanonicalKey_Fails(t *testing.T) {
	credtest.Hermetic(t)
	_, _, _, err := runCmd(t, sentinel,
		"--ref", keyring.Ref, "--key", "bogus_key", "--stdin")
	if err == nil {
		t.Fatal("expected error on non-canonical key")
	}
	if !strings.Contains(err.Error(), "bogus_key") || !strings.Contains(err.Error(), keyring.KeyAPIToken) {
		t.Fatalf("error must name bad key and only valid key, got %q", err)
	}
}

func TestSetCredential_NeverEmitsSecret_AcrossAllPaths(t *testing.T) {
	cases := []struct {
		name string
		seed func(t *testing.T)
		args []string
	}{
		{
			name: "happy",
			args: []string{"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin"},
		},
		{
			name: "happy-json",
			args: []string{"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin", "--json"},
		},
		{
			name: "fail-no-source",
			args: []string{"--ref", keyring.Ref, "--key", keyring.KeyAPIToken},
		},
		{
			name: "fail-no-source-json",
			args: []string{"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--json"},
		},
		{
			name: "fail-existing-no-overwrite",
			seed: func(t *testing.T) { credtest.SeedToken(t, sentinel) },
			args: []string{"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin"},
		},
		{
			name: "fail-existing-no-overwrite-json",
			seed: func(t *testing.T) { credtest.SeedToken(t, sentinel) },
			args: []string{"--ref", keyring.Ref, "--key", keyring.KeyAPIToken, "--stdin", "--json"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			credtest.Hermetic(t)
			if tc.seed != nil {
				tc.seed(t)
			}
			stdout, stderr, _, err := runCmd(t, sentinel, tc.args...)
			if strings.Contains(stdout, sentinel) {
				t.Fatalf("stdout leaked sentinel: %q", stdout)
			}
			if strings.Contains(stderr, sentinel) {
				t.Fatalf("stderr leaked sentinel: %q", stderr)
			}
			if err != nil && strings.Contains(err.Error(), sentinel) {
				t.Fatalf("err leaked sentinel: %v", err)
			}
		})
	}
}

func seedConfig(t *testing.T, body string) {
	t.Helper()
	p := credtest.SharedConfigPath(t)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}
