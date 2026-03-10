package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/0x-JP/tmux-todo/internal/config"
	"github.com/0x-JP/tmux-todo/internal/gitctx"
	"github.com/0x-JP/tmux-todo/internal/store"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	headerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	contextStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	warnStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	statusErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	statusOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	donePeekStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

type listEntry struct {
	Todo     store.Todo
	Depth    int
	IsHeader bool
	Header   string
	Scope    store.Scope
	CtxKey   string
}

type viewMode int

const (
	viewContext viewMode = iota
	viewGeneral
	viewAllContexts
)

type addMode int

const (
	addText addMode = iota
	addPriority
	addTags
)

type MainModel struct {
	store  *store.Store
	cfg    *config.Store
	ctx    gitctx.Context
	strike bool
	keys   KeyMap

	mode           viewMode
	cursor         int
	width          int
	height         int
	adding  bool
	editing bool
	addMode addMode
	tagPicker      bool
	tagCursor      int
	newTagInput    bool
	input          textinput.Model
	tagInput       textinput.Model
	filtering      bool
	filterInput    textinput.Model
	filterPriority store.Priority
	filterTag      string
	showHelp       bool
	tagPickerMode  string
	tagPickScope   store.Scope
	tagPickCtx     string
	tagPickID      string
	addScope       store.Scope
	addCtxKey      string
	addParent      string
	addParentLabel string
	priorityValue  store.Priority
	tagsValue []string
	editID         string
	status         string
	statusIsErr    bool
}

func NewMainModel(st *store.Store, cfg *config.Store, ctx gitctx.Context, strike bool, keys KeyMap) MainModel {
	scope := store.ScopeContext
	if !ctx.IsGit() {
		scope = store.ScopeGlobal
	}
	mode := viewContext
	if scope == store.ScopeGlobal {
		mode = viewGeneral
	}
	if ctx.IsGit() {
		_ = st.SetContextMeta(ctx.Key(), store.MetaInfo{
			RepoRoot:     ctx.RepoRoot,
			WorktreeRoot: ctx.WorktreeRoot,
			Branch:       ctx.Branch,
		})
	}

	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 300
	ti.Width = 80
	tagIn := textinput.New()
	tagIn.Prompt = "new tag> "
	tagIn.CharLimit = 64
	tagIn.Width = 40
	filterIn := textinput.New()
	filterIn.Prompt = "filter> "
	filterIn.CharLimit = 100
	filterIn.Width = 60

	m := MainModel{
		store:       st,
		cfg:         cfg,
		ctx:         ctx,
		strike:      strike,
		keys:        keys,
		mode:        mode,
		input:       ti,
		tagInput:    tagIn,
		filterInput: filterIn,
		addScope:    scope,
		addMode:     addText,
		addCtxKey: func() string {
			if ctx.IsGit() {
				return ctx.Key()
			}
			return ""
		}(),
	}
	m.restoreUIState()
	return m
}

