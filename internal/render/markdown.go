// Package render converts a note's raw Markdown body to HTML via goldmark.
// Per plan_amethyst-web-ui §1, this happens server-side rather than in the
// browser because resolving [[wiki-links]] to real vault paths (or "not
// found") needs the index Go already holds.
package render

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

// Resolver maps a wiki-link/embed's raw target text — the exact text
// between [[ ]], including any |alias or #heading/^block suffix, matching
// the links.target_raw column — to a resolved vault path. ok is false for
// anything the index couldn't resolve, which renders as a "missing" link.
type Resolver func(raw string) (targetPath string, ok bool)

var noopResolver Resolver = func(string) (string, bool) { return "", false }

// Render converts body to an HTML string. resolve may be nil, in which case
// every wiki-link/embed renders as unresolved.
func Render(body string, resolve Resolver) (string, error) {
	if resolve == nil {
		resolve = noopResolver
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, &wikiLinkExtension{resolve: resolve}),
		// Obsidian renders raw inline HTML in notes; this vault's content is
		// the single user's own, not third-party input, so trusting it here
		// matches Obsidian's behavior rather than introducing a regression.
		goldmark.WithRendererOptions(ghtml.WithUnsafe()),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
