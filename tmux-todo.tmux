#!/usr/bin/env bash

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Set default value only if the user has not already configured it.
tmux_default() {
  local val
  val="$(tmux show-option -gqv "$1")"
  if [ -z "$val" ]; then
    tmux set-option -gq "$1" "$2"
  fi
}

tmux_default @tmux-todo-bin "tmux-todo"
tmux_default @tmux-todo-popup-width "80%"
tmux_default @tmux-todo-popup-height "80%"
tmux_default @tmux-todo-peek-width "34%"
tmux_default @tmux-todo-peek-height "22%"
tmux_default @tmux-todo-quick-width "46%"
tmux_default @tmux-todo-quick-height "18%"
tmux_default @tmux-todo-focus-width "32%"
tmux_default @tmux-todo-focus-height "14%"
tmux_default @tmux-todo-strikethrough "on"
tmux_default @tmux-todo-focus-alert "off"
tmux_default @tmux-todo-focus-cooldown-sec "0"
tmux_default @tmux-todo-alert-duration-ms "5000"
tmux_default @tmux-todo-focus-duration-ms "2000"
tmux_default @tmux-todo-focus-on-context-switch "on"
tmux_default @tmux-todo-focus-include-global "off"
tmux_default @tmux-todo-bind-full "T"
tmux_default @tmux-todo-bind-peek "t"
tmux_default @tmux-todo-bind-quick "C-t"

bind_full="$(tmux show-option -gqv @tmux-todo-bind-full)"
bind_peek="$(tmux show-option -gqv @tmux-todo-bind-peek)"
bind_quick="$(tmux show-option -gqv @tmux-todo-bind-quick)"

bind_with_warning() {
  local table="$1"
  local key="$2"
  local popup_mode="$3"
  local no_prefix="${4:-0}"
  [ -z "$key" ] && return 0

  local existing
  existing="$(tmux list-keys -T "$table" 2>/dev/null | awk -v t="$table" -v k="$key" '$0 ~ (" -T " t " " k " ") {print; exit}')"
  if [ -n "$existing" ] && [[ "$existing" != *"tmux-todo/scripts/popup.sh"* ]]; then
    tmux display-message "tmux-todo: key conflict on $key in table $table (overriding existing binding)"
  fi
  if [ "$no_prefix" = "1" ]; then
    tmux bind-key -n "$key" run-shell "$CURRENT_DIR/scripts/popup.sh $popup_mode '#{pane_current_path}'"
  else
    tmux bind-key "$key" run-shell "$CURRENT_DIR/scripts/popup.sh $popup_mode '#{pane_current_path}'"
  fi
}

if [ -n "$bind_full" ]; then
  bind_with_warning "prefix" "$bind_full" "full" "0"
fi
if [ -n "$bind_peek" ]; then
  bind_with_warning "prefix" "$bind_peek" "peek" "0"
fi
if [ -n "$bind_quick" ]; then
  bind_with_warning "root" "$bind_quick" "quick" "1"
fi

focus_alert="$(tmux show-option -gqv @tmux-todo-focus-alert)"
# Remove stale tmux-todo focus hooks first, then add current one when enabled.
existing_hooks="$(tmux show-hooks -g pane-focus-in 2>/dev/null || true)"
while IFS= read -r line; do
  case "$line" in
    *focus-alert.sh*)
      idx="$(printf "%s" "$line" | sed -n 's/^pane-focus-in\[\([0-9][0-9]*\)\].*/\1/p')"
      if [ -n "$idx" ]; then
        tmux set-hook -gu "pane-focus-in[$idx]"
      fi
      ;;
  esac
done <<EOF
$existing_hooks
EOF

case "$(printf "%s" "$focus_alert" | tr '[:upper:]' '[:lower:]')" in
  on|true|1|yes)
    hook_cmd="run-shell '$CURRENT_DIR/scripts/focus-alert.sh \"#{pane_current_path}\"'"
    tmux set-hook -ag pane-focus-in "$hook_cmd"
    ;;
esac
