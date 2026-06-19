package vault

import (
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	linkPattern      = regexp.MustCompile(`(!?)\[\[([^\]]+)\]\]`)
	inlineTagPattern = regexp.MustCompile(`(?:^|\s)#([A-Za-z][\w/-]*)`)
)

// ParseNote parses the raw content of a .md file into a Note. path is the
// vault-relative path and is used only to derive a fallback title.
func ParseNote(path string, content []byte) (*Note, error) {
	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	note := &Note{
		Path:        path,
		Frontmatter: frontmatter,
		Body:        body,
		Links:       extractLinks(body),
		Tags:        mergeTags(frontmatterTags(frontmatter), extractInlineTags(body)),
	}
	note.Title = noteTitle(frontmatter, path)
	return note, nil
}

func splitFrontmatter(content []byte) (map[string]any, string, error) {
	text := string(content)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return map[string]any{}, text, nil
	}

	rest := text[strings.Index(text, "\n")+1:]
	end := findFrontmatterEnd(rest)
	if end == -1 {
		// Unterminated "---" block: treat the whole file as body, no frontmatter.
		return map[string]any{}, text, nil
	}

	raw := rest[:end]
	body := strings.TrimPrefix(rest[end:], "---")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")

	frontmatter := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		if err := yaml.Unmarshal([]byte(raw), &frontmatter); err != nil {
			return nil, "", err
		}
	}
	return frontmatter, body, nil
}

// findFrontmatterEnd returns the byte offset of the closing "---" line
// within s, or -1 if none is found.
func findFrontmatterEnd(s string) int {
	offset := 0
	for {
		idx := strings.Index(s[offset:], "\n---")
		if idx == -1 {
			return -1
		}
		lineStart := offset + idx + 1
		lineEnd := lineStart + 3
		// Must be exactly "---" on its own line (allowing trailing \r).
		rest := s[lineEnd:]
		if strings.HasPrefix(rest, "\n") || strings.HasPrefix(rest, "\r\n") || rest == "" {
			return lineStart
		}
		offset = lineEnd
	}
}

func extractLinks(body string) []Link {
	matches := linkPattern.FindAllStringSubmatch(body, -1)
	links := make([]Link, 0, len(matches))
	for _, m := range matches {
		bang, inner := m[1], m[2]
		kind := LinkWiki
		switch {
		case bang == "!":
			kind = LinkEmbed
		case strings.Contains(strings.SplitN(inner, "|", 2)[0], "^"):
			kind = LinkBlockRef
		}
		links = append(links, Link{TargetRaw: inner, Kind: kind})
	}
	return links
}

func frontmatterTags(frontmatter map[string]any) []string {
	raw, ok := frontmatter["tags"]
	if !ok {
		return nil
	}

	var tags []string
	switch v := raw.(type) {
	case string:
		for _, t := range strings.Fields(strings.ReplaceAll(v, ",", " ")) {
			tags = append(tags, strings.TrimPrefix(t, "#"))
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				tags = append(tags, strings.TrimPrefix(s, "#"))
			}
		}
	}
	return tags
}

func extractInlineTags(body string) []string {
	matches := inlineTagPattern.FindAllStringSubmatch(body, -1)
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tags = append(tags, m[1])
	}
	return tags
}

func mergeTags(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	merged := make([]string, 0, len(a)+len(b))
	for _, t := range append(a, b...) {
		if !seen[t] {
			seen[t] = true
			merged = append(merged, t)
		}
	}
	return merged
}

func noteTitle(frontmatter map[string]any, path string) string {
	if t, ok := frontmatter["title"].(string); ok && t != "" {
		return t
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ParseNoteFrom is a convenience wrapper around ParseNote for callers that
// have an io.Reader (e.g. an open *os.File) rather than a byte slice.
func ParseNoteFrom(path string, r io.Reader) (*Note, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseNote(path, content)
}
