package attachment

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestStorageAttachmentRefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		storage  string
		filename string
		want     bool
	}{
		{"image reference", `<ac:image><ri:attachment ri:filename="a.png"/></ac:image>`, "a.png", true},
		{"link reference", `<ac:link><ri:attachment ri:filename="a b.pdf"/></ac:link>`, "a b.pdf", true},
		{"entity-encoded filename", `<ac:link><ri:attachment ri:filename="Q&amp;A.pdf"/></ac:link>`, "Q&A.pdf", true},
		{"prose mention", `<p>see a.png</p>`, "a.png", false},
		{"download link to this page's attachment", `<a href="https://example.atlassian.net/wiki/download/attachments/12345/a%20b.pdf?api=v2">file</a>`, "a b.pdf", true},
		{"download link to another page's attachment", `<a href="/wiki/download/attachments/99999/a%20b.pdf">file</a>`, "a b.pdf", false},
		{"link text naming the file", `<a href="https://example.com/docs">a b.pdf</a>`, "a b.pdf", false},
		{"filename is a suffix of another", `<ri:attachment ri:filename="annual-report.pdf"/>`, "report.pdf", false},
		{"inside CDATA", `<ac:plain-text-body><![CDATA[<ri:attachment ri:filename="a.png"/>]]></ac:plain-text-body>`, "a.png", false},
		{"another page's attachment", `<ri:attachment ri:filename="a.png"><ri:page ri:content-title="Other"/></ri:attachment>`, "a.png", false},
		{"blog post attachment", `<ri:attachment ri:filename="a.png"><ri:blog-post ri:content-title="Post"/></ri:attachment>`, "a.png", false},
		{"this page named explicitly", `<ri:attachment ri:filename="a.png"><ri:page ri:content-title="This Page"/></ri:attachment>`, "a.png", true},
		{"html entities and void elements", `<p>x&nbsp;y<br /></p><ri:attachment ri:filename="a.png"/>`, "a.png", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			refs, err := storageAttachmentRefs(tt.storage, "12345", "This Page")
			testutil.RequireNoError(t, err)
			testutil.Equal(t, refs.references(api.Attachment{Title: tt.filename}), tt.want)
		})
	}
}

func TestADFAttachmentRefs(t *testing.T) {
	t.Parallel()
	doc := `{"type":"doc","content":[
		{"type":"paragraph","content":[{"type":"text","text":"prose.txt"}]},
		{"type":"mediaSingle","content":[{"type":"media","attrs":{"id":"f-1","collection":"contentId-12345","type":"file"}}]},
		{"type":"mediaGroup","content":[{"type":"media","attrs":{"id":"f-2","collection":"contentId-12345","type":"file","__fileName":"named.pdf"}}]},
		{"type":"mediaSingle","content":[{"type":"media","attrs":{"id":"f-3","collection":"contentId-99999","type":"file","__fileName":"foreign.pdf"}}]},
		{"type":"mediaSingle","content":[{"type":"media","attrs":{"type":"external","url":"https://example.com/x.png","alt":"x.png"}}]},
		{"type":"paragraph","content":[{"type":"text","text":"download","marks":[{"type":"link","attrs":{"href":"/wiki/download/attachments/12345/linked%20file.zip"}}]}]}
	]}`
	refs, err := adfAttachmentRefs(doc, "12345")
	testutil.RequireNoError(t, err)

	testutil.True(t, refs.references(api.Attachment{Title: "by-id.png", FileID: "f-1"}))
	testutil.True(t, refs.references(api.Attachment{Title: "named.pdf"}))
	testutil.False(t, refs.references(api.Attachment{Title: "prose.txt", FileID: "f-9"}))
	testutil.False(t, refs.references(api.Attachment{Title: "foreign.pdf", FileID: "f-8"}))
	testutil.False(t, refs.references(api.Attachment{Title: "x.png"}))
	testutil.True(t, refs.references(api.Attachment{Title: "linked file.zip"}))
}

func TestPageAttachmentRefs_NoBodyIsAnError(t *testing.T) {
	t.Parallel()
	_, err := pageAttachmentRefs(&api.Page{ID: "12345", Body: &api.Body{}})
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "cannot determine attachment usage")
}
