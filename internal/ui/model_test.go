package ui

import (
	"path/filepath"
	"testing"

	"github.com/0x-JP/tmux-todo/internal/config"
	"github.com/0x-JP/tmux-todo/internal/gitctx"
	"github.com/0x-JP/tmux-todo/internal/store"
)

func TestFlattenTodosScopeAndDepth(t *testing.T) {
	todos := []store.Todo{
		{ID: "a", Text: "parent"},
		{ID: "b", Text: "child", ParentID: "a"},
	}
	out := flattenTodos(todos, false, store.ScopeContext, "ctx1")
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Depth != 0 || out[1].Depth != 1 {
		t.Fatalf("depths=%d,%d", out[0].Depth, out[1].Depth)
	}
	if out[1].Scope != store.ScopeContext || out[1].CtxKey != "ctx1" {
		t.Fatalf("unexpected scope info: %#v", out[1])
	}
}

func TestRenderMeta(t *testing.T) {
	got := renderMeta(store.Todo{
		Priority: store.PriorityMed,
		Tags:     []string{"blocked"},
	})
	if got == "" {
		t.Fatal("expected metadata rendering")
	}
}

func TestToggleTag(t *testing.T) {
	got := toggleTag([]string{"review"}, "blocked")
	if !hasTag(got, "blocked") || !hasTag(got, "review") {
		t.Fatalf("unexpected tags: %v", got)
	}
	got = toggleTag(got, "blocked")
	if hasTag(got, "blocked") {
		t.Fatalf("expected blocked removed: %v", got)
	}
}

func TestRestoreUIState(t *testing.T) {
	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "todos.json"))
	if err != nil {
		t.Fatal(err)
	}
	td, err := st.Add(store.ScopeGlobal, "", "g task", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.New(filepath.Join(dir, "config.json"), store.DefaultTags)
	if err != nil {
		t.Fatal(err)
	}
	uiState := config.UIState{MainMode: "global"}
	uiState.Selected.Scope = "global"
	uiState.Selected.ID = td.ID
	if err := cfg.SaveUI(uiState); err != nil {
		t.Fatal(err)
	}
	m := NewMainModel(st, cfg, gitctx.Context{Branch: "global"}, false, DefaultKeyMap())
	if m.mode != viewGeneral {
		t.Fatalf("mode = %v, want %v", m.mode, viewGeneral)
	}
	e := m.currentEntry()
	if e == nil || e.IsHeader || e.Todo.ID != td.ID {
		t.Fatalf("unexpected selected entry: %#v", e)
	}
}

func TestRestoreUIStateDoesNotOverrideCurrentContextMode(t *testing.T) {
	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "todos.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.New(filepath.Join(dir, "config.json"), store.DefaultTags)
	if err != nil {
		t.Fatal(err)
	}
	uiState := config.UIState{MainMode: "all"}
	if err := cfg.SaveUI(uiState); err != nil {
		t.Fatal(err)
	}
	ctx := gitctx.Context{RepoRoot: "/repo", WorktreeRoot: "/repo/wt", Branch: "feat"}
	m := NewMainModel(st, cfg, ctx, false, DefaultKeyMap())
	if m.mode != viewContext {
		t.Fatalf("mode = %v, want %v", m.mode, viewContext)
	}
}
