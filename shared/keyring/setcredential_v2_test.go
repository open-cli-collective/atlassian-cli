package keyring_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"
)

const v2Sentinel = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA0123456789sentinel"

func TestSetCredentialV2_StdinSuccess(t *testing.T) {
	credtest.Hermetic(t)

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel + "\n"),
		Ref:      keyring.Ref,
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, true, res.Written)
	testutil.Equal(t, keyring.Ref, res.Ref)
	testutil.Equal(t, keyring.KeyAPIToken, res.Key)
	if res.Backend == "" {
		t.Fatal("Backend must be populated on success")
	}
	if res.Error != "" {
		t.Fatalf("Error must be empty on success, got %q", res.Error)
	}

	got, ok, err := readbackToken(t)
	testutil.RequireNoError(t, err)
	testutil.True(t, ok)
	testutil.Equal(t, v2Sentinel, got)
}

func TestSetCredentialV2_FromEnvSuccess(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("V2_TEST_TOKEN_ENV", v2Sentinel)

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Ref:     keyring.Ref,
		Key:     keyring.KeyAPIToken,
		FromEnv: "V2_TEST_TOKEN_ENV",
	})
	testutil.RequireNoError(t, err)
	testutil.True(t, res.Written)

	got, ok, err := readbackToken(t)
	testutil.RequireNoError(t, err)
	testutil.True(t, ok)
	testutil.Equal(t, v2Sentinel, got)
}

func TestSetCredentialV2_BothSourcesRejected_PreKeyring(t *testing.T) {
	credtest.Hermetic(t)

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Ref:      keyring.Ref,
		Key:      keyring.KeyAPIToken,
		FromEnv:  "ANYTHING",
		UseStdin: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	testutil.Equal(t, "", res.Backend)
	testutil.Equal(t, false, res.Written)
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error should mention mutual exclusion, got %q", err)
	}
}

func TestSetCredentialV2_NeitherSourceRejected_PreKeyring(t *testing.T) {
	credtest.Hermetic(t)

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Ref: keyring.Ref,
		Key: keyring.KeyAPIToken,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	testutil.Equal(t, "", res.Backend)
	testutil.Equal(t, false, res.Written)
	if !strings.Contains(err.Error(), "--stdin") || !strings.Contains(err.Error(), "--from-env") {
		t.Fatalf("error should mention both flag options, got %q", err)
	}
}

func TestSetCredentialV2_KeyOmittedRejected_PreKeyring(t *testing.T) {
	credtest.Hermetic(t)

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Ref:      keyring.Ref,
		UseStdin: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	testutil.Equal(t, "", res.Backend)
	if !strings.Contains(err.Error(), "--key") || !strings.Contains(err.Error(), keyring.KeyAPIToken) {
		t.Fatalf("error should mention --key and %s, got %q", keyring.KeyAPIToken, err)
	}
}

func TestSetCredentialV2_NonCanonicalKeyRejected_PreKeyring(t *testing.T) {
	credtest.Hermetic(t)

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Ref:      keyring.Ref,
		Key:      "bogus_key",
		UseStdin: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	testutil.Equal(t, "", res.Backend)
	if !strings.Contains(err.Error(), "bogus_key") || !strings.Contains(err.Error(), keyring.KeyAPIToken) {
		t.Fatalf("error should name the bad key and the only valid key, got %q", err)
	}
}

func TestSetCredentialV2_NonCanonicalRefRejected_PreKeyring(t *testing.T) {
	credtest.Hermetic(t)
	seedShared(t)

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Ref:      "bogus/ref",
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	testutil.Equal(t, "", res.Backend)
	if !strings.Contains(err.Error(), "bogus/ref") || !strings.Contains(err.Error(), keyring.Ref) {
		t.Fatalf("error should name the bad ref and the only valid ref, got %q", err)
	}
}

func TestSetCredentialV2_RefOmittedNoConfig_PreKeyring(t *testing.T) {
	credtest.Hermetic(t) // empty state — no config.yml

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	})
	if err == nil {
		t.Fatal("expected error on fresh install with no --ref")
	}
	testutil.Equal(t, "", res.Backend)
	if !strings.Contains(err.Error(), "--ref") || !strings.Contains(err.Error(), keyring.Ref) {
		t.Fatalf("error should hint at --ref %s, got %q", keyring.Ref, err)
	}
}

func TestSetCredentialV2_RefOmittedConfigExists_DefaultsToCanonical(t *testing.T) {
	credtest.Hermetic(t)
	seedShared(t)

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, keyring.Ref, res.Ref)
	testutil.True(t, res.Written)
}

