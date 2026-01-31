package configcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func TestRunClear_FileNotFound(t *testing.T) {
	// Use a temp directory that doesn't have a config file
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	rootOpts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	opts := &clearOptions{
		Options: rootOpts,
		force:   true,
		stdin:   strings.NewReader(""),
	}

	err := runClear(opts)
	require.NoError(t, err)
}

func TestRunClear_WithForce(t *testing.T) {
	// Create a temp config file
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "cfl")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configPath := filepath.Join(configDir, "config.yml")
	err := os.WriteFile(configPath, []byte("url: https://test.atlassian.net"), 0600)
	require.NoError(t, err)

	rootOpts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	opts := &clearOptions{
		Options: rootOpts,
		force:   true,
		stdin:   strings.NewReader(""),
	}

	err = runClear(opts)
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err))
}

func TestRunClear_WithConfirmation(t *testing.T) {
	// Create a temp config file
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "cfl")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configPath := filepath.Join(configDir, "config.yml")
	err := os.WriteFile(configPath, []byte("url: https://test.atlassian.net"), 0600)
	require.NoError(t, err)

	rootOpts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	opts := &clearOptions{
		Options: rootOpts,
		force:   false,
		stdin:   strings.NewReader("y\n"),
	}

	err = runClear(opts)
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err))
}

func TestRunClear_Cancelled(t *testing.T) {
	// Create a temp config file
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "cfl")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configPath := filepath.Join(configDir, "config.yml")
	err := os.WriteFile(configPath, []byte("url: https://test.atlassian.net"), 0600)
	require.NoError(t, err)

	rootOpts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	opts := &clearOptions{
		Options: rootOpts,
		force:   false,
		stdin:   strings.NewReader("n\n"),
	}

	err = runClear(opts)
	require.NoError(t, err)

	// Verify file still exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}
