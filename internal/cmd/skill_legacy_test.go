package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installLegacyFixture(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile("testdata/skill-v0.1.0.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		skillFilename: string(data), legacySkillMarker: "managed", installedVersionFile: "0.1.0",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !pristineLegacySkill(dir) {
		t.Fatal("fixture must match the v0.1.0 skill checksum")
	}
}

func TestSkillInstallRetiresLegacyCopies(t *testing.T) {
	home := isolatedAgentHome(t, ".claude", ".codex")
	for _, agent := range []string{".agents", ".claude", ".codex"} {
		installLegacyFixture(t, filepath.Join(home, agent, "skills", "lexware"))
	}
	for run := 0; run < 2; run++ {
		output, err := executeCommand(t, "--json", "skill", "install")
		if err != nil || strings.Contains(output, "legacy_notices") {
			t.Fatalf("install: %s, %v", output, err)
		}
		for _, agent := range []string{".agents", ".claude", ".codex"} {
			if _, err := os.Lstat(filepath.Join(home, agent, "skills", "lexware")); !os.IsNotExist(err) {
				t.Fatalf("legacy skill still present for %s: %v", agent, err)
			}
			data, err := os.ReadFile(filepath.Join(home, agent, "skills", "nummion", skillFilename))
			if err != nil || !strings.Contains(string(data), "name: nummion") {
				t.Fatalf("new skill missing for %s: %v", agent, err)
			}
		}
	}
}

func TestSkillInstallRetiresLegacyClaudeLink(t *testing.T) {
	home := isolatedAgentHome(t, filepath.Join(".claude", "skills"))
	installLegacyFixture(t, filepath.Join(home, ".agents", "skills", "lexware"))
	oldLink := filepath.Join(home, ".claude", "skills", "lexware")
	if err := os.Symlink(filepath.Join("..", "..", ".agents", "skills", "lexware"), oldLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := executeCommand(t, "skill", "install"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(oldLink); !os.IsNotExist(err) {
		t.Fatalf("legacy link was not retired: %v", err)
	}
	if _, err := os.ReadFile(filepath.Join(home, ".claude", "skills", "nummion", skillFilename)); err != nil {
		t.Fatalf("new Claude skill is unreadable: %v", err)
	}
}

func TestSkillInstallPreservesCustomizedLegacySkills(t *testing.T) {
	for _, kind := range []string{"edited", "extra-file", "unmarked", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			home := isolatedAgentHome(t)
			legacy := filepath.Join(home, ".agents", "skills", "lexware")
			installLegacyFixture(t, legacy)
			var err error
			switch kind {
			case "edited":
				err = os.WriteFile(filepath.Join(legacy, skillFilename), []byte("custom skill"), 0o644)
			case "extra-file":
				err = os.WriteFile(filepath.Join(legacy, "notes.txt"), []byte("custom notes"), 0o644)
			case "unmarked":
				err = os.Remove(filepath.Join(legacy, legacySkillMarker))
			case "symlink":
				target := filepath.Join(home, "custom-skill")
				if err := os.Rename(legacy, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, legacy); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(filepath.Join(legacy, skillFilename))
			if err != nil {
				t.Fatal(err)
			}
			output, err := executeCommand(t, "--json", "skill", "install")
			if err != nil || !strings.Contains(output, "legacy_notices") {
				t.Fatalf("expected preservation notice: %s, %v", output, err)
			}
			after, err := os.ReadFile(filepath.Join(legacy, skillFilename))
			if err != nil || string(after) != string(before) {
				t.Fatalf("legacy skill changed: %v", err)
			}
			if kind == "extra-file" {
				if data, err := os.ReadFile(filepath.Join(legacy, "notes.txt")); err != nil || string(data) != "custom notes" {
					t.Fatalf("custom notes changed: %v", err)
				}
			}
		})
	}
}

func TestFailedSkillInstallKeepsLegacySkill(t *testing.T) {
	home := isolatedAgentHome(t, filepath.Join(".agents", "skills", "nummion"))
	legacy := filepath.Join(home, ".agents", "skills", "lexware")
	installLegacyFixture(t, legacy)
	if err := os.WriteFile(filepath.Join(home, ".agents", "skills", "nummion", skillFilename), []byte("foreign skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "skill", "install"); err == nil {
		t.Fatal("expected refusal to overwrite the destination")
	}
	if !pristineLegacySkill(legacy) {
		t.Fatal("legacy skill was removed before successful installation")
	}
}
