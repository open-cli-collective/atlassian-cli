package automation

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunUpdate(t *testing.T) {
	t.Parallel()
	t.Run("invalid JSON file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "bad.json")
		err := os.WriteFile(filePath, []byte(`not valid json`), 0600)
		testutil.RequireNoError(t, err)

		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err = runUpdate(context.Background(), opts, "12345", filePath)
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "does not contain valid JSON")
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err := runUpdate(context.Background(), opts, "12345", "/nonexistent/path/rule.json")
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "no such file or directory")
	})
}
