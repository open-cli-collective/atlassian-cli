package md

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestProtectCodeRegions_FencedBlock(t *testing.T) {
	t.Parallel()
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
				testutil.Contains(t, output, "before\n")
				testutil.Contains(t, output, "after")
				testutil.NotContains(t, output, "[[My Page]]")
				testutil.Contains(t, regions[0].content, "[[My Page]]")
				testutil.Contains(t, regions[0].content, "```")
			},
		},
		{
			name:           "tilde fence",
			input:          "before\n~~~\n[[My Page]]\n~~~\nafter",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				testutil.NotContains(t, output, "[[My Page]]")
				testutil.Contains(t, regions[0].content, "[[My Page]]")
			},
		},
		{
			name:           "fence with language tag",
			input:          "before\n```markdown\nUse [[Page Title]] syntax\n```\nafter",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				testutil.NotContains(t, output, "[[Page Title]]")
				testutil.Contains(t, regions[0].content, "[[Page Title]]")
				testutil.Contains(t, regions[0].content, "```markdown")
			},
		},
		{
			name:           "no code block",
			input:          "See [[My Page]] for details",
			expectedRegion: 0,
			checkOutput: func(t *testing.T, output string, _ []codeRegion) {
				testutil.Equal(t, "See [[My Page]] for details", output)
			},
		},
		{
			name:           "multiple code blocks",
			input:          "```\n[[A]]\n```\ntext\n```\n[[B]]\n```",
			expectedRegion: 2,
			checkOutput: func(t *testing.T, output string, _ []codeRegion) {
				testutil.Contains(t, output, "text")
				testutil.NotContains(t, output, "[[A]]")
				testutil.NotContains(t, output, "[[B]]")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, regions := protectCodeRegions([]byte(tt.input))
			testutil.Equal(t, tt.expectedRegion, len(regions))
			tt.checkOutput(t, string(output), regions)
		})
	}
}

func TestProtectCodeRegions_InlineCode(t *testing.T) {
	t.Parallel()
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
				testutil.NotContains(t, output, "[[My Page]]")
				testutil.Contains(t, output, "Use ")
				testutil.Contains(t, output, " for links")
				testutil.Equal(t, "`[[My Page]]`", regions[0].content)
			},
		},
		{
			name:           "double backtick",
			input:          "Use ``[[My Page]]`` for links",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				testutil.NotContains(t, output, "[[My Page]]")
				testutil.Equal(t, "``[[My Page]]``", regions[0].content)
			},
		},
		{
			name:           "no inline code",
			input:          "See [[My Page]] here",
			expectedRegion: 0,
			checkOutput: func(t *testing.T, output string, _ []codeRegion) {
				testutil.Equal(t, "See [[My Page]] here", output)
			},
		},
		{
			name:           "mixed inline code and wiki link",
			input:          "Use `[[syntax]]` to link to [[Real Page]]",
			expectedRegion: 1,
			checkOutput: func(t *testing.T, output string, regions []codeRegion) {
				testutil.Contains(t, output, "[[Real Page]]")
				testutil.NotContains(t, output, "`[[syntax]]`")
				testutil.Equal(t, "`[[syntax]]`", regions[0].content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, regions := protectCodeRegions([]byte(tt.input))
			testutil.Equal(t, tt.expectedRegion, len(regions))
			tt.checkOutput(t, string(output), regions)
		})
	}
}

func TestProtectCodeRegions_Mixed(t *testing.T) {
	t.Parallel()
	input := "See [[Page A]] here.\n\n```\n[[Page B]] in code\n```\n\nAlso `[[Page C]]` inline.\n\nAnd [[Page D]] at end."

	output, regions := protectCodeRegions([]byte(input))
	outputStr := string(output)

	// Code regions should be protected
	testutil.Equal(t, 2, len(regions))
	testutil.NotContains(t, outputStr, "[[Page B]]")
	testutil.NotContains(t, outputStr, "[[Page C]]")

	// Non-code wiki links should remain
	testutil.Contains(t, outputStr, "[[Page A]]")
	testutil.Contains(t, outputStr, "[[Page D]]")
}

func TestRestoreCodeRegions(t *testing.T) {
	t.Parallel()
	// Simulate a protect → transform → restore cycle
	input := "before\n```\n[[My Page]]\n```\nafter [[Link]]"

	protected, regions := protectCodeRegions([]byte(input))

	// Simulate wiki-link replacement on the non-code parts
	protectedStr := string(protected)
	testutil.Contains(t, protectedStr, "[[Link]]")

	// Restore
	restored := restoreCodeRegions(protected, regions)
	testutil.Equal(t, input, string(restored))
}

func TestProtectCodeRegions_UnclosedFence(t *testing.T) {
	t.Parallel()
	// Unclosed fence should protect to end of input
	input := "before\n```\n[[My Page]]\nno closing fence"
	output, regions := protectCodeRegions([]byte(input))
	testutil.Equal(t, 1, len(regions))
	testutil.Contains(t, string(output), "before\n")
	testutil.NotContains(t, string(output), "[[My Page]]")
}

func TestProtectCodeRegions_UnmatchedBacktick(t *testing.T) {
	t.Parallel()
	// Unmatched backtick should not swallow content
	input := "text `unclosed [[My Page]]"
	output, regions := protectCodeRegions([]byte(input))
	testutil.Equal(t, 0, len(regions))
	testutil.Equal(t, input, string(output))
}
