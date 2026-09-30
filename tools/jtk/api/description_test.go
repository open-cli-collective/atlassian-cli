package api

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func mediaNode(typ, id string) *ADFNode {
	return &ADFNode{Type: typ, Content: []*ADFNode{{Type: "media", Attrs: map[string]any{"id": id, "type": "file"}}}}
}

func paragraph(text string) *ADFNode {
	return &ADFNode{Type: "paragraph", Content: []*ADFNode{{Type: "text", Text: text}}}
}

func TestPreserveDescriptionMedia(t *testing.T) {
	t.Parallel()

	t.Run("no existing description", func(t *testing.T) {
		t.Parallel()
		doc := &ADFDocument{Type: "doc", Version: 1, Content: []*ADFNode{paragraph("new")}}
		got := PreserveDescriptionMedia(doc, nil)
		testutil.Len(t, got.Content, 1)
	})

	t.Run("plain-text existing description", func(t *testing.T) {
		t.Parallel()
		doc := &ADFDocument{Type: "doc", Version: 1, Content: []*ADFNode{paragraph("new")}}
		got := PreserveDescriptionMedia(doc, &Description{Text: "old"})
		testutil.Len(t, got.Content, 1)
	})

	t.Run("existing description without media", func(t *testing.T) {
		t.Parallel()
		doc := &ADFDocument{Type: "doc", Version: 1, Content: []*ADFNode{paragraph("new")}}
		existing := &Description{ADF: &ADFDocument{Type: "doc", Version: 1, Content: []*ADFNode{paragraph("old")}}}
		got := PreserveDescriptionMedia(doc, existing)
		testutil.Len(t, got.Content, 1)
	})

	t.Run("top-level and nested media appended in order", func(t *testing.T) {
		t.Parallel()
		doc := &ADFDocument{Type: "doc", Version: 1, Content: []*ADFNode{paragraph("new")}}
		panel := &ADFNode{Type: "panel", Content: []*ADFNode{mediaNode("mediaGroup", "b")}}
		existing := &Description{ADF: &ADFDocument{Type: "doc", Version: 1, Content: []*ADFNode{
			paragraph("old"), mediaNode("mediaSingle", "a"), panel,
		}}}
		got := PreserveDescriptionMedia(doc, existing)
		testutil.RequireEqual(t, len(got.Content), 3)
		testutil.Equal(t, got.Content[0].Content[0].Text, "new")
		testutil.Equal(t, got.Content[1].Type, "mediaSingle")
		testutil.Equal(t, got.Content[1].Content[0].Attrs["id"], "a")
		testutil.Equal(t, got.Content[2].Type, "mediaGroup")
		testutil.Equal(t, got.Content[2].Content[0].Attrs["id"], "b")
	})

	t.Run("nil new document still keeps media", func(t *testing.T) {
		t.Parallel()
		existing := &Description{ADF: &ADFDocument{Type: "doc", Version: 1, Content: []*ADFNode{mediaNode("mediaSingle", "a")}}}
		got := PreserveDescriptionMedia(nil, existing)
		testutil.Equal(t, got.Type, "doc")
		testutil.Equal(t, got.Version, 1)
		testutil.Len(t, got.Content, 1)
	})
}