func (m MainModel) Init() tea.Cmd { return nil }

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keys.Help.Close) {
				m.showHelp = false
			}
			return m, nil
		}
		if m.newTagInput {
			switch {
			case key.Matches(msg, m.keys.Add.Cancel):
				m.newTagInput = false
				m.tagInput.SetValue("")
				return m, nil
			case key.Matches(msg, m.keys.Add.Confirm):
				newTag := store.NormalizeTags([]string{m.tagInput.Value()})
				if len(newTag) > 0 {
					switch m.tagPickerMode {
					case "task":
						if err := m.applyTaskTagToggle(newTag[0], true); err != nil {
							m.setStatus(err.Error(), true)
						}
					case "manage":
						if m.cfg != nil {
							_ = m.cfg.AddTag(newTag[0])
						}
					default:
						m.tagsValue = mergeTags(m.tagsValue, newTag)
						if m.cfg != nil {
							_ = m.cfg.AddTag(newTag[0])
						}
					}
				}
				m.tagInput.SetValue("")
				m.newTagInput = false
				return m, nil
			}
			var cmd tea.Cmd
			m.tagInput, cmd = m.tagInput.Update(msg)
			return m, cmd
		}
		if m.tagPicker {
			tags := m.knownTags()
			switch {
			case key.Matches(msg, m.keys.TagPicker.Close):
				m.tagPicker = false
				m.newTagInput = false
				return m, nil
			case key.Matches(msg, m.keys.TagPicker.MoveUp):
				if m.tagCursor > 0 {
					m.tagCursor--
				}
				return m, nil
			case key.Matches(msg, m.keys.TagPicker.MoveDown):
				if m.tagCursor < len(tags)-1 {
					m.tagCursor++
				}
				return m, nil
			case key.Matches(msg, m.keys.TagPicker.Toggle):
				if len(tags) == 0 {
					return m, nil
				}
				tag := tags[m.tagCursor]
				switch m.tagPickerMode {
				case "task":
					if err := m.applyTaskTagToggle(tag, false); err != nil {
						m.setStatus(err.Error(), true)
					}
				case "manage":
				default:
					m.tagsValue = toggleTag(m.tagsValue, tag)
				}
				return m, nil
			case key.Matches(msg, m.keys.TagPicker.DeleteTag):
				if m.tagPickerMode == "manage" && len(tags) > 0 {
					tag := tags[m.tagCursor]
					if m.cfg != nil {
						_ = m.cfg.RemoveTag(tag)
					}
					_ = m.store.RemoveTag(tag)
					if m.tagCursor > 0 {
						m.tagCursor--
					}
				}
				return m, nil
			case key.Matches(msg, m.keys.TagPicker.NewTag):
				m.newTagInput = true
				m.tagInput.Focus()
				return m, nil
			}
			return m, nil
		}
		if m.adding {
			switch m.addMode {
			case addPriority:
				switch {
				case key.Matches(msg, m.keys.Add.Cancel):
					m.cancelAdd()
				case key.Matches(msg, m.keys.Main.PriorityHigh):
					m.priorityValue = store.PriorityHigh
				case key.Matches(msg, m.keys.Main.PriorityMed):
					m.priorityValue = store.PriorityMed
				case key.Matches(msg, m.keys.Main.PriorityLow):
					m.priorityValue = store.PriorityLow
				case msg.String() == "0":
					m.priorityValue = ""
				case key.Matches(msg, m.keys.Add.Confirm):
					if err := m.saveAdd(); err != nil {
						m.setStatus(err.Error(), true)
					} else {
						m.finishAdd()
					}
				case msg.String() == "tab":
					m.addMode = addTags
					m.tagCursor = 0
				case msg.String() == "shift+tab":
					m.addMode = addText
				}
				return m, nil
			case addTags:
				tags := m.knownTags()
				switch {
				case key.Matches(msg, m.keys.Add.Cancel):
					m.cancelAdd()
				case key.Matches(msg, m.keys.TagPicker.MoveUp):
					if m.tagCursor > 0 {
						m.tagCursor--
					}
				case key.Matches(msg, m.keys.TagPicker.MoveDown):
					if m.tagCursor < len(tags)-1 {
						m.tagCursor++
					}
				case key.Matches(msg, m.keys.TagPicker.Toggle):
					if len(tags) > 0 {
						m.tagsValue = toggleTag(m.tagsValue, tags[m.tagCursor])
					}
				case key.Matches(msg, m.keys.TagPicker.NewTag):
					m.newTagInput = true
					m.tagInput.Focus()
				case key.Matches(msg, m.keys.Add.Confirm):
					if err := m.saveAdd(); err != nil {
						m.setStatus(err.Error(), true)
					} else {
						m.finishAdd()
					}
				case msg.String() == "shift+tab":
					m.addMode = addPriority
				}
				return m, nil
			default:
				switch {
				case key.Matches(msg, m.keys.Add.Cancel):
					m.cancelAdd()
					return m, nil
				case key.Matches(msg, m.keys.Add.Confirm):
					if err := m.saveAdd(); err != nil {
						m.setStatus(err.Error(), true)
					} else {
						m.finishAdd()
					}
					return m, nil
				case msg.String() == "tab":
					if strings.TrimSpace(m.input.Value()) == "" {
						m.setStatus("enter text before setting options", true)
						return m, nil
					}
					m.addMode = addPriority
					return m, nil
				}
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
		if m.filtering {
			switch {
			case key.Matches(msg, m.keys.Filter.Cancel):
				m.filtering = false
				return m, nil
			case key.Matches(msg, m.keys.Filter.Apply):
				m.applyFilter(m.filterInput.Value())
				m.filtering = false
				m.cursor = 0
				return m, nil
			}
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Global.Quit):
			m.persistUIState()
			return m, tea.Quit
		case key.Matches(msg, m.keys.Global.Help):
			m.showHelp = true
			return m, nil
		case key.Matches(msg, m.keys.Global.Filter):
			m.filtering = true
			m.filterInput.SetValue("")
			m.filterInput.Focus()
			return m, nil
		case key.Matches(msg, m.keys.Main.CycleScope):
			m.mode = (m.mode + 1) % 3
			m.cursor = 0
			return m, nil
		case key.Matches(msg, m.keys.Main.PrevContext):
			m.shiftContext(-1)
			return m, nil
		case key.Matches(msg, m.keys.Main.NextContext):
			m.shiftContext(1)
			return m, nil
		case key.Matches(msg, m.keys.Main.MoveUp):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Main.MoveDown):
			max := len(m.currentEntries(false)) - 1
			if m.cursor < max {
				m.cursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Main.Add):
			m.adding = true
			m.editing = false
			m.tagPickerMode = "add"
			m.addScope, m.addCtxKey = m.defaultAddTarget()
			m.addParent = ""
			m.addParentLabel = ""
			m.priorityValue = ""
			m.tagsValue = nil
			m.editID = ""
			m.input.SetValue("")
			m.input.Focus()
			m.addMode = addText
			m.tagCursor = 0
			return m, nil
		case key.Matches(msg, m.keys.Main.AddChild):
			t := m.currentTodo()
			if t == nil {
				m.setStatus("select a todo to add a child", false)
				return m, nil
			}
			m.adding = true
			m.editing = false
			m.tagPickerMode = "add"
			m.addScope, m.addCtxKey = m.defaultAddTarget()
			m.addParent = ""
			m.addParentLabel = ""
			m.priorityValue = t.Priority
			m.tagsValue = append([]string(nil), t.Tags...)
			if t != nil {
				m.addParent = t.ID
				m.addParentLabel = t.Text
			}
			m.input.Focus()
			m.addMode = addText
			m.tagCursor = 0
			return m, nil
		case key.Matches(msg, m.keys.Main.Edit):
			e := m.currentEntry()
			if e == nil || e.IsHeader {
				return m, nil
			}
			m.adding = true
			m.editing = true
			m.tagPickerMode = "add"
			m.addScope = e.Scope
			m.addCtxKey = e.CtxKey
			m.editID = e.Todo.ID
			m.addParent = e.Todo.ParentID
			m.addParentLabel = ""
			m.priorityValue = e.Todo.Priority
			m.tagsValue = append([]string(nil), e.Todo.Tags...)
			m.input.SetValue(e.Todo.Text)
			m.input.Focus()
			m.addMode = addText
			m.tagCursor = 0
			return m, nil
		case key.Matches(msg, m.keys.Main.PriorityHigh), key.Matches(msg, m.keys.Main.PriorityMed), key.Matches(msg, m.keys.Main.PriorityLow), key.Matches(msg, m.keys.Main.PriorityClear):
			e := m.currentEntry()
			if e == nil || e.IsHeader {
				return m, nil
			}
			var p store.Priority
			switch {
			case key.Matches(msg, m.keys.Main.PriorityHigh):
				p = store.PriorityHigh
			case key.Matches(msg, m.keys.Main.PriorityMed):
				p = store.PriorityMed
			case key.Matches(msg, m.keys.Main.PriorityLow):
				p = store.PriorityLow
			case key.Matches(msg, m.keys.Main.PriorityClear):
				p = ""
			}
			if _, err := m.store.Update(e.Scope, e.CtxKey, e.Todo.ID, store.UpdateParams{Priority: &p}); err != nil {
				m.setStatus(err.Error(), true)
			}
			return m, nil
		case key.Matches(msg, m.keys.Main.ToggleBlocked):
			e := m.currentEntry()
			if e == nil || e.IsHeader {
				return m, nil
			}
			tags := toggleTag(e.Todo.Tags, "blocked")
			if _, err := m.store.Update(e.Scope, e.CtxKey, e.Todo.ID, store.UpdateParams{Tags: &tags}); err != nil {
				m.setStatus(err.Error(), true)
			}
			return m, nil
		case key.Matches(msg, m.keys.Main.ToggleReview):
			e := m.currentEntry()
			if e == nil || e.IsHeader {
				return m, nil
			}
			tags := toggleTag(e.Todo.Tags, "review")
			if _, err := m.store.Update(e.Scope, e.CtxKey, e.Todo.ID, store.UpdateParams{Tags: &tags}); err != nil {
				m.setStatus(err.Error(), true)
			}
			return m, nil
		case key.Matches(msg, m.keys.Main.TagPicker):
			e := m.currentEntry()
			if e == nil || e.IsHeader {
				return m, nil
			}
			m.tagPicker = true
			m.newTagInput = false
			m.tagPickerMode = "task"
			m.tagPickScope = e.Scope
			m.tagPickCtx = e.CtxKey
			m.tagPickID = e.Todo.ID
			m.tagCursor = 0
			return m, nil
		case key.Matches(msg, m.keys.Main.TagManager):
			m.tagPicker = true
			m.newTagInput = false
			m.tagPickerMode = "manage"
			m.tagCursor = 0
			return m, nil
		case key.Matches(msg, m.keys.Main.ToggleDone):
			e := m.currentEntry()
			if e == nil || e.IsHeader {
				return m, nil
			}
			if err := m.store.Toggle(e.Scope, e.CtxKey, e.Todo.ID); err != nil {
				m.setStatus(err.Error(), true)
			}
			return m, nil
		case key.Matches(msg, m.keys.Main.Delete):
			e := m.currentEntry()
			if e == nil || e.IsHeader {
				return m, nil
			}
			if err := m.store.Delete(e.Scope, e.CtxKey, e.Todo.ID); err != nil {
				m.setStatus(err.Error(), true)
			} else {
				if m.cursor > 0 && m.cursor >= len(m.currentEntries(false)) {
					m.cursor--
				}
			}
			return m, nil
		}
	}
	return m, nil
}

