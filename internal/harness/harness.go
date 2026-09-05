// Package harness detects coding agents and describes their skill locations.
package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SkillOwnershipMarker marks a skill directory as written by Nummion.
const SkillOwnershipMarker = ".managed-by-nummion"

// SkillDirOwned reports whether Nummion wrote the skill directory at dir.
func SkillDirOwned(dir string) bool {
	return dir != "" && RegularSkillFile(filepath.Join(dir, SkillOwnershipMarker))
}

// RegularSkillFile only accepts a regular final path component. It deliberately
// rejects a symlinked file so installers never write through an uninspected link.
func RegularSkillFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

// DetectClaude reports whether Claude Code has a home directory or executable.
func DetectClaude() bool {
	home, err := os.UserHomeDir()
	if err == nil {
		if info, statErr := os.Stat(filepath.Join(filepath.Clean(home), ".claude")); statErr == nil && info.IsDir() {
			return true
		}
	}
	return findAgentBinary("claude") != ""
}

// DetectCodex reports whether Codex has a home directory or executable.
func DetectCodex() bool {
	if home := CodexHome(); home != "" {
		if info, err := os.Stat(home); err == nil && info.IsDir() {
			return true
		}
	}
	return findAgentBinary("codex") != ""
}

func findAgentBinary(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	candidate := filepath.Join(filepath.Clean(home), ".local", "bin", name)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

// CodexHome returns $CODEX_HOME or ~/.codex.
func CodexHome() string {
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		return filepath.Clean(codexHome)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(filepath.Clean(home), ".codex")
}

// CodexSkillPath returns where Codex reads the Nummion skill from.
func CodexSkillPath() string {
	home := CodexHome()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "skills", "nummion", "SKILL.md")
}
