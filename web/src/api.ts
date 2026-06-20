// Mirrors the JSON shapes from internal/api/notes.go. Keep in sync with the
// Go structs by hand for now — no shared schema generation yet.

export interface NoteSummary {
  path: string
  title: string
  tags: string[]
}

export interface Backlink {
  path: string
  title: string
}

export interface NoteDetail {
  path: string
  title: string
  html: string
  frontmatter: Record<string, unknown>
  tags: string[]
  backlinks: Backlink[]
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(`${url}: ${res.status} ${await res.text()}`)
  }
  return res.json() as Promise<T>
}

export function listNotes(): Promise<{ notes: NoteSummary[] }> {
  return getJSON('/api/notes')
}

export function getNote(path: string): Promise<NoteDetail> {
  return getJSON(`/api/notes/${path.split('/').map(encodeURIComponent).join('/')}`)
}
