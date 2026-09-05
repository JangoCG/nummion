package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JangoCG/nummion/internal/harness"
)

const legacySkillMarker = ".managed-by-lexware-cli"

// Only the exact skill shipped in v0.1.0 is eligible for automatic removal.
// Modified files, additional files, and foreign links remain untouched.
const legacySkillSHA256 = "f8aac390dabb622575e48a22f25ee68a574386aa7d2e1813f6f9032c62583f30"

func retireLegacySkills() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	baseline := filepath.Join(home, ".agents", "skills", "lexware")
	paths := []string{filepath.Join(home, ".claude", "skills", "lexware")}
	if codexHome := harness.CodexHome(); codexHome != "" {
		paths = append(paths, filepath.Join(codexHome, "skills", "lexware"))
	}
	// Retire the baseline last so Claude link provenance can still be checked.
	paths = append(paths, baseline)
	var notices []string
	for _, path := range paths {
		newPath := filepath.Join(filepath.Dir(path), "nummion")
		if !ownedSkillDir(newPath) || !harness.RegularSkillFile(filepath.Join(newPath, skillFilename)) {
			continue
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			target, linkErr := os.Readlink(path)
			if path == paths[0] && linkErr == nil && target == filepath.Join("..", "..", ".agents", "skills", "lexware") && pristineLegacySkill(baseline) {
				err = os.Remove(path)
			} else {
				err = fmt.Errorf("fremder oder angepasster Symlink")
			}
		} else if err == nil && pristineLegacySkill(path) {
			for _, name := range []string{skillFilename, installedVersionFile, legacySkillMarker} {
				if removeErr := os.Remove(filepath.Join(path, name)); removeErr != nil && !os.IsNotExist(removeErr) {
					err = removeErr
					break
				}
			}
			if err == nil {
				err = os.Remove(path)
			}
		} else if err == nil {
			err = fmt.Errorf("fremder oder angepasster Skill")
		}
		if err != nil {
			notices = append(notices, fmt.Sprintf("Alter Skill unter %s wurde nicht vollständig entfernt: %v; bitte manuell prüfen", path, err))
		}
	}
	return notices
}

func pristineLegacySkill(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || !harness.RegularSkillFile(filepath.Join(path, legacySkillMarker)) {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return false
		}
		switch entry.Name() {
		case skillFilename, installedVersionFile, legacySkillMarker:
		default:
			return false
		}
	}
	data, err := os.ReadFile(filepath.Join(path, skillFilename)) // #nosec G304 -- Fixed public skill filename under an operator-owned agent directory, checked above.
	return err == nil && fmt.Sprintf("%x", sha256.Sum256(data)) == legacySkillSHA256
}