func TestSetCredentialV2_RefOmittedCorruptConfig_PreKeyring(t *testing.T) {
	credtest.Hermetic(t)
	p := credtest.SharedConfigPath(t)
	if err := writeFileSimple(p, "[unclosed_array: yes\n"); err != nil {
		t.Fatalf("seed corrupt config: %v", err)
	}

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	})
	if err == nil {
		t.Fatal("expected error from corrupt config probe")
	}
	if !errors.Is(err, credstore.ErrCorruptStore) {
		t.Fatalf("error should wrap ErrCorruptStore, got %v", err)
	}
	testutil.Equal(t, "", res.Backend)
	if res.Written {
		t.Fatal("Written must be false on pre-keyring failure")
	}
}

func TestSetCredentialV2_ExistingNoOverwriteFails_PostKeyring(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "pre-existing-token-value")

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Ref:      keyring.Ref,
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	})
	if err == nil {
		t.Fatal("expected error on existing entry without --overwrite")
	}
	if res.Backend == "" {
		t.Fatal("Backend must be populated on post-keyring failure")
	}
	testutil.False(t, res.Written)
	if !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("error should mention --overwrite, got %q", err)
	}

	got, ok, rerr := readbackToken(t)
	testutil.RequireNoError(t, rerr)
	testutil.True(t, ok)
	testutil.Equal(t, "pre-existing-token-value", got)
}

func TestSetCredentialV2_ExistingWithOverwriteReplaces_PostKeyring(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "old-token")

	res, err := keyring.SetCredentialV2(keyring.SetCredentialOpts{
		Stdin:     strings.NewReader(v2Sentinel),
		Ref:       keyring.Ref,
		Key:       keyring.KeyAPIToken,
		UseStdin:  true,
		Overwrite: true,
	})
	testutil.RequireNoError(t, err)
	testutil.True(t, res.Written)

	got, ok, rerr := readbackToken(t)
	testutil.RequireNoError(t, rerr)
	testutil.True(t, ok)
	testutil.Equal(t, v2Sentinel, got)
}

func TestSetCredentialV2_NeverEmitsSecret_AllPaths(t *testing.T) {
	cases := []struct {
		name string
		seed func(t *testing.T)
		opts keyring.SetCredentialOpts
	}{
		{
			name: "happy-stdin",
			opts: keyring.SetCredentialOpts{Stdin: strings.NewReader(v2Sentinel), Ref: keyring.Ref, Key: keyring.KeyAPIToken, UseStdin: true},
		},
		{
			name: "happy-env",
			seed: func(t *testing.T) { t.Setenv("V2_LEAK_ENV", v2Sentinel) },
			opts: keyring.SetCredentialOpts{Ref: keyring.Ref, Key: keyring.KeyAPIToken, FromEnv: "V2_LEAK_ENV"},
		},
		{
			name: "fail-both-sources",
			opts: keyring.SetCredentialOpts{Stdin: strings.NewReader(v2Sentinel), Ref: keyring.Ref, Key: keyring.KeyAPIToken, FromEnv: "V2_LEAK_ENV", UseStdin: true},
		},
		{
			name: "fail-empty-source",
			opts: keyring.SetCredentialOpts{Stdin: strings.NewReader("   "), Ref: keyring.Ref, Key: keyring.KeyAPIToken, UseStdin: true},
		},
		{
			name: "fail-existing-no-overwrite",
			seed: func(t *testing.T) { credtest.SeedToken(t, v2Sentinel) },
			opts: keyring.SetCredentialOpts{Stdin: strings.NewReader(v2Sentinel), Ref: keyring.Ref, Key: keyring.KeyAPIToken, UseStdin: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			credtest.Hermetic(t)
			if tc.seed != nil {
				tc.seed(t)
			}
			res, err := keyring.SetCredentialV2(tc.opts)
			if strings.Contains(res.Ref, v2Sentinel) ||
				strings.Contains(res.Key, v2Sentinel) ||
				strings.Contains(res.Backend, v2Sentinel) ||
				strings.Contains(res.Error, v2Sentinel) {
				t.Fatalf("envelope leaked sentinel: %+v", res)
			}
			if err != nil && strings.Contains(err.Error(), v2Sentinel) {
				t.Fatalf("returned error leaked sentinel: %v", err)
			}
		})
	}
}

