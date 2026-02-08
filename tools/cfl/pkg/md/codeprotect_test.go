package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtectCodeRegions_FencedBlock(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedRegion int
		checkOutput    func(t *testing.T, output string, regions []codeRegion)
	}{
		{
			name:           "backtick fence",
			input:          "before\n```\n[[My Page]]\n```\nafter",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.Contains(t, output, "before\n")
				assert.Contains(t, output, "after")
				assert.NotContains(t, output, "[[My Page]]")
				assert.Contains(t, regions[0].content, "[[My Page]]")
				assert.Contains(t, regions[0].content, "```")
			},
		},
		{
			name:           "tilde fence",
			input:          "before\n~~~\n[[My Page]]\n~~~\nafter",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.NotContains(t, output, "[[My Page]]")
				assert.Contains(t, regions[0].content, "[[My Page]]")
			},
		},
		{
			name:           "fence with language tag",
			input:          "before\n```markdown\nUse [[Page Title]] syntax\n```\nafter",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.NotContains(t, output, "[[Page Title]]")
				assert.Contains(t, regions[0].content, "[[Page Title]]")
				assert.Contains(t, regions[0].content, "```markdown")
			},
		},
		{
			name:           "no code block",
			input:          "See [[My Page]] for details",
			expectedRegion: 0,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.Equal(t, "See [[My Page]] for details", output)
			},
		},
		{
			name:           "multiple code blocks",
			input:          "```\n[[A]]\n```\ntext\n```\n[[B]]\n```",
			expectedRegion: 2,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.Contains(t, output, "text")
				assert.NotContains(t, output, "[[A]]")
				assert.NotContains(t, output, "[[B]]")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, regions := protectCodeRegions([]byte(tt.input))
			assert.Equal(t, tt.expectedRegion, len(regions))
			tt.checkOutput(t, string(output), regions)
		})
	}
}

func TestProtectCodeRegions_InlineCode(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedRegion int
		checkOutput    func(t *testing.T, output string, regions []codeRegion)
	}{
		{
			name:           "single backtick",
			input:          "Use `[[My Page]]` for links",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.NotContains(t, output, "[[My Page]]")
				assert.Contains(t, output, "Use ")
				assert.Contains(t, output, " for links")
				assert.Equal(t, "`[[My Page]]`", regions[0].content)
			},
		},
		{
			name:           "double backtick",
			input:          "Use ``[[My Page]]`` for links",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.NotContains(t, output, "[[My Page]]")
				assert.Equal(t, "``[[My Page]]``", regions[0].content)
			},
		},
		{
			name:           "no inline code",
			input:          "See [[My Page]] here",
			expectedRegion: 0,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.Equal(t, "See [[My Page]] here", output)
			},
		},
		{
			name:           "mixed inline code and wiki link",
			input:          "Use `[[syntax]]` to link to [[Real Page]]",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				assert.Contains(t, output, "[[Real Page]]")
				assert.NotContains(t, output, "`[[syntax]]`")
				assert.Equal(t, "`[[syntax]]`", regions[0].content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, regions := protectCodeRegions([]byte(tt.input))
			assert.Equal(t, tt.expectedRegion, len(regions))
			tt.checkOutput(t, string(output), regions)
		})
	}
}

func TestProtectCodeRegions_Mixed(t *testing.T) {
	input := "See [[Page A]] here.\n\n```\n[[Page B]] in code\n```\n\nAlso `[[Page C]]` inline.\n\nAnd [[Page D]] at end."

	output, regions := protectCodeRegions([]byte(input))
	outputStr := string(output)

	// Code regions should be protected
	assert.Equal(t, 2, len(regions))
	assert.NotContains(t, outputStr, "[[Page B]]")
	assert.NotContains(t, outputStr, "[[Page C]]")

	// Non-code wiki links should remain
	assert.Contains(t, outputStr, "[[Page A]]")
	assert.Contains(t, outputStr, "[[Page D]]")
}

func TestRestoreCodeRegions(t *testing.T) {
	// Simulate a protect → transform → restore cycle
	input := "before\n```\n[[My Page]]\n```\nafter [[Link]]"

	protected, regions := protectCodeRegions([]byte(input))

	// Simulate wiki-link replacement on the non-code parts
	protectedStr := string(protected)
	assert.Contains(t, protectedStr, "[[Link]]")

	// Restore
	restored := restoreCodeRegions(protected, regions)
	assert.Equal(t, input, string(restored))
}

func TestProtectCodeRegions_UnclosedFence(t *testing.T) {
	// Unclosed fence should protect to end of input
	input := "before\n```\n[[My Page]]\nno closing fence"
	output, regions := protectCodeRegions([]byte(input))
	assert.Equal(t, 1, len(regions))
	assert.Contains(t, string(output), "before\n")
	assert.NotContains(t, string(output), "[[My Page]]")
}

func TestProtectCodeRegions_UnmatchedBacktick(t *testing.T) {
	// Unmatched backtick should not swallow content
	input := "text `unclosed [[My Page]]"
	output, regions := protectCodeRegions([]byte(input))
	assert.Equal(t, 0, len(regions))
	assert.Equal(t, input, string(output))
}
