package config

import (
	"os"
	"testing"
)

func TestGetEnvWithFallback(t *testing.T) {
	// Save and restore environment
	cleanup := func(keys ...string) func() {
		saved := make(map[string]string)
		for _, key := range keys {
			saved[key] = os.Getenv(key)
		}
		return func() {
			for key, val := range saved {
				if val == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, val)
				}
			}
		}
	}

	t.Run("primary set", func(t *testing.T) {
		restore := cleanup("TEST_PRIMARY", "TEST_FALLBACK")
		defer restore()

		os.Setenv("TEST_PRIMARY", "primary-value")
		os.Setenv("TEST_FALLBACK", "fallback-value")

		got := GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK")
		if got != "primary-value" {
			t.Errorf("GetEnvWithFallback() = %v, want primary-value", got)
		}
	})

	t.Run("primary empty uses fallback", func(t *testing.T) {
		restore := cleanup("TEST_PRIMARY", "TEST_FALLBACK")
		defer restore()

		os.Unsetenv("TEST_PRIMARY")
		os.Setenv("TEST_FALLBACK", "fallback-value")

		got := GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK")
		if got != "fallback-value" {
			t.Errorf("GetEnvWithFallback() = %v, want fallback-value", got)
		}
	})

	t.Run("both empty", func(t *testing.T) {
		restore := cleanup("TEST_PRIMARY", "TEST_FALLBACK")
		defer restore()

		os.Unsetenv("TEST_PRIMARY")
		os.Unsetenv("TEST_FALLBACK")

		got := GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK")
		if got != "" {
			t.Errorf("GetEnvWithFallback() = %v, want empty string", got)
		}
	})

	t.Run("primary explicitly empty string", func(t *testing.T) {
		restore := cleanup("TEST_PRIMARY", "TEST_FALLBACK")
		defer restore()

		os.Setenv("TEST_PRIMARY", "")
		os.Setenv("TEST_FALLBACK", "fallback-value")

		got := GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK")
		if got != "fallback-value" {
			t.Errorf("GetEnvWithFallback() = %v, want fallback-value (empty string treated as unset)", got)
		}
	})
}

func TestGetEnvWithDefault(t *testing.T) {
	cleanup := func(key string) func() {
		saved := os.Getenv(key)
		return func() {
			if saved == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, saved)
			}
		}
	}

	t.Run("env set", func(t *testing.T) {
		restore := cleanup("TEST_ENV")
		defer restore()

		os.Setenv("TEST_ENV", "env-value")

		got := GetEnvWithDefault("TEST_ENV", "default-value")
		if got != "env-value" {
			t.Errorf("GetEnvWithDefault() = %v, want env-value", got)
		}
	})

	t.Run("env not set uses default", func(t *testing.T) {
		restore := cleanup("TEST_ENV")
		defer restore()

		os.Unsetenv("TEST_ENV")

		got := GetEnvWithDefault("TEST_ENV", "default-value")
		if got != "default-value" {
			t.Errorf("GetEnvWithDefault() = %v, want default-value", got)
		}
	})

	t.Run("env empty uses default", func(t *testing.T) {
		restore := cleanup("TEST_ENV")
		defer restore()

		os.Setenv("TEST_ENV", "")

		got := GetEnvWithDefault("TEST_ENV", "default-value")
		if got != "default-value" {
			t.Errorf("GetEnvWithDefault() = %v, want default-value", got)
		}
	})
}
