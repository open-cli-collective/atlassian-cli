package setcredential

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func TestSetCredential_StdinStoresToKeyring(t *testing.T) {
	credtest.Hermetic(t)

	opts := &root.Options{
		Stdin:  strings.NewReader("  cfl-wrapper-token\n"),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	rootCmd := &cobra.Command{Use: "cfl"}
	Register(rootCmd, opts)
	rootCmd.SetArgs([]string{"set-credential"})
	testutil.RequireNoError(t, rootCmd.Execute())

	s, err := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, err)
	defer func() { _ = s.Close() }()
	got, ok, err := s.Token(credstore.ToolCFL)
	testutil.RequireNoError(t, err)
	testutil.True(t, ok)
	testutil.Equal(t, "cfl-wrapper-token", got)
}

// cfl must refuse the sibling's override key (storing it would leave a
// token cfl never resolves).
func TestSetCredential_RejectsSiblingKey(t *testing.T) {
	credtest.Hermetic(t)

	opts := &root.Options{
		Stdin:  strings.NewReader("x\n"),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	rootCmd := &cobra.Command{Use: "cfl"}
	Register(rootCmd, opts)
	rootCmd.SetArgs([]string{"set-credential", "--key", keyring.KeyJTKAPIToken})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected cfl set-credential --key jtk_api_token to be rejected")
	}
}
