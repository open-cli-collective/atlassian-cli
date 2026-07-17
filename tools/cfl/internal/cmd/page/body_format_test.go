package page

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func TestBodyForInput(t *testing.T) {
	t.Parallel()

	adf := " \n{\"type\":\"doc\",\"version\":1,\"content\":[]}\n"
	xhtml := "<p>exact XHTML</p>\n"

	markdownBody, err := bodyForInput("# Heading", bodyFormatMarkdown, false)
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, markdownBody.AtlasDocFormat)
	testutil.Contains(t, markdownBody.AtlasDocFormat.Value, `"type":"heading"`)

	adfBody, err := bodyForInput(adf, bodyFormatADF, false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, adf, adfBody.AtlasDocFormat.Value)

	xhtmlBody, err := bodyForInput(xhtml, bodyFormatXHTML, false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, xhtml, xhtmlBody.Storage.Value)

	_, err = bodyForInput("{broken", bodyFormatADF, false)
	testutil.RequireError(t, err)
	for _, input := range []string{"null", "[]", "{}", `{"type":"paragraph"}`} {
		_, err = bodyForInput(input, bodyFormatADF, false)
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "expected an object with type")
	}
	_, err = bodyForInput(adf, bodyFormatADF, true)
	testutil.RequireError(t, err)
}

func TestBodyFormatValidation(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", "pdf"} {
		cmd := newViewCmd(&root.Options{})
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"12345", "--body-format=" + value})
		err := cmd.Execute()
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "invalid body format")
	}

	cmd := newCreateCmd(&root.Options{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--title", "Page", "--body-format", "adf", "--legacy"})
	err := cmd.Execute()
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "--legacy is only supported")
}

func TestCreateMalformedADFDoesNotRequest(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()

	rootOpts := &root.Options{Stdin: strings.NewReader("{broken"), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))
	err := runCreate(context.Background(), &createOptions{
		Options: rootOpts, space: "DEV", title: "Page", bodyFormat: bodyFormatADF,
	})
	testutil.RequireError(t, err)
	testutil.Equal(t, 0, requests)
}

func TestBodyFormatFlagsReplaceLegacyFormatFlags(t *testing.T) {
	t.Parallel()
	rootOpts := &root.Options{}

	for _, cmd := range []*cobra.Command{newViewCmd(rootOpts), newCreateCmd(rootOpts), newEditCmd(rootOpts)} {
		testutil.NotNil(t, cmd.Flags().Lookup("body-format"))
		testutil.Nil(t, cmd.Flags().Lookup("raw"))
		testutil.Nil(t, cmd.Flags().Lookup("storage"))
		testutil.Nil(t, cmd.Flags().Lookup("no-markdown"))
	}
}
