package config

import (
	"errors"
	"testing"

	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestProjectShow_EnvOverridesFile(t *testing.T) {
	t.Setenv("CFL_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("CFL_EMAIL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("CFL_DEFAULT_SPACE", "")
	t.Setenv("CFL_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")
	t.Setenv("CFL_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	t.Setenv("CFL_URL", "https://env.example/wiki")
	t.Setenv("ATLASSIAN_EMAIL", "env@example.com")
	t.Setenv("CFL_DEFAULT_SPACE", "ENV")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "bearer")
	t.Setenv("CFL_CLOUD_ID", "cloud-env")

	proj := ProjectShow("/tmp/config.yml", &Config{
		URL:          "https://file.example/wiki",
		Email:        "file@example.com",
		DefaultSpace: "FILE",
		AuthMethod:   "basic",
		CloudID:      "cloud-file",
	}, nil, keyring.Info{
		Ref:             keyring.Ref,
		TokenConfigured: true,
		TokenSource:     "environment",
	}, nil)

	testutil.Equal(t, ShowValue{Value: "https://env.example/wiki", Source: "CFL_URL"}, proj.URL)
	testutil.Equal(t, ShowValue{Value: "env@example.com", Source: "ATLASSIAN_EMAIL"}, proj.Email)
	testutil.Equal(t, ShowValue{Value: "ENV", Source: "CFL_DEFAULT_SPACE"}, proj.DefaultSpace)
	testutil.Equal(t, ShowValue{Value: "bearer", Source: "ATLASSIAN_AUTH_METHOD"}, proj.AuthMethod)
	testutil.Equal(t, ShowValue{Value: "cloud-env", Source: "CFL_CLOUD_ID"}, proj.CloudID)
	testutil.Equal(t, ShowValue{Value: "configured", Source: "environment"}, proj.APIToken)
}

func TestProjectShow_PrimaryEnvWinsOverFallback(t *testing.T) {
	t.Setenv("CFL_EMAIL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("CFL_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")

	t.Setenv("CFL_EMAIL", "primary@example.com")
	t.Setenv("ATLASSIAN_EMAIL", "fallback@example.com")
	t.Setenv("CFL_AUTH_METHOD", "basic")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "bearer")

	proj := ProjectShow("/tmp/config.yml", &Config{}, nil, keyring.Info{
		Ref:         keyring.Ref,
		TokenSource: "unset",
	}, nil)

	testutil.Equal(t, ShowValue{Value: "primary@example.com", Source: "CFL_EMAIL"}, proj.Email)
	testutil.Equal(t, ShowValue{Value: "basic", Source: "CFL_AUTH_METHOD"}, proj.AuthMethod)
}

func TestProjectShow_FileFallbackAndDefaults(t *testing.T) {
	t.Setenv("CFL_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("CFL_EMAIL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("CFL_DEFAULT_SPACE", "")
	t.Setenv("CFL_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")
	t.Setenv("CFL_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	proj := ProjectShow("/tmp/config.yml", &Config{
		URL:          "https://file.example/wiki",
		Email:        "file@example.com",
		DefaultSpace: "FILE",
	}, nil, keyring.Info{
		Ref:         keyring.Ref,
		TokenSource: "unset",
	}, nil)

	testutil.Equal(t, ShowValue{Value: "https://file.example/wiki", Source: "config"}, proj.URL)
	testutil.Equal(t, ShowValue{Value: "file@example.com", Source: "config"}, proj.Email)
	testutil.Equal(t, ShowValue{Value: "FILE", Source: "config"}, proj.DefaultSpace)
	testutil.Equal(t, ShowValue{Value: "basic", Source: "default"}, proj.AuthMethod)
	testutil.Equal(t, ShowValue{Source: "not set"}, proj.CloudID)
	testutil.Equal(t, ShowValue{Value: "not set", Source: "unset"}, proj.APIToken)
	testutil.Equal(t, ShowValue{Value: keyring.Ref, Source: "fixed"}, proj.KeyringRef)
	testutil.True(t, proj.ConfigReadable)
}

func TestProjectShow_KeyringMetadataAndUnreadableConfig(t *testing.T) {
	t.Setenv("CFL_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("CFL_EMAIL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("CFL_DEFAULT_SPACE", "")
	t.Setenv("CFL_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")
	t.Setenv("CFL_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	passphraseSource := "env:" + "ATLASSIAN_CLI_KEYRING_" + "PASSPHRASE"

	proj := ProjectShow("/tmp/config.yml", &Config{}, errors.New("boom"), keyring.Info{
		Ref:              keyring.Ref,
		Backend:          "file",
		BackendSource:    "flag",
		PassphraseSource: passphraseSource,
		TokenSource:      "unset",
	}, errors.New("backend unavailable"))

	testutil.False(t, proj.ConfigReadable)
	testutil.True(t, proj.HasKeyringBackend)
	testutil.Equal(t, ShowValue{Value: "file (flag)", Source: "-"}, proj.KeyringBackend)
	testutil.True(t, proj.HasKeyringPassphrase)
	testutil.Equal(t, ShowValue{Value: passphraseSource, Source: "-"}, proj.KeyringPassphrase)
	testutil.Equal(t, ShowValue{Value: "not set", Source: "keyring error: backend unavailable"}, proj.APIToken)
}
