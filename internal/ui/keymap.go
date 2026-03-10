package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// GlobalKeyMap holds keybindings available in all contexts.
type GlobalKeyMap struct {
	Quit   key.Binding
	Help   key.Binding
	Filter key.Binding
}

// MainKeyMap holds keybindings for the main todo list view.
type MainKeyMap struct {
	CycleScope    key.Binding
	PrevContext   key.Binding
	NextContext   key.Binding
	MoveUp        key.Binding
	MoveDown      key.Binding
	Add           key.Binding
	AddChild      key.Binding
	Edit          key.Binding
	PriorityHigh  key.Binding
	PriorityMed   key.Binding
	PriorityLow   key.Binding
	PriorityClear key.Binding
	ToggleBlocked key.Binding
	ToggleReview  key.Binding
	TagPicker     key.Binding
	TagManager    key.Binding
	ToggleDone    key.Binding
	Delete        key.Binding
}

// TagPickerKeyMap holds keybindings for the tag picker overlay.
type TagPickerKeyMap struct {
	Close     key.Binding
	MoveUp    key.Binding
	MoveDown  key.Binding
	Toggle    key.Binding
	DeleteTag key.Binding
	NewTag    key.Binding
}

// AddKeyMap holds keybindings for add/edit mode.
type AddKeyMap struct {
	Cancel  key.Binding
	Confirm key.Binding
}

// FilterKeyMap holds keybindings for filter input mode.
type FilterKeyMap struct {
	Cancel key.Binding
	Apply  key.Binding
}

// QuickAddKeyMap holds keybindings for the quick-add popup.
type QuickAddKeyMap struct {
	Cancel key.Binding
	Save   key.Binding
}

// HelpKeyMap holds keybindings for the help screen.
type HelpKeyMap struct {
	Close key.Binding
}

// KeyMap is the top-level keybinding map composed of context-specific sub-keymaps.
type KeyMap struct {
	Global    GlobalKeyMap
	Main      MainKeyMap
	TagPicker TagPickerKeyMap
	Add       AddKeyMap
	Filter    FilterKeyMap
	QuickAdd  QuickAddKeyMap
	Help      HelpKeyMap
}

// DefaultKeyMap returns a KeyMap with all the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Global: GlobalKeyMap{
			Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q/ctrl+c", "quit")),
			Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
			Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		},
		Main: MainKeyMap{
			CycleScope:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle scope")),
			PrevContext:   key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev context")),
			NextContext:   key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next context")),
			MoveUp:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
			MoveDown:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
			Add:           key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			AddChild:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "add child")),
			Edit:          key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
			PriorityHigh:  key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "priority high")),
			PriorityMed:   key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "priority med")),
			PriorityLow:   key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "priority low")),
			PriorityClear: key.NewBinding(key.WithKeys("!"), key.WithHelp("!", "clear priority")),
			ToggleBlocked: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "toggle blocked")),
			ToggleReview:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "toggle review")),
			TagPicker:     key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "tag picker")),
			TagManager:    key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "tag manager")),
			ToggleDone:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle done")),
			Delete:        key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		},
		TagPicker: TagPickerKeyMap{
			Close:     key.NewBinding(key.WithKeys("esc", "enter"), key.WithHelp("esc/enter", "close")),
			MoveUp:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
			MoveDown:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
			Toggle:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
			DeleteTag: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete tag")),
			NewTag:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new tag")),
		},
		Add: AddKeyMap{
			Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
			Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		},
		Filter: FilterKeyMap{
			Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
			Apply:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
		},
		QuickAdd: QuickAddKeyMap{
			Cancel: key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
			Save:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
		},
		Help: HelpKeyMap{
			Close: key.NewBinding(key.WithKeys("?", "esc", "q", "enter", "ctrl+c"), key.WithHelp("?/esc", "close help")),
		},
	}
}

