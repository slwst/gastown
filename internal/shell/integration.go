// ABOUTME: Shell integration installation and removal for Gas Town.
// ABOUTME: Manages the shell hook in RC files with safe block markers.

package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/state"
)

const (
	markerStart = "# --- Gas Town Integration (managed by gt) ---"
	markerEnd   = "# --- End Gas Town ---"
)

func hookSourceLine(shell string) string {
	if shell == "fish" {
		return fmt.Sprintf(`test -f "%s/shell-hook.fish"; and source "%s/shell-hook.fish"`,
			state.ConfigDir(), state.ConfigDir())
	}
	return fmt.Sprintf(`[[ -f "%s/shell-hook.sh" ]] && source "%s/shell-hook.sh"`,
		state.ConfigDir(), state.ConfigDir())
}

func Install() error {
	shell := DetectShell()
	rcPath := RCFilePath(shell)

	if err := writeHookScript(shell); err != nil {
		return fmt.Errorf("writing hook script: %w", err)
	}

	if err := addToRCFile(rcPath, shell); err != nil {
		return fmt.Errorf("updating %s: %w", rcPath, err)
	}

	return state.SetShellIntegration(shell)
}

func Remove() error {
	shell := DetectShell()
	rcPath := RCFilePath(shell)

	if err := removeFromRCFile(rcPath); err != nil {
		return fmt.Errorf("updating %s: %w", rcPath, err)
	}

	// Remove the appropriate hook script
	hookFile := "shell-hook.sh"
	if shell == "fish" {
		hookFile = "shell-hook.fish"
	}
	hookPath := filepath.Join(state.ConfigDir(), hookFile)
	if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing hook script: %w", err)
	}

	return nil
}

func DetectShell() string {
	shell := os.Getenv("SHELL")
	if strings.HasSuffix(shell, "fish") {
		return "fish"
	}
	if strings.HasSuffix(shell, "zsh") {
		return "zsh"
	}
	if strings.HasSuffix(shell, "bash") {
		return "bash"
	}
	return "zsh"
}

func RCFilePath(shell string) string {
	home, _ := os.UserHomeDir()
	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "conf.d", "gastown.fish")
	default:
		return filepath.Join(home, ".zshrc")
	}
}

func writeHookScript(shell string) error {
	dir := state.ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if shell == "fish" {
		hookPath := filepath.Join(dir, "shell-hook.fish")
		return os.WriteFile(hookPath, []byte(fishHookScript), 0644)
	}

	hookPath := filepath.Join(dir, "shell-hook.sh")
	return os.WriteFile(hookPath, []byte(shellHookScript), 0644)
}

func addToRCFile(path, shell string) error {
	// For fish conf.d, ensure parent directory exists
	if shell == "fish" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("creating fish conf.d directory: %w", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(data)

	if strings.Contains(content, markerStart) {
		return updateRCFile(path, content, shell)
	}

	block := fmt.Sprintf("\n%s\n%s\n%s\n", markerStart, hookSourceLine(shell), markerEnd)

	if len(data) > 0 {
		backupPath := path + ".gastown-backup"
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("writing backup: %w", err)
		}
	}

	return os.WriteFile(path, []byte(content+block), 0644)
}

func removeFromRCFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)

	startIdx := strings.Index(content, markerStart)
	if startIdx == -1 {
		return nil
	}

	endIdx := strings.Index(content[startIdx:], markerEnd)
	if endIdx == -1 {
		return nil
	}
	endIdx += startIdx + len(markerEnd)

	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	if startIdx > 0 && content[startIdx-1] == '\n' {
		startIdx--
	}

	newContent := content[:startIdx] + content[endIdx:]
	return os.WriteFile(path, []byte(newContent), 0644)
}

func updateRCFile(path, content, shell string) error {
	startIdx := strings.Index(content, markerStart)
	endIdx := strings.Index(content[startIdx:], markerEnd)
	if endIdx == -1 {
		return fmt.Errorf("malformed Gas Town block in %s", path)
	}
	endIdx += startIdx + len(markerEnd)

	block := fmt.Sprintf("%s\n%s\n%s", markerStart, hookSourceLine(shell), markerEnd)
	newContent := content[:startIdx] + block + content[endIdx:]

	return os.WriteFile(path, []byte(newContent), 0644)
}

