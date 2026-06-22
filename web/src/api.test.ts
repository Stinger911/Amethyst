import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  ConflictError,
  getAuthConfig,
  getNote,
  listNotes,
  login,
  logout,
  saveNote,
} from './api'

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('api', () => {
  let fetchMock: ReturnType<typeof vi.fn>
  let originalLocation: Location

  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    originalLocation = window.location
    Object.defineProperty(window, 'location', {
      value: { pathname: '/notes', search: '?x=1', href: '' },
      writable: true,
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    Object.defineProperty(window, 'location', { value: originalLocation, writable: true })
  })

  it('listNotes fetches /api/notes and returns the parsed body', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ notes: [{ path: 'Hub.md', title: 'Hub', tags: [] }] }),
    )

    const result = await listNotes()

    expect(fetchMock).toHaveBeenCalledWith('/api/notes')
    expect(result.notes).toEqual([{ path: 'Hub.md', title: 'Hub', tags: [] }])
  })

  it('getNote encodes each path segment', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({
        path: 'Folder/My Note.md',
        title: 'My Note',
        html: '<p>hi</p>',
        frontmatter: {},
        tags: [],
        backlinks: [],
      }),
    )

    await getNote('Folder/My Note.md')

    expect(fetchMock).toHaveBeenCalledWith('/api/notes/Folder/My%20Note.md')
  })

  it('getAuthConfig returns the telegram bot username', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ telegramBotUsername: 'AmethystBot' }))

    const config = await getAuthConfig()

    expect(fetchMock).toHaveBeenCalledWith('/api/auth/config')
    expect(config).toEqual({ telegramBotUsername: 'AmethystBot' })
  })

  it('redirects to /login on a 401 instead of returning data', async () => {
    fetchMock.mockResolvedValue(new Response('', { status: 401 }))

    await expect(listNotes()).rejects.toThrow('authentication required')
    expect(window.location.href).toBe('/login?next=%2Fnotes%3Fx%3D1')
  })

  it('throws with the response body on other non-OK statuses', async () => {
    fetchMock.mockResolvedValue(new Response('list notes failed', { status: 500 }))

    await expect(listNotes()).rejects.toThrow('500')
  })

  it('login posts the password as JSON and resolves on success', async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 200 }))

    await login('s3cret')

    expect(fetchMock).toHaveBeenCalledWith('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: 's3cret' }),
    })
  })

  it('login rejects with the server message on a wrong password', async () => {
    fetchMock.mockResolvedValue(new Response('invalid password', { status: 401 }))

    await expect(login('wrong')).rejects.toThrow('invalid password')
  })

  it('logout posts to /api/auth/logout', async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 200 }))

    await logout()

    expect(fetchMock).toHaveBeenCalledWith('/api/auth/logout', { method: 'POST' })
  })

  it('saveNote PUTs content and baseHash, encoding the path', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ hash: 'new-hash' }))

    const result = await saveNote('Folder/My Note.md', 'updated text', 'old-hash')

    expect(fetchMock).toHaveBeenCalledWith('/api/notes/Folder/My%20Note.md', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: 'updated text', baseHash: 'old-hash' }),
    })
    expect(result).toEqual({ hash: 'new-hash' })
  })

  it('saveNote throws a ConflictError carrying conflictPath on a 409', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'conflict', conflictPath: 'Note.sync-conflict-x-web.md' }, 409),
    )

    await expect(saveNote('Note.md', 'mine', 'stale-hash')).rejects.toMatchObject({
      conflictPath: 'Note.sync-conflict-x-web.md',
    })
  })

  it('saveNote rejects ConflictError as an instanceof check', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'conflict', conflictPath: 'Note.sync-conflict-x-web.md' }, 409),
    )

    try {
      await saveNote('Note.md', 'mine', 'stale-hash')
      expect.unreachable('saveNote should have thrown')
    } catch (err) {
      expect(err).toBeInstanceOf(ConflictError)
    }
  })

  it('saveNote redirects to /login on a 401', async () => {
    fetchMock.mockResolvedValue(new Response('', { status: 401 }))

    await expect(saveNote('Note.md', 'mine', 'hash')).rejects.toThrow('authentication required')
    expect(window.location.href).toBe('/login?next=%2Fnotes%3Fx%3D1')
  })
})