// ApplyOverrides takes user-configured key overrides and applies them to a KeyMap.
// Override keys are snake_case names like "quit", "move_up", "delete".
// It also syncs shared bindings (tag picker nav mirrors main nav).
func ApplyOverrides(km KeyMap, overrides map[string][]string) KeyMap {
	bindings := map[string]*key.Binding{
		"quit":           &km.Global.Quit,
		"help":           &km.Global.Help,
		"filter":         &km.Global.Filter,
		"cycle_scope":    &km.Main.CycleScope,
		"prev_context":   &km.Main.PrevContext,
		"next_context":   &km.Main.NextContext,
		"move_up":        &km.Main.MoveUp,
		"move_down":      &km.Main.MoveDown,
		"add":            &km.Main.Add,
		"add_child":      &km.Main.AddChild,
		"edit":           &km.Main.Edit,
		"priority_high":  &km.Main.PriorityHigh,
		"priority_med":   &km.Main.PriorityMed,
		"priority_low":   &km.Main.PriorityLow,
		"priority_clear": &km.Main.PriorityClear,
		"toggle_blocked": &km.Main.ToggleBlocked,
		"toggle_review":  &km.Main.ToggleReview,
		"tag_picker":     &km.Main.TagPicker,
		"tag_manager":    &km.Main.TagManager,
		"toggle_done":    &km.Main.ToggleDone,
		"delete":         &km.Main.Delete,
		"help_close":     &km.Help.Close,
	}
	for name, keys := range overrides {
		if b, ok := bindings[name]; ok {
			b.SetKeys(keys...)
		}
	}
	// Sync tag picker navigation with main navigation.
	km.TagPicker.MoveUp.SetKeys(km.Main.MoveUp.Keys()...)
	km.TagPicker.MoveDown.SetKeys(km.Main.MoveDown.Keys()...)
	return km
}

// help.KeyMap implementations for auto-generated help text.

type mainHelpKeyMap struct {
	km KeyMap
}

var _ help.KeyMap = mainHelpKeyMap{}

func (h mainHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		h.km.Global.Help,
		h.km.Main.MoveUp,
		h.km.Main.MoveDown,
		h.km.Main.ToggleDone,
		h.km.Main.Delete,
		h.km.Global.Quit,
	}
}

func (h mainHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			h.km.Global.Help,
			h.km.Global.Quit,
			h.km.Main.CycleScope,
			h.km.Global.Filter,
			h.km.Main.PrevContext,
			h.km.Main.NextContext,
		},
		{
			h.km.Main.MoveUp,
			h.km.Main.MoveDown,
			h.km.Main.ToggleDone,
			h.km.Main.Delete,
		},
		{
			h.km.Main.Add,
			h.km.Main.AddChild,
			h.km.Main.Edit,
		},
		{
			h.km.Main.PriorityHigh,
			h.km.Main.PriorityMed,
			h.km.Main.PriorityLow,
			h.km.Main.PriorityClear,
		},
		{
			h.km.Main.ToggleBlocked,
			h.km.Main.ToggleReview,
			h.km.Main.TagPicker,
			h.km.Main.TagManager,
		},
	}
}

type tagPickerHelpKeyMap struct {
	km KeyMap
}

var _ help.KeyMap = tagPickerHelpKeyMap{}

func (h tagPickerHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		h.km.TagPicker.Toggle,
		h.km.TagPicker.NewTag,
		h.km.TagPicker.Close,
	}
}

func (h tagPickerHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			h.km.TagPicker.MoveUp,
			h.km.TagPicker.MoveDown,
			h.km.TagPicker.Toggle,
			h.km.TagPicker.NewTag,
			h.km.TagPicker.DeleteTag,
			h.km.TagPicker.Close,
		},
	}
}

type addHelpKeyMap struct {
	km KeyMap
}

var _ help.KeyMap = addHelpKeyMap{}

func (h addHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		h.km.Add.Confirm,
		h.km.Add.Cancel,
	}
}

func (h addHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{h.ShortHelp()}
}
