package bubbletea

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

// Msg represents a message delivered to the TUI update loop.
type Msg interface{}

// Cmd is a function that may produce a message.
type Cmd func() Msg

// Model is the interface TUI models must implement.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}

// Program drives execution of a TUI model.
type Program struct {
	model  Model
	reader io.RuneReader
	writer io.Writer
}

// NewProgram creates a new Program for the provided model.
func NewProgram(model Model) *Program {
	return &Program{
		model:  model,
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// Run executes the program. The implementation is intentionally minimal but
// provides enough behaviour to render the view and react to keyboard input so
// the application can be exercised interactively.
func (p *Program) Run() (Model, error) {
	if quit, err := p.executeCmd(p.model.Init()); err != nil {
		return p.model, err
	} else if quit {
		return p.model, nil
	}

	// Render once before entering the event loop.
	p.render()

	inputs := make(chan Msg)
	go p.readInput(inputs)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	defer signal.Stop(sigs)

	for {
		var (
			msg Msg
			ok  bool
		)

		select {
		case msg, ok = <-inputs:
			if !ok {
				return p.model, nil
			}
		case <-sigs:
			msg = KeyMsg{Value: "ctrl+c"}
		}

		if err, ok := msg.(errMsg); ok {
			return p.model, err.err
		}

		var cmd Cmd
		p.model, cmd = p.model.Update(msg)
		if quit, err := p.executeCmd(cmd); err != nil {
			return p.model, err
		} else if quit {
			return p.model, nil
		}

		p.render()
	}
}

func (p *Program) render() {
	view := p.model.View()
	if view == "" {
		// Nothing to render (e.g. quitting state)
		return
	}
	fmt.Fprint(p.writer, "\033[H\033[2J")
	fmt.Fprint(p.writer, view)
}

func (p *Program) readInput(ch chan<- Msg) {
	defer close(ch)
	for {
		r, _, err := p.reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			ch <- errMsg{err: err}
			return
		}
		if r == '\n' || r == '\r' {
			continue
		}
		ch <- KeyMsg{Value: string(r)}
	}
}

func (p *Program) executeCmd(cmd Cmd) (bool, error) {
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return false, nil
		}
		if err, ok := msg.(errMsg); ok {
			// Surface command errors to the caller.
			return false, err.err
		}
		if _, ok := msg.(quitMsg); ok {
			return true, nil
		}
		var next Cmd
		p.model, next = p.model.Update(msg)
		cmd = next
	}
	return false, nil
}

// errMsg is emitted when a command reports an error. It mirrors the
// behaviour Bubble Tea exposes so Update handlers can branch on it.
type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

// KeyMsg represents a keyboard event.
type KeyMsg struct {
	Value string
}

// String returns the textual representation of the key event.
func (k KeyMsg) String() string { return k.Value }

// quitMsg is emitted when Quit is invoked.
type quitMsg struct{}

// Quit terminates the program when returned from Update.
var Quit Cmd = func() Msg {
	return quitMsg{}
}
