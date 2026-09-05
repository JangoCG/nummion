package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolatedAgentHome(t *testing.T, dirs ...string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CODEX_HOME", "")
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func TestSkillCommandPrintsEmbeddedSkill(t *testing.T) {
	output, err := executeCommand(t, "skill")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "name: lexware") || !strings.Contains(output, "num invoices get <invoice_id> --json") {
		t.Fatalf("unexpected skill output: %q", output)
	}
}

func TestSkillInstallWritesBaseline(t *testing.T) {
	home := isolatedAgentHome(t)
	output, err := executeCommand(t, "--json", "skill", "install")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Fatalf("output = %q", output)
	}
	skillDir := filepath.Join(home, ".agents", "skills", "lexware")
	for _, name := range []string{skillFilename, installedVersionFile, ownershipMarkerFile} {
		if info, err := os.Lstat(filepath.Join(skillDir, name)); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("%s missing or not regular: %v, %v", name, info, err)
		}
	}
	version, err := os.ReadFile(filepath.Join(skillDir, installedVersionFile))
	if err != nil || string(version) != "test" {
		t.Fatalf("installed version = %q, %v", version, err)
	}
}

func TestSkillInstallLinksClaudeAndCopiesCodex(t *testing.T) {
	home := isolatedAgentHome(t, ".claude")
	codexHome := filepath.Join(home, "codex-home")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", codexHome)

	if _, err := executeCommand(t, "--json", "skill", "install"); err != nil {
		t.Fatal(err)
	}

	claudePath := filepath.Join(home, ".claude", "skills", "lexware")
	info, err := os.Lstat(claudePath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Claude skill is not a symlink: %v, %v", info, err)
	}
	if target, err := os.Readlink(claudePath); err != nil || target != claudeSkillLinkTarget {
		t.Fatalf("Claude target = %q, %v", target, err)
	}

	codexDir := filepath.Join(codexHome, "skills", "lexware")
	if !ownedSkillDir(codexDir) {
		t.Fatalf("Codex skill directory is not marked as managed: %s", codexDir)
	}
	if info, err := os.Lstat(filepath.Join(codexDir, skillFilename)); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("Codex skill is missing or not regular: %v, %v", info, err)
	}
}

func TestSkillInstallCopyFallbackIsIdempotent(t *testing.T) {
	home := isolatedAgentHome(t, ".claude")
	originalSymlink := makeSkillSymlink
	makeSkillSymlink = func(string, string) error { return errors.New("symlinks unavailable") }
	t.Cleanup(func() { makeSkillSymlink = originalSymlink })

	for run := 1; run <= 2; run++ {
		if _, err := executeCommand(t, "--json", "skill", "install"); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		copyPath := filepath.Join(home, ".claude", "skills", "lexware")
		info, err := os.Lstat(copyPath)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("run %d: expected copied directory, got %v, %v", run, info, err)
		}
		if !isManagedSkillCopy(copyPath) {
			t.Fatalf("run %d: copied directory is not recognized as managed", run)
		}
	}
}

func TestSkillInstallPreservesUnmanagedBaseline(t *testing.T) {
	home := isolatedAgentHome(t)
	skillDir := filepath.Join(home, ".agents", "skills", "lexware")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := []byte("# my own Lexware skill\n")
	if err := os.WriteFile(filepath.Join(skillDir, skillFilename), custom, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := executeCommand(t, "skill", "install"); err == nil {
		t.Fatal("install replaced an unmanaged baseline")
	}
	data, err := os.ReadFile(filepath.Join(skillDir, skillFilename))
	if err != nil || string(data) != string(custom) {
		t.Fatalf("custom baseline changed: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(skillDir, ownershipMarkerFile)); !os.IsNotExist(err) {
		t.Fatalf("unmanaged baseline was claimed: %v", err)
	}
}

func TestSkillInstallPreservesForeignClaudeSkill(t *testing.T) {
	home := isolatedAgentHome(t)
	skillDir := filepath.Join(home, ".claude", "skills", "lexware")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := []byte("# custom Claude skill\n")
	if err := os.WriteFile(filepath.Join(skillDir, skillFilename), custom, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := executeCommand(t, "skill", "install"); err == nil {
		t.Fatal("install replaced an unmanaged Claude skill")
	}
	data, err := os.ReadFile(filepath.Join(skillDir, skillFilename))
	if err != nil || string(data) != string(custom) {
		t.Fatalf("custom Claude skill changed: %q, %v", data, err)
	}
}

func TestSkillInstallPreservesForeignCodexSkill(t *testing.T) {
	home := isolatedAgentHome(t)
	codexHome := filepath.Join(home, "codex-home")
	skillDir := filepath.Join(codexHome, "skills", "lexware")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", codexHome)
	custom := []byte("# custom Codex skill\n")
	if err := os.WriteFile(filepath.Join(skillDir, skillFilename), custom, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := executeCommand(t, "skill", "install"); err == nil {
		t.Fatal("install replaced an unmanaged Codex skill")
	}
	data, err := os.ReadFile(filepath.Join(skillDir, skillFilename))
	if err != nil || string(data) != string(custom) {
		t.Fatalf("custom Codex skill changed: %q, %v", data, err)
	}
}

func TestSkillRefreshUpdatesOnlyManagedCopies(t *testing.T) {
	home := isolatedAgentHome(t)
	codexHome := filepath.Join(home, "codex-home")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", codexHome)
	if _, err := executeCommand(t, "skill", "install"); err != nil {
		t.Fatal(err)
	}

	baseline := filepath.Join(home, ".agents", "skills", "lexware", skillFilename)
	codex := filepath.Join(codexHome, "skills", "lexware", skillFilename)
	for _, path := range []string{baseline, codex} {
		if err := os.WriteFile(path, []byte("stale"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	versionPath := filepath.Join(home, ".agents", "skills", "lexware", installedVersionFile)
	if err := os.WriteFile(versionPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !refreshSkillsIfVersionChanged("new") {
		t.Fatal("managed skills were not refreshed")
	}
	for _, path := range []string{baseline, codex} {
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), "name: lexware") {
			t.Fatalf("%s was not refreshed: %q, %v", path, data, err)
		}
	}
	version, err := os.ReadFile(versionPath)
	if err != nil || string(version) != "new" {
		t.Fatalf("refreshed version = %q, %v", version, err)
	}
}
