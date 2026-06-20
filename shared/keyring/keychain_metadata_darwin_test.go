//go:build darwin && cgo

package keyring

import (
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	rawkeyring "github.com/byteness/keyring"
	cccredstore "github.com/open-cli-collective/cli-common/credstore"
)

func TestKeychainMetadataGated(t *testing.T) {
	if os.Getenv("ATLASSIAN_KEYCHAIN_METADATA_TEST") != "1" {
		t.Skip("set ATLASSIAN_KEYCHAIN_METADATA_TEST=1 to run real macOS Keychain metadata tests")
	}

	realHome := os.Getenv("HOME")
	hermetic(t)
	t.Setenv("HOME", realHome)
	SetBackendSelection(cccredstore.BackendKeychain, "")
	t.Cleanup(func() { SetBackendSelection("", "") })

	kr, err := rawkeyring.Open(rawkeyring.Config{
		ServiceName:              Service,
		AllowedBackends:          []rawkeyring.BackendType{rawkeyring.KeychainBackend},
		KeychainTrustApplication: true,
	})
	if err != nil {
		t.Fatalf("open raw Keychain: %v", err)
	}

	t.Run("synthetic profile", func(t *testing.T) {
		profile := "metadata-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		runKeychainMetadataScenario(t, kr, profile)
	})

	t.Run("canonical default profile", func(t *testing.T) {
		hadOriginal, err := hasRawMetadata(kr, Profile+"/"+KeyAPIToken)
		if err != nil {
			t.Fatalf("inspect existing canonical Keychain item metadata: %v", err)
		}
		if hadOriginal && os.Getenv("ATLASSIAN_KEYCHAIN_METADATA_BACKUP_EXISTING") != "1" {
			t.Skip("canonical Keychain item already exists; set ATLASSIAN_KEYCHAIN_METADATA_BACKUP_EXISTING=1 to allow secret-data backup/restore before mutation")
		}
		runKeychainMetadataScenario(t, kr, Profile)
	})
}

func runKeychainMetadataScenario(t *testing.T, kr rawkeyring.Keyring, profile string) {
	t.Helper()

	account := profile + "/" + KeyAPIToken
	ref := Service + "/" + profile
	t.Logf("using Keychain item service=%q account=%q", Service, account)

	original, hadOriginal, err := getRawItem(kr, account)
	if err != nil {
		t.Fatalf("backup existing Keychain item data: %v", err)
	}
	t.Logf("backed up existing Keychain item: %v", hadOriginal)
	t.Cleanup(func() {
		if hadOriginal {
			if err := kr.Set(original); err != nil {
				t.Errorf("restore original Keychain item: %v", err)
			}
			return
		}
		if err := removeRawItem(kr, account); err != nil {
			t.Errorf("cleanup synthetic Keychain item: %v", err)
		}
	})

	if err := removeRawItem(kr, account); err != nil {
		t.Fatalf("clear existing Keychain item before fresh write: %v", err)
	}
	t.Logf("cleared Keychain account %q for fresh-write coverage", account)

	wantLabel := Service + " " + account
	wantDescription := "Credential for " + Service + " " + account

	writeToken(t, ref, KeyAPIToken, "atlassian-fresh-token")
	assertStoredToken(t, ref, "atlassian-fresh-token")
	assertMetadata(t, kr, account, wantLabel, wantDescription)
	t.Log("verified fresh write metadata")

	if err := removeRawItem(kr, account); err != nil {
		t.Fatalf("clear fresh Keychain item before repair seed: %v", err)
	}
	if err := kr.Set(rawkeyring.Item{
		Key:         account,
		Data:        []byte("atlassian-legacy-token"),
		Label:       "stale Atlassian token",
		Description: "stale metadata before cli-common repair",
	}); err != nil {
		t.Fatalf("seed stale Keychain item: %v", err)
	}
	assertMetadata(t, kr, account, "stale Atlassian token", "stale metadata before cli-common repair")

	writeToken(t, ref, KeyAPIToken, "atlassian-repaired-token")
	assertStoredToken(t, ref, "atlassian-repaired-token")
	assertMetadata(t, kr, account, wantLabel, wantDescription)
	t.Log("verified overwrite metadata repair")
}

func writeToken(t *testing.T, ref, key, value string) {
	t.Helper()
	s, err := openRef(ref, allowedKeys)
	if err != nil {
		t.Fatalf("open shared keyring: %v", err)
	}
	defer func() { _ = s.Close() }()
	if err := s.SetToken(key, value); err != nil {
		t.Fatalf("SetToken(%s): %v", key, err)
	}
}

func assertStoredToken(t *testing.T, ref, want string) {
	t.Helper()
	s, err := openRef(ref, allowedKeys)
	if err != nil {
		t.Fatalf("open shared keyring for readback: %v", err)
	}
	defer func() { _ = s.Close() }()
	got, ok, err := s.Token()
	if err != nil || !ok || got != want {
		t.Fatalf("Token() = (%q,%v,%v), want (%q,true,nil)", got, ok, err, want)
	}
}

func assertMetadata(t *testing.T, kr rawkeyring.Keyring, account, wantLabel, wantDescription string) {
	t.Helper()
	md, err := kr.GetMetadata(account)
	if err != nil {
		t.Fatalf("GetMetadata(%s): %v", account, err)
	}
	if md.Item == nil {
		t.Fatal("metadata item is nil")
	}
	if md.Label != wantLabel {
		t.Fatalf("Label = %q, want %q", md.Label, wantLabel)
	}
	if md.Description != wantDescription {
		t.Fatalf("Description = %q, want %q", md.Description, wantDescription)
	}
	if len(md.Data) != 0 {
		t.Fatalf("metadata unexpectedly included secret data: %q", string(md.Data))
	}
}

func getRawItem(kr rawkeyring.Keyring, account string) (rawkeyring.Item, bool, error) {
	it, err := kr.Get(account)
	if errors.Is(err, rawkeyring.ErrKeyNotFound) {
		return rawkeyring.Item{}, false, nil
	}
	if err != nil {
		return rawkeyring.Item{}, false, err
	}
	return it, true, nil
}

func hasRawMetadata(kr rawkeyring.Keyring, account string) (bool, error) {
	_, err := kr.GetMetadata(account)
	if errors.Is(err, rawkeyring.ErrKeyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func removeRawItem(kr rawkeyring.Keyring, account string) error {
	if err := kr.Remove(account); err != nil && !errors.Is(err, rawkeyring.ErrKeyNotFound) {
		return err
	}
	return nil
}
