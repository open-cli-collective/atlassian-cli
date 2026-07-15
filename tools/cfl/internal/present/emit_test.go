package present

import (
	"bytes"
	"testing"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newTestOpts() (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer
	return &root.Options{Stdout: &stdout, Stderr: &stderr}, &stdout, &stderr
}

func TestEmit_SplitsStreams(t *testing.T) {
	t.Parallel()
	opts, stdout, stderr := newTestOpts()

	model := &sharedpresent.OutputModel{
		Sections: []sharedpresent.Section{
			&sharedpresent.DetailSection{Fields: []sharedpresent.Field{{Label: "ID", Value: "1"}}},
			&sharedpresent.MessageSection{Kind: sharedpresent.MessageInfo, Message: "diag", Stream: sharedpresent.StreamStderr},
		},
	}

	testutil.RequireNoError(t, Emit(opts, model))
	testutil.Equal(t, "ID: 1\n", stdout.String())
	testutil.Equal(t, "diag\n", stderr.String())
}

func TestEmit_UsesRenderStyleFromRootOptions(t *testing.T) {
	t.Parallel()
	opts, stdout, stderr := newTestOpts()

	model := &sharedpresent.OutputModel{
		Sections: []sharedpresent.Section{
			&sharedpresent.MessageSection{Kind: sharedpresent.MessageSuccess, Message: "Created page"},
		},
	}

	testutil.RequireNoError(t, Emit(opts, model))
	testutil.Equal(t, "✓ Created page\n", stdout.String())
	testutil.Equal(t, "", stderr.String())
}

func TestEmit_NilModel_WritesNothing(t *testing.T) {
	t.Parallel()
	opts, stdout, stderr := newTestOpts()

	testutil.RequireNoError(t, Emit(opts, nil))
	testutil.Equal(t, "", stdout.String())
	testutil.Equal(t, "", stderr.String())
}
