package skills

import (
	"strings"
	"testing"
)

func TestLexwareSkillContainsAgentSafetyRules(t *testing.T) {
	data, err := FS.ReadFile("lexware/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	for _, want := range []string{
		"num invoices list --year 2025 --json",
		"num invoices get <invoice_id> --json",
		"Never run `num auth set` unattended",
		"never pass `--token`",
		"--finalize",
		"--dry-run --json",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("embedded Lexware skill does not contain %q", want)
		}
	}
}
