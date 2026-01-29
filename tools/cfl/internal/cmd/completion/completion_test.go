package completion

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// createTestRootCmd creates a minimal root command for testing with completion registered.
func createTestRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cfl",
		Short: "Test CLI",
	}

	opts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	Register(rootCmd, opts)
	return rootCmd
}

func TestCompletionCommand(t *testing.T) {
	rootCmd := createTestRootCmd()

	// Find the completion command
	completionCmd, _, err := rootCmd.Find([]string{"completion"})
	require.NoError(t, err)

	assert.Equal(t, "completion", completionCmd.Use)
	assert.NotEmpty(t, completionCmd.Short)
	assert.NotEmpty(t, completionCmd.Long)

	// Should have 4 subcommands
	assert.Len(t, completionCmd.Commands(), 4)
}

func TestBashCompletion(t *testing.T) {
	root := createTestRootCmd()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"completion", "bash"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.NotEmpty(t, output)
	// Bash completions should contain bash-specific markers
	assert.Contains(t, output, "bash completion")
}

func TestZshCompletion(t *testing.T) {
	root := createTestRootCmd()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"completion", "zsh"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.NotEmpty(t, output)
	// Zsh completions should contain zsh-specific markers
	assert.Contains(t, output, "compdef")
}

func TestFishCompletion(t *testing.T) {
	root := createTestRootCmd()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"completion", "fish"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.NotEmpty(t, output)
	// Fish completions should contain fish-specific markers
	assert.Contains(t, output, "complete -c")
}

func TestPowerShellCompletion(t *testing.T) {
	root := createTestRootCmd()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"completion", "powershell"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.NotEmpty(t, output)
	// PowerShell completions should contain PowerShell-specific markers
	assert.Contains(t, output, "Register-ArgumentCompleter")
}

func TestCompletionRejectsExtraArgs(t *testing.T) {
	testCases := []struct {
		name  string
		shell string
	}{
		{"bash rejects args", "bash"},
		{"zsh rejects args", "zsh"},
		{"fish rejects args", "fish"},
		{"powershell rejects args", "powershell"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := createTestRootCmd()

			root.SetArgs([]string{"completion", tc.shell, "unexpected-arg"})

			err := root.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown command")
		})
	}
}