func TestRunSetCredential_JSONEmitsEnvelope_OnSuccess(t *testing.T) {
	credtest.Hermetic(t)
	var stdout, stderr bytes.Buffer

	err := keyring.RunSetCredential(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Ref:      keyring.Ref,
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	}, &stdout, &stderr, true)
	testutil.RequireNoError(t, err)

	if stderr.Len() != 0 {
		t.Fatalf("stderr must be empty under --json on success, got %q", stderr.String())
	}
	line := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(line, `{"ref":"`+keyring.Ref+`"`) {
		t.Fatalf("envelope shape mismatch: %q", line)
	}
	if !strings.Contains(line, `"written":true`) {
		t.Fatalf("written must be true: %q", line)
	}
	if strings.Contains(line, v2Sentinel) {
		t.Fatal("envelope leaked sentinel")
	}
}

func TestRunSetCredential_JSONEmitsEnvelope_OnPreKeyringFailure(t *testing.T) {
	credtest.Hermetic(t)
	var stdout, stderr bytes.Buffer

	err := keyring.RunSetCredential(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	}, &stdout, &stderr, true)
	if err == nil {
		t.Fatal("expected pre-keyring error")
	}
	if !errors.Is(err, keyring.ErrSetCredentialEnvelopeEmitted) {
		t.Fatalf("error must wrap ErrSetCredentialEnvelopeEmitted (so main() suppresses double-print), got %v", err)
	}

	line := strings.TrimSpace(stdout.String())
	if !strings.Contains(line, `"backend":""`) {
		t.Fatalf("pre-keyring envelope must have empty backend: %q", line)
	}
	if !strings.Contains(line, `"written":false`) {
		t.Fatalf("written must be false: %q", line)
	}
	if !strings.Contains(line, `"error":"`) {
		t.Fatalf("envelope must carry error field: %q", line)
	}
}

func TestRunSetCredential_HumanLineOnSuccess(t *testing.T) {
	credtest.Hermetic(t)
	var stdout, stderr bytes.Buffer

	err := keyring.RunSetCredential(keyring.SetCredentialOpts{
		Stdin:    strings.NewReader(v2Sentinel),
		Ref:      keyring.Ref,
		Key:      keyring.KeyAPIToken,
		UseStdin: true,
	}, &stdout, &stderr, false)
	testutil.RequireNoError(t, err)
	if stdout.Len() != 0 {
		t.Fatalf("stdout must be empty without --json, got %q", stdout.String())
	}
	// §1.5.2 verbatim — "wrote <key> to <ref> via <backend>"; no tool suffix.
	want := "wrote " + keyring.KeyAPIToken + " to " + keyring.Ref + " via "
	if !strings.HasPrefix(stderr.String(), want) {
		t.Fatalf("stderr line shape mismatch: got %q, want prefix %q", stderr.String(), want)
	}
	if strings.Contains(stderr.String(), "(") {
		t.Fatalf("stderr line must not have tool suffix per §1.5.2: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), v2Sentinel) {
		t.Fatal("stderr leaked sentinel")
	}
}

// TestRunSetCredential_JSONDrainsMigrationNotice — under --json, the
// §1.8 one-time migration notice that SetCredentialV2's migrating Open()
// may record MUST be drained so the caller's later FlushMigrationNotice
// has nothing to write. Stderr stays empty even after a migration.
func TestRunSetCredential_JSONDrainsMigrationNotice(t *testing.T) {
	keyring.ResetMigrationNotice()
	t.Cleanup(keyring.ResetMigrationNotice)
	credtest.Hermetic(t)
	// Seed a deprecated per-tool key so §1.8 migration runs during Open().
	credtest.SeedDeprecatedKey(t, "cfl_api_token", "legacy-token-value")

	var stdout, stderr bytes.Buffer
	err := keyring.RunSetCredential(keyring.SetCredentialOpts{
		Stdin:     strings.NewReader(v2Sentinel),
		Ref:       keyring.Ref,
		Key:       keyring.KeyAPIToken,
		UseStdin:  true,
		Overwrite: true, // migration consolidates to api_token; --overwrite required
	}, &stdout, &stderr, true)
	testutil.RequireNoError(t, err)

	// Caller's downstream FlushMigrationNotice into a buffer to assert it
	// got drained by RunSetCredential.
	var sink bytes.Buffer
	keyring.FlushMigrationNotice(&sink)
	if sink.Len() != 0 {
		t.Fatalf("--json must drain the migration notice; caller would have written %q", sink.String())
	}
}

func readbackToken(t *testing.T) (string, bool, error) {
	t.Helper()
	s, err := keyring.OpenNoMigrate()
	if err != nil {
		return "", false, err
	}
	defer func() { _ = s.Close() }()
	return s.Token()
}

func seedShared(t *testing.T) {
	t.Helper()
	p := credtest.SharedConfigPath(t)
	if err := writeFileSimple(p, "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\n"); err != nil {
		t.Fatalf("seed shared: %v", err)
	}
}

func writeFileSimple(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o600)
}
