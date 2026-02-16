package config

import (
	"testing"
)

func TestGetEnvWithFallback(t *testing.T) {
	t.Run("primary set", func(t *testing.T) {
		t.Setenv("TEST_PRIMARY", "primary-value")
		t.Setenv("TEST_FALLBACK", "fallback-value")

		got := GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK")
		if got != "primary-value" {
			t.Errorf("GetEnvWithFallback() = %v, want primary-value", got)
		}
	})

	t.Run("primary empty uses fallback", func(t *testing.T) {
		t.Setenv("TEST_PRIMARY", "")
		t.Setenv("TEST_FALLBACK", "fallback-value")

		got := GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK")
		if got != "fallback-value" {
			t.Errorf("GetEnvWithFallback() = %v, want fallback-value", got)
		}
	})

	t.Run("both empty", func(t *testing.T) {
		t.Setenv("TEST_PRIMARY", "")
		t.Setenv("TEST_FALLBACK", "")

		got := GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK")
		if got != "" {
			t.Errorf("GetEnvWithFallback() = %v, want empty string", got)
		}
	})

	t.Run("primary explicitly empty string", func(t *testing.T) {
		t.Setenv("TEST_PRIMARY", "")
		t.Setenv("TEST_FALLBACK", "fallback-value")

		got := GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK")
		if got != "fallback-value" {
			t.Errorf("GetEnvWithFallback() = %v, want fallback-value (empty string treated as unset)", got)
		}
	})
}

func TestGetEnvWithDefault(t *testing.T) {
	t.Run("env set", func(t *testing.T) {
		t.Setenv("TEST_ENV", "env-value")

		got := GetEnvWithDefault("TEST_ENV", "default-value")
		if got != "env-value" {
			t.Errorf("GetEnvWithDefault() = %v, want env-value", got)
		}
	})

	t.Run("env not set uses default", func(t *testing.T) {
		t.Setenv("TEST_ENV", "")

		got := GetEnvWithDefault("TEST_ENV", "default-value")
		if got != "default-value" {
			t.Errorf("GetEnvWithDefault() = %v, want default-value", got)
		}
	})

	t.Run("env empty uses default", func(t *testing.T) {
		t.Setenv("TEST_ENV", "")

		got := GetEnvWithDefault("TEST_ENV", "default-value")
		if got != "default-value" {
			t.Errorf("GetEnvWithDefault() = %v, want default-value", got)
		}
	})
}