func (m MainModel) View() string {
	if m.showHelp {
		return m.helpView()
	}
	if m.tagPicker && !m.adding && m.tagPickerMode == "manage" {
		return m.tagPickerView()
	}
	var b strings.Builder

	scopeLabel := "Context"
	if m.mode == viewGeneral {
		scopeLabel = "Global"
	} else if m.mode == viewAllContexts {
		scopeLabel = "All Contexts"
	}
	b.WriteString(titleStyle.Render("󱑢 tmux-todo"))
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Scope: " + scopeLabel + " | Current: "))
	b.WriteString(contextStyle.Render(m.ctx.Label()))
	b.WriteString("\n")
	b.WriteString(headerStyle.Render(m.summaryLine()))
	if m.filterTag != "" || m.filterPriority != "" {
		b.WriteString("\n")
		b.WriteString(headerStyle.Render("Filter: " + m.filterExpr()))
	}
	b.WriteString("\n\n")

	entries := m.currentEntries(false)
	if len(entries) == 0 {
		b.WriteString("(no todos)\n")
	} else {
		for i, e := range entries {
			if e.IsHeader {
				line := fmt.Sprintf("  %s", e.Header)
				if i == m.cursor {
					line = "> " + e.Header
				}
				if e.Scope == store.ScopeContext {
					b.WriteString(contextStyle.Render(line))
				} else {
					b.WriteString(headerStyle.Render(line))
				}
				b.WriteString("\n")
				continue
			}
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			mark := "󰄱"
			if e.Todo.Done {
				mark = "󰄲"
			}
			indent := strings.Repeat("  ", e.Depth)
			text := e.Todo.Text
			if e.Todo.Done {
				text = maybeStrike(text, m.strike)
			}
			meta := renderMeta(e.Todo)
			if meta != "" {
				text += " " + meta
			}
			line := fmt.Sprintf("%s%s%s %s", prefix, indent, mark, text)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if m.adding {
		if m.addParent != "" {
			b.WriteString(fmt.Sprintf("Child of: %s\n", m.parentDisplay()))
		}
		switch m.addMode {
		case addPriority:
			b.WriteString(fmt.Sprintf("Task: %s\n", strings.TrimSpace(m.input.Value())))
			b.WriteString(fmt.Sprintf("Priority  %s\n", displayPriority(m.priorityValue)))
			b.WriteString(subtleStyle.Render("1 high  2 med  3 low  0 none  enter save  tab tags  esc cancel"))
			b.WriteString("\n")
		case addTags:
			b.WriteString(fmt.Sprintf("Task: %s\n", strings.TrimSpace(m.input.Value())))
			b.WriteString(fmt.Sprintf("Priority: %s\n", displayPriority(m.priorityValue)))
			b.WriteString(fmt.Sprintf("Tags  %s\n", displayTags(m.tagsValue)))
			tags := m.knownTags()
			if len(tags) == 0 {
				b.WriteString("(no tags)\n")
			} else {
				for i, tg := range tags {
					prefix := "  "
					if i == m.tagCursor {
						prefix = "> "
					}
					mark := "o"
					if hasTag(m.tagsValue, tg) {
						mark = "●"
					}
					b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, mark, tg))
				}
			}
			if m.newTagInput {
				b.WriteString(m.tagInput.View())
				b.WriteString("\n")
			}
			b.WriteString(subtleStyle.Render("space toggle  n new  enter save  esc cancel"))
			b.WriteString("\n")
		default:
			b.WriteString(m.input.View())
			b.WriteString("\n")
			b.WriteString(subtleStyle.Render("enter save  tab more  esc cancel"))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(subtleStyle.Render(fmt.Sprintf("Keys: %s help | %s scope | %s filter | %s add | %s add-child | %s edit",
			m.keys.Global.Help.Help().Key, m.keys.Main.CycleScope.Help().Key, m.keys.Global.Filter.Help().Key,
			m.keys.Main.Add.Help().Key, m.keys.Main.AddChild.Help().Key, m.keys.Main.Edit.Help().Key)))
		b.WriteString("\n")
		b.WriteString(subtleStyle.Render(fmt.Sprintf("      %s/%s/%s/%s priority | %s/%s tags | %s toggle | %s delete | %s/%s move | %s quit",
			m.keys.Main.PriorityHigh.Help().Key, m.keys.Main.PriorityMed.Help().Key, m.keys.Main.PriorityLow.Help().Key, m.keys.Main.PriorityClear.Help().Key,
			m.keys.Main.ToggleBlocked.Help().Key, m.keys.Main.ToggleReview.Help().Key,
			m.keys.Main.ToggleDone.Help().Key, m.keys.Main.Delete.Help().Key,
			m.keys.Main.MoveUp.Help().Key, m.keys.Main.MoveDown.Help().Key,
			m.keys.Global.Quit.Help().Key)))
		b.WriteString("\n")
		if m.tagPicker && m.tagPickerMode == "task" {
			b.WriteString("\n")
			b.WriteString(headerStyle.Render("Task Tag Picker"))
			b.WriteString("\n")
			b.WriteString("space toggle | n new tag | esc close\n")
			t := m.lookupTodo(m.tagPickScope, m.tagPickCtx, m.tagPickID)
			if t != nil {
				b.WriteString("Task: " + t.Text + "\n")
			}
			tags := m.knownTags()
			for i, tg := range tags {
				prefix := "  "
				if i == m.tagCursor {
					prefix = "> "
				}
				mark := "󰄱"
				if t != nil && hasTag(t.Tags, tg) {
					mark = "󰄲"
				}
				b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, mark, tg))
			}
			if m.newTagInput {
				b.WriteString(m.tagInput.View() + "\n")
			}
		}
	}
	if m.filtering {
		b.WriteString(m.filterInput.View())
		b.WriteString("\n")
	}
	if m.status != "" {
		if m.statusIsErr {
			b.WriteString(statusErr.Render("Status: " + m.status))
		} else {
			b.WriteString(statusOK.Render("Status: " + m.status))
		}
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m MainModel) tagPickerView() string {
	var b strings.Builder
	title := "Tag Picker"
	if m.tagPickerMode == "manage" {
		title = "Tag Manager"
	}
	b.WriteString(titleStyle.Render("󰝗 tmux-todo " + title))
	b.WriteString("\n")

	if m.tagPickerMode == "task" {
		t := m.lookupTodo(m.tagPickScope, m.tagPickCtx, m.tagPickID)
		if t != nil {
			b.WriteString(headerStyle.Render("Task: " + t.Text))
		} else {
			b.WriteString(headerStyle.Render("Task: (not found)"))
		}
		b.WriteString("\n")
	}
	b.WriteString(headerStyle.Render("space toggle | n new tag | esc close"))
	if m.tagPickerMode == "manage" {
		b.WriteString(headerStyle.Render(" | d remove globally"))
	}
	b.WriteString("\n\n")

	tags := m.knownTags()
	if len(tags) == 0 {
		b.WriteString("(no tags)\n")
	} else {
		for i, tg := range tags {
			prefix := "  "
			if i == m.tagCursor {
				prefix = "> "
			}
			mark := "󰄱"
			if m.tagPickerMode == "task" {
				t := m.lookupTodo(m.tagPickScope, m.tagPickCtx, m.tagPickID)
				if t != nil && hasTag(t.Tags, tg) {
					mark = "󰄲"
				}
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, mark, tg))
		}
	}
	if m.newTagInput {
		b.WriteString("\n")
		b.WriteString(m.tagInput.View())
		b.WriteString("\n")
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m MainModel) currentBaseList() []store.Todo {
	d := m.store.Snapshot()
	if m.mode == viewGeneral {
		return d.Global
	}
	return d.Contexts[m.ctx.Key()]
}

