package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"
)

func TestMutationPresenter_Success(t *testing.T) {
	t.Parallel()
	p := MutationPresenter{}
	model := p.Success("Created issue %s", "PROJ-123")

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}

	msg, ok := model.Sections[0].(*present.MessageSection)
	if !ok {
		t.Fatalf("expected MessageSection, got %T", model.Sections[0])
	}

	if msg.Kind != present.MessageSuccess {
		t.Errorf("expected MessageSuccess, got %v", msg.Kind)
	}
	if msg.Message != "Created issue PROJ-123" {
		t.Errorf("expected 'Created issue PROJ-123', got %q", msg.Message)
	}
	if msg.Stream != present.StreamStdout {
		t.Errorf("expected StreamStdout, got %v", msg.Stream)
	}
}

func TestMutationPresenter_Info(t *testing.T) {
	t.Parallel()
	p := MutationPresenter{}
	model := p.Info("Processing %d items", 5)

	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageInfo {
		t.Errorf("expected MessageInfo, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStdout {
		t.Errorf("expected StreamStdout, got %v", msg.Stream)
	}
}

func TestMutationPresenter_Advisory(t *testing.T) {
	t.Parallel()
	p := MutationPresenter{}
	model := p.Advisory("More results available")

	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageInfo {
		t.Errorf("expected MessageInfo kind, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStderr {
		t.Errorf("expected StreamStderr (advisory goes to stderr), got %v", msg.Stream)
	}
}

func TestMutationPresenter_Warning(t *testing.T) {
	t.Parallel()
	p := MutationPresenter{}
	model := p.Warning("Deprecated feature")

	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageWarning {
		t.Errorf("expected MessageWarning, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStderr {
		t.Errorf("expected StreamStderr, got %v", msg.Stream)
	}
}

func TestMutationPresenter_Error(t *testing.T) {
	t.Parallel()
	p := MutationPresenter{}
	model := p.Error("Connection failed")

	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageError {
		t.Errorf("expected MessageError, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStderr {
		t.Errorf("expected StreamStderr, got %v", msg.Stream)
	}
}
