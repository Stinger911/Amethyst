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
  raw: string
  hash: string
}

function redirectToLogin() {
  const next = encodeURIComponent(window.location.pathname + window.location.search)
  window.location.href = `/login?next=${next}`
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (res.status === 401) {
    redirectToLogin()
    throw new Error('authentication required')
  }
  if (!res.ok) {
    throw new Error(`${url}: ${res.status} ${await res.text()}`)
  }
  return res.json() as Promise<T>
}

export function listNotes(): Promise<{ notes: NoteSummary[] }> {
  return getJSON('/api/notes')
}

function notePathURL(path: string): string {
  return `/api/notes/${path.split('/').map(encodeURIComponent).join('/')}`
}

export function getNote(path: string): Promise<NoteDetail> {
  return getJSON(notePathURL(path))
}

// ConflictError signals a 409 from PUT /api/notes/:path: the server
// preserved the caller's content at conflictPath instead of overwriting a
// version that changed on disk since baseHash was loaded (see
// internal/api/notes_write.go and plan_amethyst-mvp Фаза 3).
export class ConflictError extends Error {
  conflictPath: string
  constructor(conflictPath: string) {
    super(`conflict: saved as ${conflictPath} instead`)
    this.conflictPath = conflictPath
  }
}

export interface SaveNoteResult {
  hash: string
}

export async function saveNote(
  path: string,
  content: string,
  baseHash: string,
): Promise<SaveNoteResult> {
  const res = await fetch(notePathURL(path), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content, baseHash }),
  })
  if (res.status === 401) {
    redirectToLogin()
    throw new Error('authentication required')
  }
  if (res.status === 409) {
    const body = (await res.json()) as { conflictPath: string }
    throw new ConflictError(body.conflictPath)
  }
  if (!res.ok) {
    throw new Error(`save note: ${res.status} ${await res.text()}`)
  }
  return res.json() as Promise<SaveNoteResult>
}

export interface AuthConfig {
  telegramBotUsername: string
}

export function getAuthConfig(): Promise<AuthConfig> {
  return getJSON('/api/auth/config')
}

export async function login(password: string): Promise<void> {
  const res = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!res.ok) {
    throw new Error((await res.text()) || 'login failed')
  }
}

export async function logout(): Promise<void> {
  await fetch('/api/auth/logout', { method: 'POST' })
}
