import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getNote, type NoteDetail } from '../api'
import NotePage from './NotePage'

vi.mock('../api', () => ({
  getNote: vi.fn(),
}))

const mockGetNote = vi.mocked(getNote)

class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  onmessage: ((event: { data: string }) => void) | null = null
  url: string

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  close() {}

  emit(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) })
  }
}

const hub: NoteDetail = {
  path: 'Hub.md',
  title: 'Hub',
  html: '<p>See <a href="/note/Leaf.md">Leaf</a></p>',
  frontmatter: {},
  tags: ['demo'],
  backlinks: [{ path: 'Other.md', title: 'Other' }],
  raw: 'See [[Leaf]]',
  hash: 'hub-hash',
}

const leaf: NoteDetail = {
  path: 'Leaf.md',
  title: 'Leaf',
  html: '<p>leaf content</p>',
  frontmatter: {},
  tags: [],
  backlinks: [],
  raw: 'leaf content',
  hash: 'leaf-hash',
}

function renderNotePage(initialPath: string) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/note/*" element={<NotePage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('NotePage', () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
  })

  afterEach(() => {
    mockGetNote.mockReset()
    vi.unstubAllGlobals()
  })

  it('renders the note title, rendered HTML and backlinks', async () => {
    mockGetNote.mockResolvedValue(hub)

    renderNotePage('/note/Hub.md')

    expect(await screen.findByRole('heading', { name: 'Hub' })).toBeInTheDocument()
    expect(screen.getByText('See')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Other' })).toHaveAttribute(
      'href',
      '/note/Other.md',
    )
    expect(mockGetNote).toHaveBeenCalledWith('Hub.md')
  })

  it('shows an error message when loading fails', async () => {
    mockGetNote.mockRejectedValue(new Error('boom'))

    renderNotePage('/note/Hub.md')

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('navigates client-side when a rendered wiki-link is clicked', async () => {
    mockGetNote.mockImplementation((path) =>
      Promise.resolve(path === 'Leaf.md' ? leaf : hub),
    )

    renderNotePage('/note/Hub.md')

    await screen.findByRole('heading', { name: 'Hub' })
    await userEvent.click(screen.getByRole('link', { name: 'Leaf' }))

    expect(await screen.findByRole('heading', { name: 'Leaf' })).toBeInTheDocument()
    expect(mockGetNote).toHaveBeenCalledWith('Leaf.md')
  })

  it('shows a banner when the viewed note changes externally', async () => {
    mockGetNote.mockResolvedValue(hub)

    renderNotePage('/note/Hub.md')
    await screen.findByRole('heading', { name: 'Hub' })

    FakeWebSocket.instances[0].emit({ path: 'Hub.md' })

    expect(await screen.findByText('This note changed outside the app.')).toBeInTheDocument()
  })

  it('ignores a change notice for a different note', async () => {
    mockGetNote.mockResolvedValue(hub)

    renderNotePage('/note/Hub.md')
    await screen.findByRole('heading', { name: 'Hub' })

    FakeWebSocket.instances[0].emit({ path: 'SomeOtherNote.md' })

    expect(screen.queryByText('This note changed outside the app.')).not.toBeInTheDocument()
  })

  it('reload clears the banner and refetches the note', async () => {
    mockGetNote.mockResolvedValue(hub)

    renderNotePage('/note/Hub.md')
    await screen.findByRole('heading', { name: 'Hub' })
    FakeWebSocket.instances[0].emit({ path: 'Hub.md' })
    await screen.findByText('This note changed outside the app.')

    mockGetNote.mockClear()
    const updatedHub: NoteDetail = { ...hub, title: 'Hub (edited)' }
    mockGetNote.mockResolvedValue(updatedHub)

    await userEvent.click(screen.getByRole('button', { name: 'Reload' }))

    expect(mockGetNote).toHaveBeenCalledWith('Hub.md')
    expect(await screen.findByRole('heading', { name: 'Hub (edited)' })).toBeInTheDocument()
    expect(screen.queryByText('This note changed outside the app.')).not.toBeInTheDocument()
  })
})
