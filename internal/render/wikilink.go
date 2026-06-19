package render

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var kindWikiLink = ast.NewNodeKind("WikiLink")

// wikiLinkNode is an inline AST node for [[..]] / ![[..]] syntax, resolved
// at parse time (not render time) since resolution is just a map lookup
// against rows the indexer already computed.
type wikiLinkNode struct {
	ast.BaseInline
	Display    string
	Embed      bool
	TargetPath string
	Resolved   bool
}

func (n *wikiLinkNode) Kind() ast.NodeKind { return kindWikiLink }

func (n *wikiLinkNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Display":    n.Display,
		"Embed":      fmt.Sprint(n.Embed),
		"TargetPath": n.TargetPath,
		"Resolved":   fmt.Sprint(n.Resolved),
	}, nil)
}

// wikiLinkExtension wires the inline parser and renderer into a goldmark
// instance, closing over resolve so each Render call gets its own
// independent, per-note resolution without any shared mutable state.
type wikiLinkExtension struct{ resolve Resolver }

func (e *wikiLinkExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&wikiLinkParser{resolve: e.resolve}, 0),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&wikiLinkRenderer{}, 0),
	))
}

type wikiLinkParser struct{ resolve Resolver }

func (p *wikiLinkParser) Trigger() []byte { return []byte{'[', '!'} }

// Parse recognizes [[target]], [[target|alias]] and ![[target]] on a
// single line (Obsidian itself doesn't allow wiki-links to span lines).
// Returning nil falls through to goldmark's normal link/image parsing, so
// ordinary [text](url) and ![alt](url) syntax is unaffected.
func (p *wikiLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()

	embed := false
	offset := 0
	switch {
	case bytes.HasPrefix(line, []byte("![[")):
		embed = true
		offset = 3
	case bytes.HasPrefix(line, []byte("[[")):
		offset = 2
	default:
		return nil
	}

	rest := line[offset:]
	closeIdx := bytes.Index(rest, []byte("]]"))
	if closeIdx <= 0 {
		return nil
	}

	// Raw must match links.target_raw verbatim (vault.extractLinks stores
	// the inner text unmodified) so the resolver lookup hits.
	raw := string(rest[:closeIdx])
	display := raw
	if idx := strings.IndexByte(raw, '|'); idx >= 0 {
		display = raw[idx+1:]
	}
	display = strings.TrimSpace(display)
	if display == "" {
		display = raw
	}

	block.Advance(offset + closeIdx + 2)

	node := &wikiLinkNode{Display: display, Embed: embed}
	if p.resolve != nil {
		if target, ok := p.resolve(raw); ok {
			node.TargetPath = target
			node.Resolved = true
		}
	}
	return node
}

type wikiLinkRenderer struct{}

func (r *wikiLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindWikiLink, r.render)
}

func (r *wikiLinkRenderer) render(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	node := n.(*wikiLinkNode)

	class := "wikilink"
	if node.Embed {
		class = "wikilink wikilink-embed"
	}

	if !node.Resolved {
		fmt.Fprintf(w, `<span class="%s wikilink-missing">%s</span>`, class, util.EscapeHTML([]byte(node.Display)))
		return ast.WalkContinue, nil
	}

	href := "/note/" + (&url.URL{Path: node.TargetPath}).EscapedPath()
	fmt.Fprintf(w, `<a class="%s" href="%s">%s</a>`, class, util.EscapeHTML([]byte(href)), util.EscapeHTML([]byte(node.Display)))
	return ast.WalkContinue, nil
}
