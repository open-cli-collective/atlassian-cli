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

	got := Render(model, StyleAgent)
	want := "Name: Alice\nID: 123\n"
	if got != want {
		t.Errorf("detail agent:\ngot:\n%s\nwant:\n%s", got, want)
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

	got := Render(model, StyleHuman)
	want := "Name: Alice\nID: 123\n"
	if got != want {
		t.Errorf("detail human:\ngot:\n%s\nwant:\n%s", got, want)
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

	got := Render(model, StyleAgent)
	want := "KEY | SUMMARY\nPROJ-1 | First issue\nPROJ-2 | Second issue\n"
	if got != want {
		t.Errorf("table agent:\ngot:\n%s\nwant:\n%s", got, want)
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

	got := Render(model, StyleHuman)
	// Human style uses tabwriter - verify structure, not exact spacing
	if !strings.Contains(got, "KEY") || !strings.Contains(got, "SUMMARY") {
		t.Errorf("missing headers in output: %s", got)
	}
	if !strings.Contains(got, "PROJ-1") || !strings.Contains(got, "First") {
		t.Errorf("missing row 1 in output: %s", got)
	}
	if !strings.Contains(got, "PROJ-2") || !strings.Contains(got, "Second") {
		t.Errorf("missing row 2 in output: %s", got)
	}
}

func TestRender_MessageSection_Success_Agent(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageSuccess, Message: "Issue updated"},
		},
	}

	got := Render(model, StyleAgent)
	want := "Issue updated\n"
	if got != want {
		t.Errorf("message agent:\ngot: %q\nwant: %q", got, want)
	}
}

func TestRender_MessageSection_Success_Human(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageSuccess, Message: "Issue updated"},
		},
	}

	got := Render(model, StyleHuman)
	want := "✓ Issue updated\n"
	if got != want {
		t.Errorf("message human:\ngot: %q\nwant: %q", got, want)
	}
}

func TestRender_MessageSection_Warning_Human(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageWarning, Message: "Deprecated API"},
		},
	}

	got := Render(model, StyleHuman)
	want := "⚠ Deprecated API\n"
	if got != want {
		t.Errorf("message warning human:\ngot: %q\nwant: %q", got, want)
	}
}

func TestRender_MessageSection_Info_Human(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&MessageSection{Kind: MessageInfo, Message: "Processing..."},
		},
	}

	got := Render(model, StyleHuman)
	want := "Processing...\n"
	if got != want {
		t.Errorf("message info human:\ngot: %q\nwant: %q", got, want)
	}
}

func TestRender_MixedSections(t *testing.T) {
	t.Parallel()
	model := &OutputModel{
		Sections: []Section{
			&DetailSection{
				Fields: []Field{{Label: "ID", Value: "123"}},
			},
			&TableSection{
				Headers: []string{"NAME", "VALUE"},
				Rows:    []Row{{Cells: []string{"foo", "bar"}}},
			},
		},
	}

	got := Render(model, StyleAgent)
	want := "ID: 123\nNAME | VALUE\nfoo | bar\n"
	if got != want {
		t.Errorf("mixed agent:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRender_EmptyModel(t *testing.T) {
	t.Parallel()
	model := &OutputModel{Sections: []Section{}}
	got := Render(model, StyleAgent)
	if got != "" {
		t.Errorf("empty model should produce empty string, got: %q", got)
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

	got := Render(model, StyleAgent)
	want := "KEY | SUMMARY\n"
	if got != want {
		t.Errorf("empty table:\ngot: %q\nwant: %q", got, want)
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

	// Both styles should render the message without decorators for unknown kinds
	gotAgent := Render(model, StyleAgent)
	wantAgent := "Unknown kind\n"
	if gotAgent != wantAgent {
		t.Errorf("unknown kind agent:\ngot: %q\nwant: %q", gotAgent, wantAgent)
	}

	gotHuman := Render(model, StyleHuman)
	wantHuman := "Unknown kind\n"
	if gotHuman != wantHuman {
		t.Errorf("unknown kind human:\ngot: %q\nwant: %q", gotHuman, wantHuman)
	}
}
