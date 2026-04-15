package present

import (
	"strings"
	"testing"
)

func TestRender_DetailSection_Agent(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&DetailSection{
				Fields: []Field{
					{Label: "Name", Value: "Alice"},
					{Label: "ID", Value: "123"},
				},
			},
		},
	}

	out := Render(model, StyleAgent)
	want := "Name: Alice\nID: 123\n"
	if out.Stdout != want {
		t.Errorf("detail agent stdout:\ngot:\n%s\nwant:\n%s", out.Stdout, want)
	}
	if out.Stderr != "" {
		t.Errorf("detail agent stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_DetailSection_Human(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&DetailSection{
				Fields: []Field{
					{Label: "Name", Value: "Alice"},
					{Label: "ID", Value: "123"},
				},
			},
		},
	}

	out := Render(model, StyleHuman)
	want := "Name: Alice\nID: 123\n"
	if out.Stdout != want {
		t.Errorf("detail human stdout:\ngot:\n%s\nwant:\n%s", out.Stdout, want)
	}
	if out.Stderr != "" {
		t.Errorf("detail human stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_TableSection_Agent(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&TableSection{
				Headers: []string{"KEY", "SUMMARY"},
				Rows: []Row{
					{Cells: []string{"PROJ-1", "First issue"}},
					{Cells: []string{"PROJ-2", "Second issue"}},
				},
			},
		},
	}

	out := Render(model, StyleAgent)
	want := "KEY | SUMMARY\nPROJ-1 | First issue\nPROJ-2 | Second issue\n"
	if out.Stdout != want {
		t.Errorf("table agent stdout:\ngot:\n%s\nwant:\n%s", out.Stdout, want)
	}
	if out.Stderr != "" {
		t.Errorf("table agent stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_TableSection_Human(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&TableSection{
				Headers: []string{"KEY", "SUMMARY"},
				Rows: []Row{
					{Cells: []string{"PROJ-1", "First"}},
					{Cells: []string{"PROJ-2", "Second"}},
				},
			},
		},
	}

	out := Render(model, StyleHuman)
	// Human style uses tabwriter - verify structure, not exact spacing
	if !strings.Contains(out.Stdout, "KEY") || !strings.Contains(out.Stdout, "SUMMARY") {
		t.Errorf("missing headers in stdout: %s", out.Stdout)
	}
	if !strings.Contains(out.Stdout, "PROJ-1") || !strings.Contains(out.Stdout, "First") {
		t.Errorf("missing row 1 in stdout: %s", out.Stdout)
	}
	if !strings.Contains(out.Stdout, "PROJ-2") || !strings.Contains(out.Stdout, "Second") {
		t.Errorf("missing row 2 in stdout: %s", out.Stdout)
	}
	if out.Stderr != "" {
		t.Errorf("table human stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_MessageSection_Success_Agent(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageSuccess, Message: "Issue updated"},
		},
	}

	out := Render(model, StyleAgent)
	want := "Issue updated\n"
	if out.Stdout != want {
		t.Errorf("message success agent stdout:\ngot: %q\nwant: %q", out.Stdout, want)
	}
	if out.Stderr != "" {
		t.Errorf("message success agent stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_MessageSection_Success_Human(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageSuccess, Message: "Issue updated"},
		},
	}

	out := Render(model, StyleHuman)
	want := "✓ Issue updated\n"
	if out.Stdout != want {
		t.Errorf("message success human stdout:\ngot: %q\nwant: %q", out.Stdout, want)
	}
	if out.Stderr != "" {
		t.Errorf("message success human stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_MessageSection_Warning_GoesToStderr(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageWarning, Message: "Deprecated API", Stream: StreamStderr},
		},
	}

	// Agent style - warning goes to stderr
	outAgent := Render(model, StyleAgent)
	if outAgent.Stdout != "" {
		t.Errorf("warning agent stdout should be empty, got: %q", outAgent.Stdout)
	}
	if outAgent.Stderr != "Deprecated API\n" {
		t.Errorf("warning agent stderr:\ngot: %q\nwant: %q", outAgent.Stderr, "Deprecated API\n")
	}

	// Human style - warning goes to stderr with decorator
	outHuman := Render(model, StyleHuman)
	if outHuman.Stdout != "" {
		t.Errorf("warning human stdout should be empty, got: %q", outHuman.Stdout)
	}
	if outHuman.Stderr != "⚠ Deprecated API\n" {
		t.Errorf("warning human stderr:\ngot: %q\nwant: %q", outHuman.Stderr, "⚠ Deprecated API\n")
	}
}

