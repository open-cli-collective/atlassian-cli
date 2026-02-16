package automation

import (
	"context"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunUpdate(t *testing.T) {
	t.Run("invalid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "bad.json")
		err := os.WriteFile(filePath, []byte(`not valid json`), 0644)
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
		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err := runUpdate(context.Background(), opts, "12345", "/nonexistent/path/rule.json")
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "failed to read file")
	})
}
