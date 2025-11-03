package bubbletea

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
	model Model
}

// NewProgram creates a new Program for the provided model.
func NewProgram(model Model) *Program {
	return &Program{model: model}
}

// Run executes the program. The stub simply runs the initial command, if any,
// and returns the model without starting an interactive session.
func (p *Program) Run() (Model, error) {
	if cmd := p.model.Init(); cmd != nil {
		if msg := cmd(); msg != nil {
			p.model, _ = p.model.Update(msg)
		}
	}
	return p.model, nil
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
