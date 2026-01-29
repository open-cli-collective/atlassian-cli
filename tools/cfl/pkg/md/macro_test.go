package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMacroRegistry_ContainsExpectedMacros(t *testing.T) {
	expectedMacros := []string{"toc", "info", "warning", "note", "tip", "expand", "code"}

	for _, name := range expectedMacros {
		t.Run(name, func(t *testing.T) {
			mt, ok := MacroRegistry[name]
			assert.True(t, ok, "MacroRegistry should contain %q", name)
			assert.Equal(t, name, mt.Name)
		})
	}
}

func TestLookupMacro_CaseInsensitive(t *testing.T) {
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
			mt, ok := LookupMacro(tt.input)
			assert.Equal(t, tt.found, ok)
			if tt.found {
				assert.Equal(t, tt.expected, mt.Name)
			}
		})
	}
}

func TestMacroType_BodyConfiguration(t *testing.T) {
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
			mt, ok := MacroRegistry[tt.name]
			assert.True(t, ok)
			assert.Equal(t, tt.hasBody, mt.HasBody)
			assert.Equal(t, tt.bodyType, mt.BodyType)
		})
	}
}

func TestMacroNode_Construction(t *testing.T) {
	// Test basic construction
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Important"},
		Body:       "This is the content",
		Children:   nil,
	}

	assert.Equal(t, "info", node.Name)
	assert.Equal(t, "Important", node.Parameters["title"])
	assert.Equal(t, "This is the content", node.Body)
	assert.Nil(t, node.Children)
}

func TestMacroNode_WithChildren(t *testing.T) {
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

	assert.Equal(t, "expand", parent.Name)
	assert.Len(t, parent.Children, 1)
	assert.Equal(t, "code", parent.Children[0].Name)
	assert.Equal(t, "go", parent.Children[0].Parameters["language"])
}
