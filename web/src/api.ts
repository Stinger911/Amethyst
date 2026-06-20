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

export function getNote(path: string): Promise<NoteDetail> {
  return getJSON(`/api/notes/${path.split('/').map(encodeURIComponent).join('/')}`)
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
