package index

import (
	"path"
	"strings"
)

// targetResolver matches a wiki-link's raw target text against the full
// set of paths in the vault. Resolution needs that complete set up front,
// which is why it can't happen inside vault.ParseNote (which only ever
// sees one file at a time).
type targetResolver struct {
	byFullPath map[string]string
	byStem     map[string][]string
}

func newTargetResolver(paths []string) *targetResolver {
	r := &targetResolver{
		byFullPath: make(map[string]string, len(paths)),
		byStem:     make(map[string][]string, len(paths)),
	}
	for _, p := range paths {
		r.byFullPath[p] = p
		stem := strings.ToLower(strings.TrimSuffix(path.Base(p), path.Ext(p)))
		r.byStem[stem] = append(r.byStem[stem], p)
	}
	return r
}

// resolve returns the matching vault path for a raw link target, or ""
// if it can't be resolved unambiguously. Alias (`|`) and heading/block
// suffixes (`#`, `^`) are stripped before matching, since they qualify a
// location within the target, not the target itself.
func (r *targetResolver) resolve(raw string) string {
	target := raw
	if idx := strings.Index(target, "|"); idx != -1 {
		target = target[:idx]
	}
	if idx := strings.IndexAny(target, "#^"); idx != -1 {
		target = target[:idx]
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}

	if p, ok := r.byFullPath[target]; ok {
		return p
	}
	if p, ok := r.byFullPath[target+".md"]; ok {
		return p
	}

	stem := strings.ToLower(strings.TrimSuffix(path.Base(target), path.Ext(target)))
	if matches := r.byStem[stem]; len(matches) == 1 {
		return matches[0]
	}
	return ""
}
