import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getNote, type NoteDetail } from '../api'
import NotePage from './NotePage'

vi.mock('../api', () => ({
  getNote: vi.fn(),
}))

const mockGetNote = vi.mocked(getNote)

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
  afterEach(() => {
    mockGetNote.mockReset()
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
})
