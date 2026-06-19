package vault

// LinkKind classifies a wiki-style reference found in a note's body.
type LinkKind string

const (
	LinkWiki     LinkKind = "wikilink"
	LinkEmbed    LinkKind = "embed"
	LinkBlockRef LinkKind = "blockref"
)

// Link is a raw, unresolved reference extracted from a note. Resolving
// TargetRaw to an actual vault path requires the full file list and
// happens at index-build time, not here (see plan_amethyst-storage-index).
type Link struct {
	TargetRaw string
	Kind      LinkKind
}

// Note is the parsed result of a single .md file. Frontmatter and Tags
// are exposed separately even though tags can originate from either,
// matching the `tags` table design (frontmatter `tags:` + inline `#tag`).
type Note struct {
	Path        string
	Title       string
	Frontmatter map[string]any
	Tags        []string
	Body        string
	Links       []Link
}