func (m MainModel) currentList() []store.Todo {
	base := m.currentBaseList()
	if m.filterPriority == "" && m.filterTag == "" {
		return base
	}
	return store.FilterAndSort(base, store.Filter{
		ShowDone: true,
		Priority: m.filterPriority,
		Tag:      m.filterTag,
		Sort:     "priority_due_created",
	})
}

func (m MainModel) currentEntries(openOnly bool) []listEntry {
	if m.mode == viewAllContexts {
		return m.allContextEntries(openOnly)
	}
	scope := store.ScopeContext
	ctxKey := m.ctx.Key()
	if m.mode == viewGeneral {
		scope = store.ScopeGlobal
		ctxKey = ""
	}
	return flattenTodos(m.currentList(), openOnly, scope, ctxKey)
}

func (m *MainModel) currentTodo() *store.Todo {
	e := m.currentEntry()
	if e == nil || e.IsHeader {
		return nil
	}
	t := e.Todo
	return &t
}

func (m *MainModel) currentEntry() *listEntry {
	entries := m.currentEntries(false)
	if len(entries) == 0 {
		return nil
	}
	if m.cursor >= len(entries) {
		m.cursor = len(entries) - 1
	}
	e := entries[m.cursor]
	return &e
}

func (m *MainModel) setStatus(s string, isErr bool) {
	m.status = s
	m.statusIsErr = isErr
}

