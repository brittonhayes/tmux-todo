# tmux-todo

Context-aware tmux todo manager with:

- Full popup manager (`tui`)
- Compact peek popup (`peek`)
- Quick-add popup (`quick`)
- Focus alert for high-priority context tasks (`peek-high`)
- Global + context-scoped tasks (auto-detected from git repo/worktree/branch)
- Nested tasks, priorities, and tags
- AI-friendly CLI with JSON output

Requires a [Nerd Font](https://www.nerdfonts.com/) for icon rendering.

## Install

### Prerequisites

- tmux 3.2+ (for `display-popup` support)
- Go 1.22+ (to build the binary)
- A Nerd Font patched terminal font

### 1. Install the binary

TPM does **not** build the Go binary. Install it separately using one of these methods.

**Option A — `go install` (recommended)**

```bash
go install github.com/0x-JP/tmux-todo/cmd/tmux-todo@latest
```

**Option B — Build from source**

```bash
cd /path/to/tmux-todo
go build -o ~/.local/bin/tmux-todo ./cmd/tmux-todo
```

**Option C — Download a release binary**

When release assets are available, download the binary for your platform and place it on PATH (e.g. `~/.local/bin/tmux-todo`).

Make sure the binary location is on PATH for tmux sessions, or set `@tmux-todo-bin` to the absolute path in `.tmux.conf`.

### 2. Install the plugin via TPM

Add to `~/.tmux.conf` (before the TPM loader):

```tmux
set -g @plugin '0x-JP/tmux-todo'
```

Then install with `prefix + I` inside tmux, or reload:

```bash
tmux source-file ~/.tmux.conf
```

**Local development with TPM:**

```bash
ln -sfn /path/to/tmux-todo ~/.tmux/plugins/tmux-todo
```

### macOS note

macOS may set `com.apple.provenance` on cloned files, blocking script execution (exit code 126). Fix with:

```bash
sudo xattr -dr com.apple.provenance ~/.tmux/plugins/tmux-todo
```

Or build the binary locally (Option B above) to avoid the attribute entirely.

## Configuration

Add any of these to `~/.tmux.conf` **before** the TPM `run` line. Only set the options you want to change — all have sensible defaults.

```tmux
# Binary location (default: "tmux-todo", found via PATH)
set -g @tmux-todo-bin "$HOME/.local/bin/tmux-todo"

# Popup sizing
set -g @tmux-todo-popup-width "80%"     # default: 80%
set -g @tmux-todo-popup-height "80%"    # default: 80%
set -g @tmux-todo-peek-width "34%"      # default: 34%
set -g @tmux-todo-peek-height "22%"     # default: 22%
set -g @tmux-todo-quick-width "46%"     # default: 46%
set -g @tmux-todo-quick-height "18%"    # default: 18%
set -g @tmux-todo-focus-width "32%"     # default: 32%
set -g @tmux-todo-focus-height "14%"    # default: 14%

# Rendering
set -g @tmux-todo-strikethrough "on"    # default: on

# Focus alert (popup on pane switch when high-priority tasks exist)
set -g @tmux-todo-focus-alert "on"              # default: off
set -g @tmux-todo-focus-on-context-switch "on"  # default: on
set -g @tmux-todo-focus-include-global "off"    # default: off
set -g @tmux-todo-focus-cooldown-sec "0"        # default: 0
set -g @tmux-todo-alert-duration-ms "5000"      # default: 5000
set -g @tmux-todo-focus-duration-ms "2000"      # default: 2000

# Keybindings (set to "" to disable)
set -g @tmux-todo-bind-full "T"         # default: T (prefix + T)
set -g @tmux-todo-bind-peek "t"         # default: t (prefix + t)
set -g @tmux-todo-bind-quick "C-t"      # default: C-t (no-prefix key)

# TPM loader (keep at bottom)
run '~/.tmux/plugins/tpm/tpm'
```

You can also bind keys manually if you prefer:

```tmux
bind-key T   run-shell "~/.tmux/plugins/tmux-todo/scripts/popup.sh full '#{pane_current_path}'"
bind-key t   run-shell "~/.tmux/plugins/tmux-todo/scripts/popup.sh peek '#{pane_current_path}'"
bind-key -n C-t run-shell "~/.tmux/plugins/tmux-todo/scripts/popup.sh quick '#{pane_current_path}'"
```

## TUI Keys

| Key | Action |
|-----|--------|
| `?` | Help overlay |
| `tab` | Cycle scope (context / global / all-contexts) |
| `/` | Filter input (`p:high tag:blocked`) |
| `a` | Add (enter saves, tab for priority/tags) |
| `c` | Add child task |
| `e` | Edit selected task |
| `g` | Tag picker for selected task |
| `G` | Global tag manager |
| `1` / `2` / `3` | Set priority (high / med / low) |
| `!` | Clear priority |
| `b` / `r` | Toggle blocked / review tag |
| `space` | Toggle done |
| `d` | Delete selected task |
| `j` / `k` / arrows | Move cursor |
| `[` / `]` | Previous / next context |
| `q` | Quit |

## Quick Add Popup

Default binding: `Ctrl-t` (no prefix required).

Input grammar:

```
task name                          -> add to current context
global | task name                 -> add to Global context
task name | p=1                    -> high priority (1=high, 2=med, 3=low)
task name | p=high                 -> same as above
task name | t=blocked,review       -> add tags inline
global | task name | p=2 | t=review  -> combine all overrides
```

## CLI Reference

Run `tmux-todo help` for built-in command help.

### Add

```bash
tmux-todo add --text "Investigate flaky test"
tmux-todo add --scope global --text "Book travel"
tmux-todo add --text "Child task" --parent <PARENT_ID>
tmux-todo add --text "Fix CI" --priority high --tag blocked --tag review
```

### List

```bash
tmux-todo list
tmux-todo list --scope global
tmux-todo list --scope all --all
tmux-todo list --priority high
tmux-todo list --tag blocked --sort created
```

### State Changes

```bash
tmux-todo done --id <TODO_ID>
tmux-todo undone --id <TODO_ID>
tmux-todo edit --id <TODO_ID> --text "Updated" --priority med --tags blocked,review
tmux-todo delete --id <TODO_ID>
tmux-todo move --id <TODO_ID> --from-scope context --to-scope global
tmux-todo reparent --id <TODO_ID> --parent <PARENT_ID>
```

### Read Helpers

```bash
tmux-todo get --id <TODO_ID>
tmux-todo context-key
tmux-todo has-high
tmux-todo has-high --context-only
tmux-todo summary --json
tmux-todo doctor --json
tmux-todo export --out ~/tmux-todo-export.json --json
tmux-todo clear-all --yes --json
```

`clear-all` is destructive and requires `--yes`.

### Tag Registry

```bash
tmux-todo tags list
tmux-todo tags add review
tmux-todo tags remove whatever --global-clean
```

`tags remove --global-clean` removes the tag from the registry and all existing todos.

### JSON Output

Add `--json` to any command for machine-readable output:

```bash
tmux-todo add --text "fix flaky test" --priority high --json
tmux-todo list --scope all --all --json
tmux-todo done --id <TODO_ID> --json
tmux-todo tags list --json
```

## Data and Config Paths

**Todo data:**

1. `--data /custom/path.json` (CLI flag)
2. `$XDG_STATE_HOME/tmux-todo/todos.json`
3. `~/.local/state/tmux-todo/todos.json` (fallback)

**User config (tag registry + UI state):**

1. `$XDG_CONFIG_HOME/tmux-todo/config.json`
2. `~/.config/tmux-todo/config.json` (fallback)

For consistent paths across machines, you can set in `.tmux.conf`:

```tmux
set-environment -g XDG_STATE_HOME "$HOME/.local/state"
set-environment -g XDG_CONFIG_HOME "$HOME/.config"
```

## Agent Integration (Claude Code / Codex)

For AI agents that manage tasks via CLI, add to `~/.claude/CLAUDE.md`:

```markdown
When managing my tasks, use tmux-todo CLI (never edit todo files directly).

Workflow:
1. tmux-todo context-key                              # detect context
2. tmux-todo add --text "<task>" --priority high --json  # add tasks
3. tmux-todo done --id <id> --json                     # mark complete
4. tmux-todo list --scope context --json               # read tasks

Rules:
- Prefer --json for machine-readable operations.
- Keep task updates in current context unless told otherwise.
- Use canonical tags: blocked, review
```

Each session auto-scopes to its cwd/worktree, so parallel sessions work without conflict.

## Troubleshooting

### Exit code 126 on macOS

Scripts are blocked by Gatekeeper. See [macOS note](#macos-note) above.

### Focus alert does not appear

1. Check the option is set:
   ```bash
   tmux show-option -gqv @tmux-todo-focus-alert   # should be "on"
   ```
2. Check the hook is registered:
   ```bash
   tmux show-hooks -g pane-focus-in
   ```
3. Confirm a high-priority task exists in the current context:
   ```bash
   tmux-todo has-high --context-only
   ```

### Popup too big or small

Adjust the sizing options in your `.tmux.conf` — see [Configuration](#configuration).

## Development

```bash
go test ./...
go build ./cmd/tmux-todo
```
