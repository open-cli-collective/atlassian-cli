package present

import (
	"testing"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	cflconfig "github.com/open-cli-collective/confluence-cli/internal/config"
	"github.com/open-cli-collective/confluence-cli/internal/pageview"
)

func TestSpacePresenter_PresentDetail(t *testing.T) {
	t.Parallel()

	model := SpacePresenter{}.PresentDetail(&api.Space{
		ID:     "123456",
		Key:    "TEST",
		Name:   "Test Space",
		Type:   "global",
		Status: "current",
		Description: &api.SpaceDescription{
			Plain: &api.DescriptionValue{Value: "A test space"},
		},
	}, false)

	detail := requireDetailSection(t, model, 0)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "Key", Value: "TEST"},
		{Label: "Name", Value: "Test Space"},
		{Label: "ID", Value: "123456"},
		{Label: "Type", Value: "global"},
	}, detail.Fields)
}

func TestSpacePresenter_PresentDetail_Full(t *testing.T) {
	t.Parallel()

	model := SpacePresenter{}.PresentDetail(&api.Space{
		ID:     "123456",
		Key:    "TEST",
		Name:   "Test Space",
		Type:   "global",
		Status: "current",
		Description: &api.SpaceDescription{
			Plain: &api.DescriptionValue{Value: "A test space"},
		},
	}, true)

	detail := requireDetailSection(t, model, 0)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "Key", Value: "TEST"},
		{Label: "Name", Value: "Test Space"},
		{Label: "ID", Value: "123456"},
		{Label: "Type", Value: "global"},
		{Label: "Status", Value: "current"},
		{Label: "Description", Value: "A test space"},
	}, detail.Fields)
}

func TestConfigShowPresenter_PresentDetail(t *testing.T) {
	t.Parallel()

	model := ConfigShowPresenter{}.PresentDetail(cflconfig.ShowProjection{
		URL:               cflconfig.ShowValue{Value: "https://example.com/wiki", Source: "config"},
		Email:             cflconfig.ShowValue{Value: "test@example.com", Source: "ATLASSIAN_EMAIL"},
		APIToken:          cflconfig.ShowValue{Value: "configured", Source: "environment"},
		DefaultSpace:      cflconfig.ShowValue{Value: "TEST", Source: "config"},
		AuthMethod:        cflconfig.ShowValue{Value: "basic", Source: "default"},
		CloudID:           cflconfig.ShowValue{Source: "not set"},
		KeyringRef:        cflconfig.ShowValue{Value: "atlassian-cli/default", Source: "fixed"},
		KeyringBackend:    cflconfig.ShowValue{Value: "file (flag)", Source: "-"},
		KeyringPassphrase: cflconfig.ShowValue{Value: "env:ATLASSIAN_CLI_KEYRING_PASSPHRASE", Source: "-"},
		HasKeyringBackend: true, HasKeyringPassphrase: true,
		ConfigPath: "/tmp/config.yml", ConfigReadable: false,
	})

	detail := requireDetailSection(t, model, 0)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "URL", Value: "https://example.com/wiki  (source: config)"},
		{Label: "Email", Value: "test@example.com  (source: ATLASSIAN_EMAIL)"},
		{Label: "API Token", Value: "configured  (source: environment)"},
		{Label: "Default Space", Value: "TEST  (source: config)"},
		{Label: "Auth Method", Value: "basic  (source: default)"},
		{Label: "Cloud ID", Value: "(source: not set)"},
		{Label: "Keyring Ref", Value: "atlassian-cli/default  (source: fixed)"},
		{Label: "Keyring Backend", Value: "file (flag)  (source: -)"},
		{Label: "Keyring Passphrase", Value: "env:ATLASSIAN_CLI_KEYRING_PASSPHRASE  (source: -)"},
	}, detail.Fields)

	msg := requireMessageSection(t, model, 1)
	testutil.Equal(t, sharedpresent.StreamStderr, msg.Stream)
	testutil.Equal(t, "\nConfig file: /tmp/config.yml\n  (file not found or unreadable)", msg.Message)
}

func TestFormatValueWithSource(t *testing.T) {
	t.Parallel()

	testutil.Equal(t, "https://example.com  (source: config)", formatValueWithSource(cflconfig.ShowValue{
		Value:  "https://example.com",
		Source: "config",
	}))
	testutil.Equal(t, "(source: not set)", formatValueWithSource(cflconfig.ShowValue{Source: "not set"}))
}

func TestPagePresenter_PresentView_Default(t *testing.T) {
	t.Parallel()

	model := PagePresenter{}.PresentView(pageview.Projection{
		Title:      "Test Page",
		ID:         "12345",
		SpaceKey:   "TEST",
		SpaceID:    "98765",
		Version:    3,
		HasVersion: true,
		Body:       "Hello world",
		BodyKind:   pageview.BodyKindMarkdown,
		HasContent: true,
	})

	detail := requireDetailSection(t, model, 0)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "Title", Value: "Test Page"},
		{Label: "ID", Value: "12345"},
		{Label: "Space", Value: "TEST (ID: 98765)"},
		{Label: "Version", Value: "3"},
	}, detail.Fields)

	separator := requireMessageSection(t, model, 1)
	testutil.Equal(t, sharedpresent.StreamStdout, separator.Stream)
	testutil.Equal(t, "", separator.Message)

	body := requireMessageSection(t, model, 2)
	testutil.Equal(t, sharedpresent.StreamStdout, body.Stream)
	testutil.Equal(t, "Hello world", body.Message)
}

func TestPagePresenter_PresentView_ContentOnlyWithAdvisory(t *testing.T) {
	t.Parallel()

	model := PagePresenter{}.PresentView(pageview.Projection{
		ContentOnly: true,
		Body:        "<p>Raw</p>",
		BodyKind:    pageview.BodyKindStorageRaw,
		Fallback:    pageview.FallbackStorageRaw,
		HasContent:  true,
	})

	msg := requireMessageSection(t, model, 0)
	testutil.Equal(t, sharedpresent.StreamStderr, msg.Stream)
	testutil.Equal(t, "(Failed to convert to markdown, showing raw HTML)", msg.Message)

	body := requireMessageSection(t, model, 1)
	testutil.Equal(t, sharedpresent.StreamStdout, body.Stream)
	testutil.Equal(t, "<p>Raw</p>", body.Message)
}

func TestPagePresenter_PresentView_EmptyAndTruncated(t *testing.T) {
	t.Parallel()

	emptyModel := PagePresenter{}.PresentView(pageview.Projection{ContentOnly: true})
	emptyBody := requireMessageSection(t, emptyModel, 0)
	testutil.Equal(t, "(No content)", emptyBody.Message)

	truncatedModel := PagePresenter{}.PresentView(pageview.Projection{
		ContentOnly: true,
		Body:        "abc",
		BodyKind:    pageview.BodyKindMarkdown,
		HasContent:  true,
		Truncated:   true,
	})
	truncatedBody := requireMessageSection(t, truncatedModel, 0)
	testutil.Equal(t, "abc\n\n... [truncated at 5000 chars, use --no-truncate for complete text]", truncatedBody.Message)
}

func requireDetailSection(t *testing.T, model *sharedpresent.OutputModel, idx int) *sharedpresent.DetailSection {
	t.Helper()

	if len(model.Sections) <= idx {
		t.Fatalf("expected section index %d, got %d sections", idx, len(model.Sections))
	}
	sec, ok := model.Sections[idx].(*sharedpresent.DetailSection)
	if !ok {
		t.Fatalf("expected DetailSection at %d, got %T", idx, model.Sections[idx])
	}
	return sec
}
