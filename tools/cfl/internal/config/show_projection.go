package config

import (
	"os"

	sharedconfig "github.com/open-cli-collective/atlassian-go/config"
	"github.com/open-cli-collective/atlassian-go/keyring"
)

type ShowValue struct {
	Value  string
	Source string
}

type ShowProjection struct {
	URL               ShowValue
	Email             ShowValue
	APIToken          ShowValue
	DefaultSpace      ShowValue
	AuthMethod        ShowValue
	CloudID           ShowValue
	KeyringRef        ShowValue
	KeyringBackend    ShowValue
	KeyringPassphrase ShowValue

	HasKeyringBackend    bool
	HasKeyringPassphrase bool
	ConfigPath           string
	ConfigReadable       bool
}

func ProjectShow(configPath string, fileCfg *Config, fileErr error, kr keyring.Info, keyringErr error) ShowProjection {
	if fileCfg == nil {
		fileCfg = &Config{}
	}

	url := resolveShowValue(
		sharedconfig.GetEnvWithFallback("CFL_URL", "ATLASSIAN_URL"),
		fileCfg.URL,
		activeEnvVarName("CFL_URL", "ATLASSIAN_URL"),
	)
	email := resolveShowValue(
		sharedconfig.GetEnvWithFallback("CFL_EMAIL", "ATLASSIAN_EMAIL"),
		fileCfg.Email,
		activeEnvVarName("CFL_EMAIL", "ATLASSIAN_EMAIL"),
	)
	defaultSpace := resolveShowValue(os.Getenv("CFL_DEFAULT_SPACE"), fileCfg.DefaultSpace, "CFL_DEFAULT_SPACE")
	authMethod := resolveShowValue(
		sharedconfig.GetEnvWithFallback("CFL_AUTH_METHOD", "ATLASSIAN_AUTH_METHOD"),
		fileCfg.AuthMethod,
		activeEnvVarName("CFL_AUTH_METHOD", "ATLASSIAN_AUTH_METHOD"),
	)
	cloudID := resolveShowValue(
		sharedconfig.GetEnvWithFallback("CFL_CLOUD_ID", "ATLASSIAN_CLOUD_ID"),
		fileCfg.CloudID,
		activeEnvVarName("CFL_CLOUD_ID", "ATLASSIAN_CLOUD_ID"),
	)

	if authMethod.Value == "" {
		authMethod = ShowValue{Value: "basic", Source: "default"}
	}

	tokenStatus := "not set"
	if kr.TokenConfigured {
		tokenStatus = "configured"
	}
	tokenSource := kr.TokenSource
	if keyringErr != nil {
		tokenSource = "keyring error: " + keyringErr.Error()
	}

	projection := ShowProjection{
		URL:          url,
		Email:        email,
		APIToken:     ShowValue{Value: tokenStatus, Source: tokenSource},
		DefaultSpace: defaultSpace,
		AuthMethod:   authMethod,
		CloudID:      cloudID,
		KeyringRef: ShowValue{
			Value:  kr.Ref,
			Source: "fixed",
		},
		ConfigPath:     configPath,
		ConfigReadable: fileErr == nil,
	}

	if kr.Backend != "" {
		backendValue := kr.Backend
		if kr.BackendSource != "" {
			backendValue += " (" + kr.BackendSource + ")"
		}
		projection.KeyringBackend = ShowValue{Value: backendValue, Source: "-"}
		projection.HasKeyringBackend = true
	}

	if kr.PassphraseSource != "" {
		projection.KeyringPassphrase = ShowValue{Value: kr.PassphraseSource, Source: "-"}
		projection.HasKeyringPassphrase = true
	}

	return projection
}

func resolveShowValue(envValue, fileValue, envVarName string) ShowValue {
	if envValue != "" {
		return ShowValue{Value: envValue, Source: envVarName}
	}
	if fileValue != "" {
		return ShowValue{Value: fileValue, Source: "config"}
	}
	return ShowValue{Source: "not set"}
}

func activeEnvVarName(primary, fallback string) string {
	if os.Getenv(primary) != "" {
		return primary
	}
	if os.Getenv(fallback) != "" {
		return fallback
	}
	return primary
}