func TestRender_MessageSection_Info_Human(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageInfo, Message: "Processing..."},
		},
	}

	out := Render(model, StyleHuman)
	want := "Processing...\n"
	if out.Stdout != want {
		t.Errorf("message info human stdout:\ngot: %q\nwant: %q", out.Stdout, want)
	}
	if out.Stderr != "" {
		t.Errorf("message info human stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_MixedSections_WithWarning(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&DetailSection{
				Fields: []Field{{Label: "ID", Value: "123"}},
			},
			&MessageSection{Kind: MessageWarning, Message: "Field deprecated", Stream: StreamStderr},
			&TableSection{
				Headers: []string{"NAME", "VALUE"},
				Rows:    []Row{{Cells: []string{"foo", "bar"}}},
			},
		},
	}

	out := Render(model, StyleAgent)
	wantStdout := "ID: 123\nNAME | VALUE\nfoo | bar\n"
	wantStderr := "Field deprecated\n"
	if out.Stdout != wantStdout {
		t.Errorf("mixed stdout:\ngot:\n%s\nwant:\n%s", out.Stdout, wantStdout)
	}
	if out.Stderr != wantStderr {
		t.Errorf("mixed stderr:\ngot:\n%s\nwant:\n%s", out.Stderr, wantStderr)
	}
}

func TestRender_EmptyModel(t *testing.T) {
	t.Parallel()
	model := &OutputModel{Sections: []Section{}}
	out := Render(model, StyleAgent)
	if out.Stdout != "" {
		t.Errorf("empty model stdout should be empty, got: %q", out.Stdout)
	}
	if out.Stderr != "" {
		t.Errorf("empty model stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_NilModel(t *testing.T) {
	t.Parallel()
	out := Render(nil, StyleAgent)
	if out.Stdout != "" {
		t.Errorf("nil model stdout should be empty, got: %q", out.Stdout)
	}
	if out.Stderr != "" {
		t.Errorf("nil model stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_EmptyTable(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&TableSection{
				Headers: []string{"KEY", "SUMMARY"},
				Rows:    []Row{},
			},
		},
	}

	out := Render(model, StyleAgent)
	want := "KEY | SUMMARY\n"
	if out.Stdout != want {
		t.Errorf("empty table stdout:\ngot: %q\nwant: %q", out.Stdout, want)
	}
	if out.Stderr != "" {
		t.Errorf("empty table stderr should be empty, got: %q", out.Stderr)
	}
}

func TestRender_MessageSection_UnknownKind(t *testing.T) {
	t.Parallel()
	// Test that unknown MessageKind values fall through gracefully
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageKind(99), Message: "Unknown kind"},
		},
	}

	// Unknown kinds go to stdout (not stderr)
	outAgent := Render(model, StyleAgent)
	if outAgent.Stdout != "Unknown kind\n" {
		t.Errorf("unknown kind agent stdout:\ngot: %q\nwant: %q", outAgent.Stdout, "Unknown kind\n")
	}
	if outAgent.Stderr != "" {
		t.Errorf("unknown kind agent stderr should be empty, got: %q", outAgent.Stderr)
	}

	outHuman := Render(model, StyleHuman)
	if outHuman.Stdout != "Unknown kind\n" {
		t.Errorf("unknown kind human stdout:\ngot: %q\nwant: %q", outHuman.Stdout, "Unknown kind\n")
	}
	if outHuman.Stderr != "" {
		t.Errorf("unknown kind human stderr should be empty, got: %q", outHuman.Stderr)
	}
}

func TestStyleFromMode(t *testing.T) {
	t.Parallel()
	if got := StyleFromMode(RenderModeAgent); got != StyleAgent {
		t.Errorf("StyleFromMode(RenderModeAgent) = %v, want %v", got, StyleAgent)
	}
	if got := StyleFromMode(RenderModeHuman); got != StyleHuman {
		t.Errorf("StyleFromMode(RenderModeHuman) = %v, want %v", got, StyleHuman)
	}
}