var shellHookScript = `#!/bin/bash
# Gas Town Shell Integration
# Installed by: gt install --shell
# Location: ~/.config/gastown/shell-hook.sh

_gastown_enabled() {
    [[ -n "$GASTOWN_DISABLED" ]] && return 1
    [[ -n "$GASTOWN_ENABLED" ]] && return 0
    local state_file="$HOME/.local/state/gastown/state.json"
    [[ -f "$state_file" ]] && grep -q '"enabled":\s*true' "$state_file" 2>/dev/null
}

_gastown_ignored() {
    local dir="$PWD"
    while [[ "$dir" != "/" ]]; do
        [[ -f "$dir/.gastown-ignore" ]] && return 0
        dir="$(dirname "$dir")"
    done
    return 1
}

_gastown_already_asked() {
    local repo_root="$1"
    local asked_file="$HOME/.cache/gastown/asked-repos"
    [[ -f "$asked_file" ]] && grep -qF "$repo_root" "$asked_file" 2>/dev/null
}

_gastown_mark_asked() {
    local repo_root="$1"
    local asked_file="$HOME/.cache/gastown/asked-repos"
    mkdir -p "$(dirname "$asked_file")"
    echo "$repo_root" >> "$asked_file"
}

_gastown_offer_add() {
    local repo_root="$1"

    [[ "${GASTOWN_DISABLE_OFFER_ADD:-}" == "1" ]] && return 0
    _gastown_already_asked "$repo_root" && return 0
    
    [[ -t 0 ]] || return 0
    
    local repo_name
    repo_name=$(basename "$repo_root")
    
    echo ""
    echo -n "Add '$repo_name' to Gas Town? [y/N/never] "
    read -r response </dev/tty
    
    _gastown_mark_asked "$repo_root"
    
    case "$response" in
        y|Y|yes)
            echo "Adding to Gas Town..."
            local output
            output=$(gt rig quick-add "$repo_root" --yes 2>&1)
            local exit_code=$?
            echo "$output"
            
            if [[ $exit_code -eq 0 ]]; then
                local crew_path
                crew_path=$(echo "$output" | grep "^GT_CREW_PATH=" | cut -d= -f2)
                if [[ -n "$crew_path" && -d "$crew_path" ]]; then
                    echo ""
                    echo "Switching to crew workspace..."
                    cd "$crew_path" || true
                    # Re-run hook to set GT_TOWN_ROOT and GT_RIG
                    _gastown_hook
                fi
            fi
            ;;
        never)
            touch "$repo_root/.gastown-ignore"
            echo "Created .gastown-ignore - won't ask again for this repo."
            ;;
        *)
            echo "Skipped. Run 'gt rig quick-add' later to add manually."
            ;;
    esac
}

_gastown_hook() {
    local previous_exit_status=$?

    _gastown_enabled || {
        unset GT_TOWN_ROOT GT_RIG
        return $previous_exit_status
    }

    _gastown_ignored && {
        unset GT_TOWN_ROOT GT_RIG
        return $previous_exit_status
    }

    if ! git rev-parse --git-dir &>/dev/null; then
        unset GT_TOWN_ROOT GT_RIG
        return $previous_exit_status
    fi

    local repo_root
    repo_root=$(git rev-parse --show-toplevel 2>/dev/null) || {
        unset GT_TOWN_ROOT GT_RIG
        return $previous_exit_status
    }

    local cache_file="$HOME/.cache/gastown/rigs.cache"
    if [[ -f "$cache_file" ]]; then
        local cached
        cached=$(grep "^${repo_root}:" "$cache_file" 2>/dev/null)
        if [[ -n "$cached" ]]; then
            eval "${cached#*:}"
            return $previous_exit_status
        fi
    fi

    if command -v gt &>/dev/null; then
        local detect_output
        detect_output=$(gt rig detect "$repo_root" 2>/dev/null)
        eval "$detect_output"
        
        if [[ -n "$GT_TOWN_ROOT" ]]; then
            (gt rig detect --cache "$repo_root" &>/dev/null &)
        elif [[ -n "$_GASTOWN_OFFER_ADD" ]]; then
            _gastown_offer_add "$repo_root"
            unset _GASTOWN_OFFER_ADD
        fi
    fi

    return $previous_exit_status
}

_gastown_chpwd_hook() {
    _GASTOWN_OFFER_ADD=1
    _gastown_hook
}

case "${SHELL##*/}" in
    zsh)
        autoload -Uz add-zsh-hook
        add-zsh-hook chpwd _gastown_chpwd_hook
        add-zsh-hook precmd _gastown_hook
        ;;
    bash)
        if [[ ";${PROMPT_COMMAND[*]:-};" != *";_gastown_hook;"* ]]; then
            PROMPT_COMMAND="_gastown_chpwd_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
        fi
        ;;
esac

_gastown_hook
`

