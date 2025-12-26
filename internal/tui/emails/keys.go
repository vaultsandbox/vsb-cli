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
	Refresh   key.Binding
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
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
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
		key.WithKeys("left"),
		key.WithHelp("←", "prev inbox"),
	),
	NextInbox: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next inbox"),
	),
	NewInbox: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new inbox"),
	),
}
