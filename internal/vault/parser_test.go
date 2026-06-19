package vault

import (
	"reflect"
	"testing"
)

func TestParseNote_NoFrontmatter(t *testing.T) {
	note, err := ParseNote("Inbox/Idea.md", []byte("Just a plain note.\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.Title != "Idea" {
		t.Errorf("Title = %q, want %q", note.Title, "Idea")
	}
	if len(note.Frontmatter) != 0 {
		t.Errorf("Frontmatter = %v, want empty", note.Frontmatter)
	}
	if note.Body != "Just a plain note.\n" {
		t.Errorf("Body = %q", note.Body)
	}
}

func TestParseNote_FrontmatterAndTitle(t *testing.T) {
	content := []byte("---\ntitle: Custom Title\ntags: [foo, bar]\n---\nBody text.\n")
	note, err := ParseNote("Notes/page.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", note.Title, "Custom Title")
	}
	if note.Body != "Body text.\n" {
		t.Errorf("Body = %q", note.Body)
	}
	wantTags := []string{"foo", "bar"}
	if !reflect.DeepEqual(note.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", note.Tags, wantTags)
	}
}

func TestParseNote_FrontmatterStringTags(t *testing.T) {
	content := []byte("---\ntags: \"#foo, #bar\"\n---\nBody.\n")
	note, err := ParseNote("page.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantTags := []string{"foo", "bar"}
	if !reflect.DeepEqual(note.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", note.Tags, wantTags)
	}
}

func TestParseNote_UnterminatedFrontmatter(t *testing.T) {
	content := []byte("---\ntitle: Oops\nno closing delimiter\n")
	note, err := ParseNote("page.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(note.Frontmatter) != 0 {
		t.Errorf("Frontmatter = %v, want empty (unterminated block treated as body)", note.Frontmatter)
	}
	if note.Title != "page" {
		t.Errorf("Title = %q, want fallback %q", note.Title, "page")
	}
}

func TestParseNote_Links(t *testing.T) {
	content := []byte("See [[Other Note]] and [[Other Note|alias]].\n" +
		"Embed: ![[image.png]]\n" +
		"Block ref: [[Other Note^abc123]]\n" +
		"Embedded block: ![[Other Note^abc123]]\n")
	note, err := ParseNote("page.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []Link{
		{TargetRaw: "Other Note", Kind: LinkWiki},
		{TargetRaw: "Other Note|alias", Kind: LinkWiki},
		{TargetRaw: "image.png", Kind: LinkEmbed},
		{TargetRaw: "Other Note^abc123", Kind: LinkBlockRef},
		{TargetRaw: "Other Note^abc123", Kind: LinkEmbed},
	}
	if !reflect.DeepEqual(note.Links, want) {
		t.Errorf("Links = %+v, want %+v", note.Links, want)
	}
}

func TestParseNote_InlineTags(t *testing.T) {
	content := []byte("# Heading\n\nThis has #project/amethyst and #2026 (not a tag) and a C#sharp mention.\n")
	note, err := ParseNote("page.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"project/amethyst"}
	if !reflect.DeepEqual(note.Tags, want) {
		t.Errorf("Tags = %v, want %v", note.Tags, want)
	}
}

func TestParseNote_MergesFrontmatterAndInlineTagsDeduped(t *testing.T) {
	content := []byte("---\ntags: [shared]\n---\nBody with #shared and #extra.\n")
	note, err := ParseNote("page.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"shared", "extra"}
	if !reflect.DeepEqual(note.Tags, want) {
		t.Errorf("Tags = %v, want %v", note.Tags, want)
	}
}
