package render

import (
	"strings"
	"testing"
)

func TestRender_ResolvedWikiLinkBecomesAnchor(t *testing.T) {
	resolve := func(raw string) (string, bool) {
		if raw == "Second Note" {
			return "Folder/Second Note.md", true
		}
		return "", false
	}
	html, err := Render("See [[Second Note]] for more.", resolve)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, `<a class="wikilink" href="/note/Folder/Second%20Note.md">Second Note</a>`) {
		t.Errorf("html = %q, missing expected anchor", html)
	}
}

func TestRender_AliasUsesDisplayTextButResolvesByFullRaw(t *testing.T) {
	resolve := func(raw string) (string, bool) {
		if raw == "Second Note|click here" {
			return "Second Note.md", true
		}
		return "", false
	}
	html, err := Render("[[Second Note|click here]]", resolve)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, `href="/note/Second%20Note.md"`) || !strings.Contains(html, `>click here<`) {
		t.Errorf("html = %q, want resolved href with alias display text", html)
	}
}

func TestRender_UnresolvedWikiLinkBecomesMissingSpan(t *testing.T) {
	html, err := Render("[[Nowhere]]", func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, `<span class="wikilink wikilink-missing">Nowhere</span>`) {
		t.Errorf("html = %q, want missing span", html)
	}
}

func TestRender_EmbedGetsEmbedClass(t *testing.T) {
	resolve := func(raw string) (string, bool) { return "Pic.png", true }
	html, err := Render("![[Pic.png]]", resolve)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, `class="wikilink wikilink-embed"`) {
		t.Errorf("html = %q, want embed class", html)
	}
}

func TestRender_NilResolverRendersEverythingAsMissing(t *testing.T) {
	html, err := Render("[[Anything]]", nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, "wikilink-missing") {
		t.Errorf("html = %q, want missing span with nil resolver", html)
	}
}

func TestRender_OrdinaryMarkdownLinksAndImagesStillWork(t *testing.T) {
	html, err := Render("[a link](https://example.com) and ![alt](pic.png)", nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, `<a href="https://example.com">a link</a>`) {
		t.Errorf("html = %q, standard link broken", html)
	}
	if !strings.Contains(html, `<img src="pic.png" alt="alt">`) {
		t.Errorf("html = %q, standard image broken", html)
	}
}

func TestRender_GFMTableRenders(t *testing.T) {
	html, err := Render("| a | b |\n|---|---|\n| 1 | 2 |\n", nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, "<table>") {
		t.Errorf("html = %q, want a rendered GFM table", html)
	}
}