func (m *MainModel) saveAdd() error {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return fmt.Errorf("todo text cannot be empty")
	}
	if m.editing {
		up := store.UpdateParams{}
		up.Text = &text
		up.Priority = &m.priorityValue
		tags := store.NormalizeTags(m.tagsValue)
		up.Tags = &tags
		_, err := m.store.Update(m.addScope, m.addCtxKey, m.editID, up)
		if err != nil {
			return err
		}
		return nil
	}
	_, err := m.store.AddWithParams(m.addScope, m.addCtxKey, store.AddParams{
		Text:     text,
		ParentID: m.addParent,
		Priority: m.priorityValue,
		Tags:     store.NormalizeTags(m.tagsValue),
	})
	return err
}

func (m *MainModel) persistUIState() {
	if m.cfg == nil {
		return
	}
	ui := config.UIState{
		MainMode: map[viewMode]string{
			viewContext:     "context",
			viewGeneral:     "global",
			viewAllContexts: "all",
		}[m.mode],
	}
	if e := m.currentEntry(); e != nil && !e.IsHeader {
		ui.Selected.Scope = string(e.Scope)
		ui.Selected.ContextKey = e.CtxKey
		ui.Selected.ID = e.Todo.ID
	}
	_ = m.cfg.SaveUI(ui)
}

func (m *MainModel) restoreUIState() {
	if m.cfg == nil {
		return
	}
	ui := m.cfg.UI()

	if ui.Selected.ID == "" {
		return
	}
	if m.mode == viewGeneral {
		if ui.Selected.Scope != string(store.ScopeGlobal) {
			return
		}
	} else {
		if ui.Selected.Scope != string(store.ScopeContext) || ui.Selected.ContextKey != m.ctx.Key() {
			return
		}
	}
	entries := m.currentEntries(false)
	for i, e := range entries {
		if e.IsHeader {
			continue
		}
		if e.Scope == store.Scope(ui.Selected.Scope) &&
			e.CtxKey == ui.Selected.ContextKey &&
			e.Todo.ID == ui.Selected.ID {
			m.cursor = i
			return
		}
	}
}

func (m *MainModel) cancelAdd() {
	m.adding = false
	m.editing = false
	m.tagPicker = false
	m.newTagInput = false
	m.tagInput.SetValue("")
	m.input.SetValue("")
	m.addParent = ""
	m.addParentLabel = ""
	m.editID = ""
	m.priorityValue = ""
	m.tagsValue = nil
	m.tagPickerMode = ""
	m.addMode = addText
}

func (m *MainModel) finishAdd() {
	m.adding = false
	m.editing = false
	m.tagPicker = false
	m.newTagInput = false
	m.tagInput.SetValue("")
	m.input.SetValue("")
	m.addParent = ""
	m.addParentLabel = ""
	m.addCtxKey = m.ctx.Key()
	m.editID = ""
	m.priorityValue = ""
	m.tagsValue = nil
	m.tagPickerMode = ""
	m.addMode = addText
}

func (m MainModel) helpView() string {
	k := m.keys
	var b strings.Builder
	b.WriteString(titleStyle.Render("tmux-todo Help"))
	b.WriteString("\n")
	b.WriteString("General:\n")
	b.WriteString(fmt.Sprintf("  %s toggle help | %s quit | %s cycle scope | %s/%s move | %s open filter\n",
		k.Global.Help.Help().Key, k.Global.Quit.Help().Key,
		k.Main.CycleScope.Help().Key, k.Main.MoveUp.Help().Key, k.Main.MoveDown.Help().Key,
		k.Global.Filter.Help().Key))
	b.WriteString("\n")
	b.WriteString("Task actions:\n")
	b.WriteString(fmt.Sprintf("  %s add (type text, enter saves, tab for priority/tags)\n", k.Main.Add.Help().Key))
	b.WriteString(fmt.Sprintf("  %s add child task\n", k.Main.AddChild.Help().Key))
	b.WriteString(fmt.Sprintf("  %s edit selected task\n", k.Main.Edit.Help().Key))
	b.WriteString(fmt.Sprintf("  %s add/remove tags for selected task\n", k.Main.TagPicker.Help().Key))
	b.WriteString(fmt.Sprintf("  %s tag manager (remove tag globally)\n", k.Main.TagManager.Help().Key))
	b.WriteString(fmt.Sprintf("  %s toggle done | %s delete\n", k.Main.ToggleDone.Help().Key, k.Main.Delete.Help().Key))
	b.WriteString(fmt.Sprintf("  %s high | %s med | %s low | %s clear priority\n",
		k.Main.PriorityHigh.Help().Key, k.Main.PriorityMed.Help().Key,
		k.Main.PriorityLow.Help().Key, k.Main.PriorityClear.Help().Key))
	b.WriteString(fmt.Sprintf("  %s toggle blocked tag | %s toggle review tag\n",
		k.Main.ToggleBlocked.Help().Key, k.Main.ToggleReview.Help().Key))
	b.WriteString("\n")
	b.WriteString("Add/edit flow:\n")
	b.WriteString("  text step: enter save, tab for priority\n")
	b.WriteString("  priority step: 1/2/3/0, enter save, tab for tags\n")
	b.WriteString("  tags step: space toggle, n new, enter save\n")
	b.WriteString("  shift+tab go back, esc cancel\n")
	b.WriteString("\n")
	b.WriteString("Tag management:\n")
	b.WriteString(fmt.Sprintf("  %s task tag picker (selection mode)\n", k.Main.TagPicker.Help().Key))
	b.WriteString(fmt.Sprintf("  %s global tag manager, %s removes selected tag everywhere\n",
		k.Main.TagManager.Help().Key, k.TagPicker.DeleteTag.Help().Key))
	b.WriteString("\n")
	b.WriteString("Filter examples:\n")
	b.WriteString("  p:high\n")
	b.WriteString("  tag:blocked\n")
	b.WriteString("  p:med tag:review\n")
	b.WriteString(fmt.Sprintf("\nPress %s to close help.", k.Help.Close.Help().Key))
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m MainModel) parentDisplay() string {
	label := strings.TrimSpace(m.addParentLabel)
	if label == "" {
		if t := m.lookupTodo(m.addScope, m.addCtxKey, m.addParent); t != nil {
			label = strings.TrimSpace(t.Text)
		}
	}
	if label == "" {
		return shortID(m.addParent)
	}
	if len(label) > 40 {
		label = label[:40] + "..."
	}
	return fmt.Sprintf("%s (%s)", label, shortID(m.addParent))
}

