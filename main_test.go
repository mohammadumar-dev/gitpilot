package main

import (
	"strings"
	"testing"
)

func TestSemanticSummaryIncludesSymbolsFromLargeNewFile(t *testing.T) {
	var diff strings.Builder
	diff.WriteString("diff --git a/main.go b/main.go\n")
	diff.WriteString("new file mode 100644\n")
	diff.WriteString("--- /dev/null\n")
	diff.WriteString("+++ b/main.go\n")
	diff.WriteString("@@ -0,0 +1,220 @@\n")
	diff.WriteString("+package main\n")
	for i := 0; i < 120; i++ {
		diff.WriteString("+// filler line\n")
	}
	diff.WriteString("+func generateUsefulCommitMessage() string {\n")
	diff.WriteString("+\treturn \"ok\"\n")
	diff.WriteString("+}\n")

	change := FileChange{FileName: "main.go", Status: "??", Diff: diff.String()}
	context := buildFileChangeContext(change, 1200)

	if !strings.Contains(context, "Change type: new file") {
		t.Fatalf("expected new file context, got:\n%s", context)
	}
	if !strings.Contains(context, "func generateUsefulCommitMessage") {
		t.Fatalf("expected semantic symbol from large new file, got:\n%s", context)
	}
}

func TestCommitMessageValidationRejectsNullAndGenericSubjects(t *testing.T) {
	invalid := []string{
		"feat: null handle",
		"fix: update files",
		"docs: changes",
		"add useful context",
	}
	for _, message := range invalid {
		if isUsableCommitMessage(message) {
			t.Fatalf("expected %q to be rejected", message)
		}
	}

	if !isUsableCommitMessage("feat: add semantic diff summaries") {
		t.Fatal("expected specific conventional subject to be accepted")
	}
}

func TestFallbackCommitMessageUsesAreaPrefix(t *testing.T) {
	change := FileChange{
		FileName: "README.md",
		Status:   "M",
		Diff:     "diff --git a/README.md b/README.md\n@@ -1 +1 @@\n-old\n+new\n",
	}

	message := fallbackCommitMessage([]FileChange{change})
	if !strings.HasPrefix(message, "docs:") {
		t.Fatalf("expected docs fallback prefix, got %q", message)
	}
}
