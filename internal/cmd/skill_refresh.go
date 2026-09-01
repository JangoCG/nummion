package cmd

import (
	"os"
	"path/filepath"

	"lexware-cli/internal/harness"
	"lexware-cli/skills"
)

// refreshSkillsIfVersionChanged keeps already-installed managed copies in
// sync after a CLI upgrade. It never installs a new skill and never touches an
// unmarked directory or symlink.
func refreshSkillsIfVersionChanged(version string) bool {
	if version == "" || version == "dev" || !baselineSkillInstalled() || installedSkillVersion() == version {
		return false
	}
	embedded, err := skills.FS.ReadFile("lexware/SKILL.md")
	if err != nil {
		return false
	}

	updated := false
	failed := false
	for _, location := range skillRefreshLocations() {
		info, statErr := os.Lstat(location)
		if statErr != nil {
			if !os.IsNotExist(statErr) {
				failed = true
			}
			continue
		}
		if !info.Mode().IsRegular() || isSymlink(filepath.Dir(location)) {
			continue
		}
		if !ownedSkillDir(filepath.Dir(location)) {
			continue
		}
		if err := writeSkillFile(location, embedded); err == nil {
			updated = true
		} else {
			failed = true
		}
	}
	if failed || !updated {
		return false
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	baselineDir := filepath.Join(filepath.Clean(home), ".agents", "skills", "lexware")
	if !ownedSkillDir(baselineDir) {
		return false
	}
	return writeSkillFile(filepath.Join(baselineDir, installedVersionFile), []byte(version)) == nil
}

func skillRefreshLocations() []string {
	var locations []string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		locations = append(locations,
			filepath.Join(filepath.Clean(home), ".agents", "skills", "lexware", skillFilename),
			filepath.Join(filepath.Clean(home), ".claude", "skills", "lexware", skillFilename),
		)
	}
	if codexPath := harness.CodexSkillPath(); codexPath != "" {
		locations = append(locations, codexPath)
	}
	return locations
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
