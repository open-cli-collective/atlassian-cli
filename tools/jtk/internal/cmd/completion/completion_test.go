package completion

import (
	"io"
	"os"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newTestRootCmd() *cobra.Command {
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)
	return rootCmd
}

// captureStdout captures output written to os.Stdout during the execution of fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	testutil.RequireNoError(t, err)

	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = origStdout

	out, err := io.ReadAll(r)
	testutil.RequireNoError(t, err)

	return string(out)
}

func TestNewCompletionCmd(t *testing.T) {
	t.Parallel()
	rootCmd := newTestRootCmd()

	cmd, _, err := rootCmd.Find([]string{"completion"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, cmd.Name(), "completion")
	testutil.True(t, cmd.DisableFlagsInUseLine)

	testutil.Len(t, cmd.ValidArgs, 4)
	testutil.Contains(t, stringSliceToString(cmd.ValidArgs), "bash")
	testutil.Contains(t, stringSliceToString(cmd.ValidArgs), "zsh")
	testutil.Contains(t, stringSliceToString(cmd.ValidArgs), "fish")
	testutil.Contains(t, stringSliceToString(cmd.ValidArgs), "powershell")
}

func TestRunCompletion_Bash(t *testing.T) {
	rootCmd := newTestRootCmd()
	rootCmd.SetArgs([]string{"completion", "bash"})

	output := captureStdout(t, func() {
		err := rootCmd.Execute()
		testutil.RequireNoError(t, err)
	})

	testutil.Contains(t, output, "bash")
	testutil.Contains(t, output, "__jtk")
}

func TestRunCompletion_Zsh(t *testing.T) {
	rootCmd := newTestRootCmd()
	rootCmd.SetArgs([]string{"completion", "zsh"})

	output := captureStdout(t, func() {
		err := rootCmd.Execute()
		testutil.RequireNoError(t, err)
	})

	testutil.Contains(t, output, "zsh")
}

func TestRunCompletion_Fish(t *testing.T) {
	rootCmd := newTestRootCmd()
	rootCmd.SetArgs([]string{"completion", "fish"})

	output := captureStdout(t, func() {
		err := rootCmd.Execute()
		testutil.RequireNoError(t, err)
	})

	testutil.Contains(t, output, "fish")
}

func TestRunCompletion_PowerShell(t *testing.T) {
	rootCmd := newTestRootCmd()
	rootCmd.SetArgs([]string{"completion", "powershell"})

	output := captureStdout(t, func() {
		err := rootCmd.Execute()
		testutil.RequireNoError(t, err)
	})

	testutil.Contains(t, output, "jtk")
}

func stringSliceToString(ss []string) string {
	result := ""
	for _, s := range ss {
		result += s + " "
	}
	return result
}