var fishHookScript = `# Gas Town Shell Integration (fish)
# Installed by: gt shell install
# Location: ~/.config/gastown/shell-hook.fish

function _gastown_enabled
    if set -q GASTOWN_DISABLED
        return 1
    end
    if set -q GASTOWN_ENABLED
        return 0
    end
    set -l state_file "$HOME/.local/state/gastown/state.json"
    if test -f "$state_file"
        command grep -q '"enabled":\s*true' "$state_file" 2>/dev/null
        return $status
    end
    return 1
end

function _gastown_ignored
    set -l dir $PWD
    while test "$dir" != /
        if test -f "$dir/.gastown-ignore"
            return 0
        end
        set dir (dirname "$dir")
    end
    return 1
end

function _gastown_already_asked
    set -l repo_root $argv[1]
    set -l asked_file "$HOME/.cache/gastown/asked-repos"
    if test -f "$asked_file"
        command grep -qF "$repo_root" "$asked_file" 2>/dev/null
        return $status
    end
    return 1
end

function _gastown_mark_asked
    set -l repo_root $argv[1]
    set -l asked_file "$HOME/.cache/gastown/asked-repos"
    mkdir -p (dirname "$asked_file")
    echo "$repo_root" >>"$asked_file"
end

function _gastown_offer_add
    set -l repo_root $argv[1]

    if test "$GASTOWN_DISABLE_OFFER_ADD" = 1
        return 0
    end
    if _gastown_already_asked "$repo_root"
        return 0
    end

    # Only prompt in interactive terminals
    if not isatty stdin
        return 0
    end

    set -l repo_name (basename "$repo_root")

    echo ""
    read -P "Add '$repo_name' to Gas Town? [y/N/never] " response

    _gastown_mark_asked "$repo_root"

    switch "$response"
        case y Y yes
            echo "Adding to Gas Town..."
            set -l output (gt rig quick-add "$repo_root" --yes 2>&1)
            set -l exit_code $status
            echo "$output"

            if test $exit_code -eq 0
                set -l crew_path (echo "$output" | command grep "^GT_CREW_PATH=" | cut -d= -f2)
                if test -n "$crew_path" -a -d "$crew_path"
                    echo ""
                    echo "Switching to crew workspace..."
                    cd "$crew_path"
                    _gastown_hook
                end
            end
        case never
            touch "$repo_root/.gastown-ignore"
            echo "Created .gastown-ignore - won't ask again for this repo."
        case '*'
            echo "Skipped. Run 'gt rig quick-add' later to add manually."
    end
end

function _gastown_hook
    set -l previous_exit_status $status

    if not _gastown_enabled
        set -e GT_TOWN_ROOT
        set -e GT_RIG
        return $previous_exit_status
    end

    if _gastown_ignored
        set -e GT_TOWN_ROOT
        set -e GT_RIG
        return $previous_exit_status
    end

    if not command git rev-parse --git-dir >/dev/null 2>&1
        set -e GT_TOWN_ROOT
        set -e GT_RIG
        return $previous_exit_status
    end

    set -l repo_root (command git rev-parse --show-toplevel 2>/dev/null)
    if test $status -ne 0
        set -e GT_TOWN_ROOT
        set -e GT_RIG
        return $previous_exit_status
    end

    set -l cache_file "$HOME/.cache/gastown/rigs.cache"
    if test -f "$cache_file"
        set -l cached (command grep "^$repo_root:" "$cache_file" 2>/dev/null)
        if test -n "$cached"
            # Parse KEY=VALUE pairs from cached line (after the colon)
            set -l assignments (string replace -- "$repo_root:" "" "$cached")
            for assignment in (string split " " "$assignments")
                set -l parts (string split "=" "$assignment")
                if test (count $parts) -ge 2
                    set -l key $parts[1]
                    set -l val (string join "=" $parts[2..-1])
                    # Strip surrounding quotes if present
                    set val (string trim --chars='"' "$val")
                    # Export as global variable
                    set -gx $key $val
                end
            end
            return $previous_exit_status
        end
    end

    if command -v gt >/dev/null 2>&1
        set -l detect_output (gt rig detect "$repo_root" 2>/dev/null)
        # Parse KEY=VALUE export lines from detect output
        for line in (string split \n "$detect_output")
            if string match -rq '^export ' "$line"
                set line (string replace 'export ' '' "$line")
            end
            set -l parts (string split "=" "$line")
            if test (count $parts) -ge 2
                set -l key $parts[1]
                set -l val (string join "=" $parts[2..-1])
                set val (string trim --chars='"' "$val")
                set -gx $key $val
            end
        end

        if set -q GT_TOWN_ROOT; and test -n "$GT_TOWN_ROOT"
            command gt rig detect --cache "$repo_root" >/dev/null 2>&1 &
        else if set -q _GASTOWN_OFFER_ADD
            _gastown_offer_add "$repo_root"
            set -e _GASTOWN_OFFER_ADD
        end
    end

    return $previous_exit_status
end

function _gastown_chpwd_hook --on-variable PWD
    set -g _GASTOWN_OFFER_ADD 1
    _gastown_hook
end

# Run hook on source (initial shell setup)
_gastown_hook
`