func (m MainModel) lookupTodo(scope store.Scope, ctxKey, id string) *store.Todo {
	if id == "" {
		return nil
	}
	d := m.store.Snapshot()
	var list []store.Todo
	if scope == store.ScopeGlobal {
		list = d.Global
	} else {
		list = d.Contexts[ctxKey]
	}
	for _, t := range list {
		if t.ID == id {
			c := t
			return &c
		}
	}
	return nil
}

func (m MainModel) allContextEntries(openOnly bool) []listEntry {
	d := m.store.Snapshot()
	out := make([]listEntry, 0)
	out = append(out, listEntry{IsHeader: true, Header: "Global", Scope: store.ScopeGlobal})
	global := flattenTodos(m.applyTodoFilter(d.Global), openOnly, store.ScopeGlobal, "")
	for _, e := range global {
		e.Depth++
		out = append(out, e)
	}

	keySet := make(map[string]struct{}, len(d.Contexts)+len(d.Meta))
	for k := range d.Contexts {
		keySet[k] = struct{}{}
	}
	for k := range d.Meta {
		keySet[k] = struct{}{}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		label := k
		if meta, ok := d.Meta[k]; ok {
			label = fmt.Sprintf("%s/%s \U000F062C %s",
				baseOr(meta.RepoRoot, "repo"),
				baseOr(meta.WorktreeRoot, "worktree"),
				meta.Branch)
		}
		out = append(out, listEntry{IsHeader: true, Header: label, Scope: store.ScopeContext, CtxKey: k})
		child := flattenTodos(m.applyTodoFilter(d.Contexts[k]), openOnly, store.ScopeContext, k)
		for _, e := range child {
			e.Depth++
			out = append(out, e)
		}
	}
	return out
}

func (m *MainModel) shiftContext(delta int) {
	if !m.ctx.IsGit() {
		return
	}
	keys := m.contextKeys()
	if len(keys) == 0 {
		return
	}
	cur := m.ctx.Key()
	idx := 0
	for i, k := range keys {
		if k == cur {
			idx = i
			break
		}
	}
	next := (idx + delta + len(keys)) % len(keys)
	if keys[next] == cur {
		return
	}
	if !m.setContextByKey(keys[next]) {
		return
	}
	m.mode = viewContext
	m.cursor = 0
	m.setStatus("context: "+m.ctx.Label(), false)
}

