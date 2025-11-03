package bubbletea

import (
	"bytes"
	"strings"
	"testing"
)

type testModel struct {
	view     string
	quitting bool
}

func (m *testModel) Init() Cmd {
	return nil
}

func (m *testModel) Update(msg Msg) (Model, Cmd) {
	if key, ok := msg.(KeyMsg); ok {
		if key.Value == "q" || key.Value == "ctrl+c" {
			m.quitting = true
			return m, Quit
		}
	}
	return m, nil
}

func (m *testModel) View() string {
	if m.quitting {
		return ""
	}
	return m.view
}

func TestProgramRunRendersAndQuitsOnQ(t *testing.T) {
	model := &testModel{view: "hello"}
	program := NewProgram(model)
	program.reader = strings.NewReader("q\n")
	var output bytes.Buffer
	program.writer = &output

	finalModel, err := program.Run()
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	typed, ok := finalModel.(*testModel)
	if !ok {
		t.Fatalf("expected *testModel, got %T", finalModel)
	}
	if !typed.quitting {
		t.Fatal("expected model to be in quitting state")
	}

	if got := output.String(); !strings.Contains(got, "hello") {
		t.Fatalf("expected output to contain view, got %q", got)
	}
}
