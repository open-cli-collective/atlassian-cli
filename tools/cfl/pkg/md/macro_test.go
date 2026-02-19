package md

import (
	"fmt"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestMacroRegistry_ContainsExpectedMacros(t *testing.T) {
	t.Parallel()
	expectedMacros := []string{"toc", "info", "warning", "note", "tip", "expand", "code"}

	for _, name := range expectedMacros {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mt, ok := MacroRegistry[name]
			testutil.True(t, ok, fmt.Sprintf("MacroRegistry should contain %q", name))
			testutil.Equal(t, name, mt.Name)
		})
	}
}

func TestLookupMacro_CaseInsensitive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
		found    bool
	}{
		{"toc", "toc", true},
		{"TOC", "toc", true},
		{"Toc", "toc", true},
		{"INFO", "info", true},
		{"Info", "info", true},
		{"unknown", "", false},
		{"UNKNOWN", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			mt, ok := LookupMacro(tt.input)
			testutil.Equal(t, tt.found, ok)
			if tt.found {
				testutil.Equal(t, tt.expected, mt.Name)
			}
		})
	}
}

func TestMacroType_BodyConfiguration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		hasBody  bool
		bodyType BodyType
	}{
		{"toc", false, BodyTypeNone},
		{"info", true, BodyTypeRichText},
		{"warning", true, BodyTypeRichText},
		{"note", true, BodyTypeRichText},
		{"tip", true, BodyTypeRichText},
		{"expand", true, BodyTypeRichText},
		{"code", true, BodyTypePlainText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mt, ok := MacroRegistry[tt.name]
			testutil.True(t, ok)
			testutil.Equal(t, tt.hasBody, mt.HasBody)
			testutil.Equal(t, tt.bodyType, mt.BodyType)
		})
	}
}

func TestMacroNode_Construction(t *testing.T) {
	t.Parallel()
	// Test basic construction
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Important"},
		Body:       "This is the content",
		Children:   nil,
	}

	testutil.Equal(t, "info", node.Name)
	testutil.Equal(t, "Important", node.Parameters["title"])
	testutil.Equal(t, "This is the content", node.Body)
	testutil.Nil(t, node.Children)
}

func TestMacroNode_WithChildren(t *testing.T) {
	t.Parallel()
	// Test nested structure
	child := &MacroNode{
		Name:       "code",
		Parameters: map[string]string{"language": "go"},
		Body:       "fmt.Println(\"hello\")",
	}

	parent := &MacroNode{
		Name:       "expand",
		Parameters: map[string]string{"title": "Show code"},
		Body:       "",
		Children:   []*MacroNode{child},
	}

	testutil.Equal(t, "expand", parent.Name)
	testutil.Len(t, parent.Children, 1)
	testutil.Equal(t, "code", parent.Children[0].Name)
	testutil.Equal(t, "go", parent.Children[0].Parameters["language"])
}
