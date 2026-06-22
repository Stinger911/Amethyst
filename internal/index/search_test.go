package index

import (
	"strings"
	"testing"
)

func TestSearch_MatchesAndRanks(t *testing.T) {
	root := t.TempDir()
	writeVaultFile(t, root, "Apple.md", "# Apple\n\nApple pie is a dessert made with apples.\n")
	writeVaultFile(t, root, "Banana.md", "# Banana\n\nBanana bread is a dessert made with bananas.\n")
	writeVaultFile(t, root, "Unrelated.md", "# Unrelated\n\nNothing to do with fruit.\n")

	db := openTestDB(t)
	if _, err := ColdScan(db, root); err != nil {
		t.Fatalf("ColdScan: %v", err)
	}

	matchQuery, err := ToFTSQuery("banana")
	if err != nil {
		t.Fatalf("ToFTSQuery: %v", err)
	}
	results, err := Search(db, matchQuery, 20, "", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].Path != "Banana.md" {
		t.Fatalf("results = %+v, want only Banana.md", results)
	}
}

func TestSearch_SnippetMarkersAreApplied(t *testing.T) {
	root := t.TempDir()
	writeVaultFile(t, root, "Apple.md", "# Apple\n\nApple pie is a dessert.\n")

	db := openTestDB(t)
	if _, err := ColdScan(db, root); err != nil {
		t.Fatalf("ColdScan: %v", err)
	}

	matchQuery, err := ToFTSQuery("dessert")
	if err != nil {
		t.Fatalf("ToFTSQuery: %v", err)
	}
	results, err := Search(db, matchQuery, 20, "<mark>", "</mark>")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v, want 1", results)
	}
	if results[0].Snippet == "" {
		t.Fatal("Snippet is empty")
	}
	if !strings.Contains(results[0].Snippet, "<mark>") {
		t.Errorf("Snippet = %q, want it to contain <mark>", results[0].Snippet)
	}
}

func TestToFTSQuery_RejectsEmptyInput(t *testing.T) {
	if _, err := ToFTSQuery("   "); err == nil {
		t.Error("ToFTSQuery(whitespace) = nil error, want one for empty input")
	}
}
