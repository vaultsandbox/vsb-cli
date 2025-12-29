package emails

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Back      key.Binding
	OpenURL   key.Binding
	ViewHTML  key.Binding
	Delete    key.Binding
	Quit      key.Binding
	Help      key.Binding
	PrevInbox key.Binding
	NextInbox key.Binding
	NewInbox  key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view email"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	OpenURL: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open url"),
	),
	ViewHTML: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view html"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	PrevInbox: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "prev inbox"),
	),
	NextInbox: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "next inbox"),
	),
	NewInbox: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new inbox"),
	),
}
