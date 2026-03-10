package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/0x-JP/tmux-todo/internal/gitctx"
	"github.com/0x-JP/tmux-todo/internal/quickadd"
	"github.com/0x-JP/tmux-todo/internal/store"
)

type QuickAddModel struct {
	store *store.Store
	ctx   gitctx.Context
	keys  KeyMap

	input       textinput.Model
	height      int
	status      string
	statusIsErr bool
}

var (
	quickHelpBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)
	quickHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
)

func NewQuickAddModel(st *store.Store, ctx gitctx.Context, keys KeyMap) QuickAddModel {
	in := textinput.New()
	in.Prompt = "> "
	in.CharLimit = 300
	in.Width = 64
	in.Focus()
	return QuickAddModel{
		store: st,
		ctx:   ctx,
		keys:  keys,
		input: in,
	}
}

func (m QuickAddModel) Init() tea.Cmd { return textinput.Blink }

func (m QuickAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		w := msg.Width - 6
		if w < 24 {
			w = 24
		}
		if w > 96 {
			w = 96
		}
		m.input.Width = w
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.QuickAdd.Cancel):
			return m, tea.Quit
		case key.Matches(msg, m.keys.QuickAdd.Save):
			spec, err := quickadd.Parse(strings.TrimSpace(m.input.Value()), m.ctx.Key())
			if err != nil {
				m.status = err.Error()
				m.statusIsErr = true
				return m, nil
			}
			spec = normalizeQuickSpecForContext(m.ctx, spec)
			_, err = m.store.AddWithParams(spec.Scope, spec.ContextKey, store.AddParams{
				Text:     spec.Text,
				Priority: spec.Priority,
				Tags:     spec.Tags,
			})
			if err != nil {
				m.status = err.Error()
				m.statusIsErr = true
				return m, nil
			}
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m QuickAddModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("󱑢 tmux-todo"))
	b.WriteString(" ")
	b.WriteString(headerStyle.Render("󰛄 Add Task"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("󰉋 "))
	b.WriteString(contextStyle.Render(quickContextLabel(m.ctx)))
	b.WriteString("\n")
	if m.height <= 0 || m.height >= 12 {
		hints := []string{
			"󰋗 default -> `task name`",
			"󰆓 overrides -> `global | task name`",
			"󰄬 overrides -> `task | p=1|2|3` (high|med|low)",
			"󰓹 overrides -> `task | t=blocked,review`",
		}
		if m.height > 0 && m.height < 15 {
			hints = hints[:2]
		}
		b.WriteString(quickHelpBox.Render(quickHintStyle.Render(strings.Join(hints, "\n"))))
		b.WriteString("\n")
	} else {
		b.WriteString(quickHintStyle.Render("default -> `task name`  |  overrides -> `global | task | p=1 | t=blocked,review`"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.input.View())
	if m.status != "" {
		b.WriteString("\n")
		if m.statusIsErr {
			b.WriteString(statusErr.Render("Error: " + m.status))
		} else {
			b.WriteString(statusOK.Render(m.status))
		}
	}
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("󰌑 enter save  |  esc cancel"))
	b.WriteString("\n")
	return b.String()
}

func (m QuickAddModel) String() string {
	return fmt.Sprintf("quick-add(%s)", m.ctx.Key())
}

func normalizeQuickSpecForContext(ctx gitctx.Context, spec quickadd.Spec) quickadd.Spec {
	if !ctx.IsGit() && spec.Scope == store.ScopeContext {
		spec.Scope = store.ScopeGlobal
		spec.ContextKey = ""
	}
	return spec
}

func quickContextLabel(ctx gitctx.Context) string {
	if !ctx.IsGit() {
		return "Context: Global"
	}
	return "Context: " + ctx.Label()
}