func TestRender_TableSection_Agent_EscapesPipes(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&TableSection{
				Headers: []string{"KEY", "SUMMARY"},
				Rows: []Row{
					{Cells: []string{"PROJ-1", "Fix A | B pipeline"}},
				},
			},
		},
	}

	out := Render(model, StyleAgent)
	// Pipes in cell content should be escaped to avoid delimiter confusion
	want := "KEY | SUMMARY\nPROJ-1 | Fix A \\| B pipeline\n"
	if out.Stdout != want {
		t.Errorf("pipe escaping:\ngot:\n%s\nwant:\n%s", out.Stdout, want)
	}
}

func TestRender_TableSection_Agent_NormalizesNewlines(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&TableSection{
				Headers: []string{"KEY", "DESC"},
				Rows: []Row{
					{Cells: []string{"PROJ-1", "Line one\nLine two"}},
				},
			},
		},
	}

	out := Render(model, StyleAgent)
	// Newlines in cell content should be normalized to spaces
	want := "KEY | DESC\nPROJ-1 | Line one Line two\n"
	if out.Stdout != want {
		t.Errorf("newline normalization:\ngot:\n%s\nwant:\n%s", out.Stdout, want)
	}
}

func TestRender_TableSection_Human_NoEscaping(t *testing.T) {
	t.Parallel()
	// Human mode should NOT escape pipes - verifies style-agnostic presenters
	model := &OutputModel{
		Sections: []Section{
			&TableSection{
				Headers: []string{"KEY", "DESC"},
				Rows: []Row{
					{Cells: []string{"PROJ-1", "A | B"}},
				},
			},
		},
	}

	out := Render(model, StyleHuman)
	// Human mode passes through raw content (tabwriter handles alignment)
	if strings.Contains(out.Stdout, "\\|") {
		t.Errorf("human mode should not escape pipes, got: %s", out.Stdout)
	}
	if !strings.Contains(out.Stdout, "A | B") {
		t.Errorf("human mode should preserve raw pipe character, got: %s", out.Stdout)
	}
}

func TestRender_MessageSection_Error(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageError, Message: "Connection failed", Stream: StreamStderr},
		},
	}

	// Agent style - error goes to stderr, no decorator
	outAgent := Render(model, StyleAgent)
	if outAgent.Stdout != "" {
		t.Errorf("error agent stdout should be empty, got: %q", outAgent.Stdout)
	}
	if outAgent.Stderr != "Connection failed\n" {
		t.Errorf("error agent stderr:\ngot: %q\nwant: %q", outAgent.Stderr, "Connection failed\n")
	}

	// Human style - error goes to stderr with ✗ decorator
	outHuman := Render(model, StyleHuman)
	if outHuman.Stdout != "" {
		t.Errorf("error human stdout should be empty, got: %q", outHuman.Stdout)
	}
	if outHuman.Stderr != "✗ Connection failed\n" {
		t.Errorf("error human stderr:\ngot: %q\nwant: %q", outHuman.Stderr, "✗ Connection failed\n")
	}
}

func TestRender_Stream_ExplicitRouting(t *testing.T) {
	t.Parallel()
	// Test that Stream field controls routing, not Kind
	model := &OutputModel{
		Sections: []Section{
			// Info message explicitly routed to stderr (advisory)
			&MessageSection{Kind: MessageInfo, Message: "More results available", Stream: StreamStderr},
			// Success message to stdout (default)
			&MessageSection{Kind: MessageSuccess, Message: "Operation completed", Stream: StreamStdout},
		},
	}

	out := Render(model, StyleAgent)
	if out.Stdout != "Operation completed\n" {
		t.Errorf("explicit stdout routing:\ngot: %q\nwant: %q", out.Stdout, "Operation completed\n")
	}
	if out.Stderr != "More results available\n" {
		t.Errorf("explicit stderr routing:\ngot: %q\nwant: %q", out.Stderr, "More results available\n")
	}
}
