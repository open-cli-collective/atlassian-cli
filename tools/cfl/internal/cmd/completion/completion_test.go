package completion

import (
	"bytes"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

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
	t.Parallel()
	rootCmd := createTestRootCmd()

	// Find the completion command
	completionCmd, _, err := rootCmd.Find([]string{"completion"})
	testutil.RequireNoError(t, err)

	testutil.Equal(t, "completion [bash|zsh|fish|powershell]", completionCmd.Use)
	testutil.NotEmpty(t, completionCmd.Short)
	testutil.NotEmpty(t, completionCmd.Long)
	testutil.Equal(t, []string{"bash", "zsh", "fish", "powershell"}, completionCmd.ValidArgs)
}

func TestBashCompletion(t *testing.T) {
	t.Parallel()
	root := createTestRootCmd()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"completion", "bash"})

	err := root.Execute()
	testutil.RequireNoError(t, err)

	output := buf.String()
	testutil.NotEmpty(t, output)
	// Bash completions should contain bash-specific markers
	testutil.Contains(t, output, "bash completion")
}

func TestZshCompletion(t *testing.T) {
	t.Parallel()
	root := createTestRootCmd()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"completion", "zsh"})

	err := root.Execute()
	testutil.RequireNoError(t, err)

	output := buf.String()
	testutil.NotEmpty(t, output)
	// Zsh completions should contain zsh-specific markers
	testutil.Contains(t, output, "compdef")
}

func TestFishCompletion(t *testing.T) {
	t.Parallel()
	root := createTestRootCmd()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"completion", "fish"})

	err := root.Execute()
	testutil.RequireNoError(t, err)

	output := buf.String()
	testutil.NotEmpty(t, output)
	// Fish completions should contain fish-specific markers
	testutil.Contains(t, output, "complete -c")
}

func TestPowerShellCompletion(t *testing.T) {
	t.Parallel()
	root := createTestRootCmd()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"completion", "powershell"})

	err := root.Execute()
	testutil.RequireNoError(t, err)

	output := buf.String()
	testutil.NotEmpty(t, output)
	// PowerShell completions should contain PowerShell-specific markers
	testutil.Contains(t, output, "Register-ArgumentCompleter")
}

func TestCompletionRequiresShellArg(t *testing.T) {
	t.Parallel()
	root := createTestRootCmd()

	root.SetArgs([]string{"completion"})
	root.SetErr(&bytes.Buffer{}) // Suppress error output

	err := root.Execute()
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestCompletionRejectsInvalidShell(t *testing.T) {
	t.Parallel()
	root := createTestRootCmd()

	root.SetArgs([]string{"completion", "invalid-shell"})
	root.SetErr(&bytes.Buffer{}) // Suppress error output

	err := root.Execute()
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid argument")
}

func TestCompletionRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	root := createTestRootCmd()

	root.SetArgs([]string{"completion", "bash", "extra-arg"})
	root.SetErr(&bytes.Buffer{}) // Suppress error output

	err := root.Execute()
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "accepts 1 arg(s)")
}