func (m MainModel) contextKeys() []string {
	d := m.store.Snapshot()
	set := map[string]struct{}{}
	if m.ctx.IsGit() {
		set[m.ctx.Key()] = struct{}{}
	}
	for k := range d.Contexts {
		if k != "" && k != "global" {
			set[k] = struct{}{}
		}
	}
	for k := range d.Meta {
		if k != "" && k != "global" {
			set[k] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (m *MainModel) setContextByKey(key string) bool {
	if key == "" || key == "global" {
		return false
	}
	d := m.store.Snapshot()
	if meta, ok := d.Meta[key]; ok {
		m.ctx = gitctx.Context{
			RepoRoot:     meta.RepoRoot,
			WorktreeRoot: meta.WorktreeRoot,
			Branch:       meta.Branch,
		}
		return true
	}
	parts := strings.Split(key, "|")
	kv := map[string]string{}
	for _, p := range parts {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		kv[k] = v
	}
	if kv["repo"] == "" || kv["wt"] == "" || kv["br"] == "" {
		return false
	}
	m.ctx = gitctx.Context{
		RepoRoot:     kv["repo"],
		WorktreeRoot: kv["wt"],
		Branch:       kv["br"],
	}
	return true
}

func (m MainModel) defaultAddTarget() (store.Scope, string) {
	e := m.currentEntry()
	if e != nil {
		if e.IsHeader {
			if e.Scope == store.ScopeContext {
				return store.ScopeContext, e.CtxKey
			}
			return store.ScopeGlobal, ""
		}
		return e.Scope, e.CtxKey
	}
	if m.mode == viewGeneral {
		return store.ScopeGlobal, ""
	}
	return store.ScopeContext, m.ctx.Key()
}

type PeekModel struct {
	store      *store.Store
	ctx        gitctx.Context
	strike     bool
	closeAfter time.Duration
	highOnly   bool
	width      int
	height     int
}

func NewPeekModel(st *store.Store, ctx gitctx.Context, strike bool, closeAfter time.Duration) PeekModel {
	if closeAfter <= 0 {
		closeAfter = 5 * time.Second
	}
	return PeekModel{store: st, ctx: ctx, strike: strike, closeAfter: closeAfter}
}

func NewHighPeekModel(st *store.Store, ctx gitctx.Context, closeAfter time.Duration) PeekModel {
	if closeAfter <= 0 {
		closeAfter = 2 * time.Second
	}
	return PeekModel{
		store:      st,
		ctx:        ctx,
		strike:     false,
		closeAfter: closeAfter,
		highOnly:   true,
	}
}

func (m PeekModel) Init() tea.Cmd {
	return tea.Tick(m.closeAfter, func(time.Time) tea.Msg {
		return tea.QuitMsg{}
	})
}

func (m PeekModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.QuitMsg:
		return m, tea.Quit
	case tea.KeyMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m PeekModel) View() string {
	if m.highOnly {
		return m.viewHighAlert()
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("󰄬 Quick Todos"))
	b.WriteString("\n")
	b.WriteString(contextStyle.Render(m.ctx.Label()))
	b.WriteString("\n\n")

	m.renderPeekSection(&b, "Open", m.peekOpen(store.ScopeContext), 4, false)
	m.renderPeekSection(&b, "Recently Done", m.peekDone(store.ScopeContext, 2), 2, true)
	b.WriteString("\n")
	m.renderPeekSection(&b, "Global Open", m.peekOpen(store.ScopeGlobal), 4, false)
	m.renderPeekSection(&b, "Global Recently Done", m.peekDone(store.ScopeGlobal, 2), 2, true)
	secs := int(m.closeAfter.Round(time.Second) / time.Second)
	b.WriteString(fmt.Sprintf("\n(closes in %ds)", secs))
	return lipgloss.NewStyle().Padding(0, 1).Render(b.String())
}

func (m PeekModel) viewHighAlert() string {
	var b strings.Builder
	msg := "  High priority items in " + m.ctx.Label()
	if m.width > 0 {
		msg = wrapText(msg, m.width-4)
	}
	b.WriteString(warnStyle.Render(msg))
	b.WriteString("\n")
	b.WriteString("\n")

	entries := m.peekOpen(store.ScopeContext)
	high := make([]listEntry, 0, len(entries))
	for _, e := range entries {
		if e.Todo.Priority == store.PriorityHigh {
			high = append(high, e)
		}
	}
	if len(high) == 0 {
		b.WriteString("(no high-priority open tasks)\n")
	} else {
		if len(high) > 3 {
			high = high[:3]
		}
		for _, e := range high {
			indent := strings.Repeat("  ", e.Depth)
			b.WriteString(fmt.Sprintf("  %s󰄱 %s\n", indent, e.Todo.Text))
		}
	}
	secs := int(m.closeAfter.Round(time.Second) / time.Second)
	if secs < 1 {
		secs = 1
	}
	b.WriteString(fmt.Sprintf("\n(closes in %ds)", secs))
	return lipgloss.NewStyle().Padding(0, 1).Render(b.String())
}

func (m PeekModel) peekOpen(scope store.Scope) []listEntry {
	d := m.store.Snapshot()
	if scope == store.ScopeContext {
		return flattenTodos(d.Contexts[m.ctx.Key()], true, scope, m.ctx.Key())
	}
	return flattenTodos(d.Global, true, scope, "")
}

func (m PeekModel) peekDone(scope store.Scope, limit int) []listEntry {
	d := m.store.Snapshot()
	var all []listEntry
	if scope == store.ScopeContext {
		all = flattenTodos(d.Contexts[m.ctx.Key()], false, scope, m.ctx.Key())
	} else {
		all = flattenTodos(d.Global, false, scope, "")
	}
	done := make([]listEntry, 0, len(all))
	for _, e := range all {
		if e.Todo.Done {
			done = append(done, e)
		}
	}
	sort.Slice(done, func(i, j int) bool {
		a := done[i].Todo.CompletedAt
		b := done[j].Todo.CompletedAt
		if a == nil && b == nil {
			return done[i].Todo.CreatedAt.After(done[j].Todo.CreatedAt)
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.After(*b)
	})
	if len(done) > limit {
		done = done[:limit]
	}
	return done
}

func (m PeekModel) renderPeekSection(b *strings.Builder, title string, entries []listEntry, max int, done bool) {
	b.WriteString(headerStyle.Render(title))
	b.WriteString("\n")
	if len(entries) == 0 {
		b.WriteString("  (none)\n")
		return
	}
	if len(entries) > max {
		entries = entries[:max]
	}
	for _, e := range entries {
		indent := strings.Repeat("  ", e.Depth)
		line := fmt.Sprintf("  %s󰄱 %s", indent, e.Todo.Text)
		if done {
			line = fmt.Sprintf("  %s󰄲 %s", indent, maybeStrike(e.Todo.Text, m.strike))
			line = donePeekStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func flattenTodos(todos []store.Todo, openOnly bool, scope store.Scope, ctxKey string) []listEntry {
	byParent := make(map[string][]store.Todo, len(todos))
	knownIDs := make(map[string]struct{}, len(todos))
	for _, t := range todos {
		knownIDs[t.ID] = struct{}{}
	}
	for _, t := range todos {
		p := t.ParentID
		if p != "" {
			if _, ok := knownIDs[p]; !ok {
				p = ""
			}
		}
		byParent[p] = append(byParent[p], t)
	}
	out := make([]listEntry, 0, len(todos))
	seen := map[string]struct{}{}
	var walk func(parent string, depth int)
	walk = func(parent string, depth int) {
		for _, t := range byParent[parent] {
			if _, ok := seen[t.ID]; ok {
				continue
			}
			seen[t.ID] = struct{}{}
			if !openOnly || !t.Done {
				out = append(out, listEntry{Todo: t, Depth: depth, Scope: scope, CtxKey: ctxKey})
			}
			walk(t.ID, depth+1)
		}
	}
	walk("", 0)

	for _, t := range todos {
		if _, ok := seen[t.ID]; ok {
			continue
		}
		if !openOnly || !t.Done {
			out = append(out, listEntry{Todo: t, Depth: 0, Scope: scope, CtxKey: ctxKey})
		}
	}
	return out
}

func strikeText(s string) string {
	const comb = '\u0336'
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(r)
		if r != ' ' {
			b.WriteRune(comb)
		}
	}
	return b.String()
}

func maybeStrike(s string, enabled bool) string {
	if !enabled {
		return s
	}
	return strikeText(s)
}

func baseOr(path, fallback string) string {
	if path == "" {
		return fallback
	}
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return fallback
	}
	return parts[len(parts)-1]
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func renderMeta(t store.Todo) string {
	parts := []string{}
	if t.Priority != "" {
		parts = append(parts, "p="+string(t.Priority))
	}
	if len(t.Tags) > 0 {
		parts = append(parts, "#"+strings.Join(t.Tags, ",#"))
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func (m MainModel) applyTodoFilter(list []store.Todo) []store.Todo {
	if m.filterPriority == "" && m.filterTag == "" {
		return list
	}
	return store.FilterAndSort(list, store.Filter{
		ShowDone: true,
		Priority: m.filterPriority,
		Tag:      m.filterTag,
		Sort:     "priority_due_created",
	})
}

func (m MainModel) knownTags() []string {
	set := map[string]struct{}{}
	for _, t := range store.KnownTags(m.store.Snapshot()) {
		set[t] = struct{}{}
	}
	if m.cfg != nil {
		for _, t := range m.cfg.Tags() {
			set[t] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func (m MainModel) summaryLine() string {
	entries := m.currentEntries(false)
	open := 0
	high := 0
	blocked := 0
	for _, e := range entries {
		if e.IsHeader {
			continue
		}
		if !e.Todo.Done {
			open++
			if e.Todo.Priority == store.PriorityHigh {
				high++
			}
			if hasTag(e.Todo.Tags, "blocked") {
				blocked++
			}
		}
	}
	return fmt.Sprintf("Open:%d High:%d Blocked:%d", open, high, blocked)
}

func (m MainModel) filterExpr() string {
	parts := []string{}
	if m.filterPriority != "" {
		parts = append(parts, "p:"+string(m.filterPriority))
	}
	if m.filterTag != "" {
		parts = append(parts, "tag:"+m.filterTag)
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " ")
}

func (m *MainModel) applyFilter(expr string) {
	m.filterPriority = ""
	m.filterTag = ""
	for _, tok := range strings.Fields(strings.ToLower(strings.TrimSpace(expr))) {
		if strings.HasPrefix(tok, "p:") {
			if p, err := store.NormalizePriority(strings.TrimPrefix(tok, "p:")); err == nil {
				m.filterPriority = p
			}
		}
		if strings.HasPrefix(tok, "tag:") {
			m.filterTag = strings.TrimSpace(strings.TrimPrefix(tok, "tag:"))
		}
	}
}

func displayPriority(p store.Priority) string {
	if p == "" {
		return "none"
	}
	return string(p)
}

func displayTags(tags []string) string {
	tags = store.NormalizeTags(tags)
	if len(tags) == 0 {
		return "none"
	}
	return strings.Join(tags, ",")
}

func toggleTag(tags []string, tag string) []string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return store.NormalizeTags(tags)
	}
	out := []string{}
	found := false
	for _, t := range store.NormalizeTags(tags) {
		if t == tag {
			found = true
			continue
		}
		out = append(out, t)
	}
	if !found {
		out = append(out, tag)
	}
	return store.NormalizeTags(out)
}

func hasTag(tags []string, tag string) bool {
	tag = strings.ToLower(strings.TrimSpace(tag))
	for _, t := range store.NormalizeTags(tags) {
		if t == tag {
			return true
		}
	}
	return false
}

func mergeTags(base, extra []string) []string {
	all := append([]string(nil), base...)
	all = append(all, extra...)
	return store.NormalizeTags(all)
}

func wrapText(s string, max int) string {
	if max <= 0 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	lines := []string{}
	cur := words[0]
	for _, w := range words[1:] {
		if len([]rune(cur))+1+len([]rune(w)) <= max {
			cur += " " + w
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	lines = append(lines, cur)
	return strings.Join(lines, "\n")
}

func (m *MainModel) applyTaskTagToggle(tag string, forceAdd bool) error {
	if m.tagPickID == "" {
		return nil
	}
	t := m.lookupTodo(m.tagPickScope, m.tagPickCtx, m.tagPickID)
	if t == nil {
		return fmt.Errorf("selected task no longer exists")
	}
	tags := append([]string(nil), t.Tags...)
	if forceAdd {
		tags = mergeTags(tags, []string{tag})
	} else {
		tags = toggleTag(tags, tag)
	}
	if _, err := m.store.Update(m.tagPickScope, m.tagPickCtx, m.tagPickID, store.UpdateParams{
		Tags: &tags,
	}); err != nil {
		return err
	}
	_ = m.cfg.AddTag(tag)
	return nil
}
