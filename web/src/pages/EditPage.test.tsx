import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ConflictError, getNote, saveNote, type NoteDetail } from '../api'
import EditPage from './EditPage'

vi.mock('../api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api')>()
  return { ...actual, getNote: vi.fn(), saveNote: vi.fn() }
})

const mockGetNote = vi.mocked(getNote)
const mockSaveNote = vi.mocked(saveNote)

const note: NoteDetail = {
  path: 'Leaf.md',
  title: 'Leaf',
  html: '<p>leaf content</p>',
  frontmatter: {},
  tags: [],
  backlinks: [],
  raw: '# Leaf\n\nOriginal text.\n',
  hash: 'original-hash',
}

function renderEditPage(initialPath: string) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/edit/*" element={<EditPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('EditPage', () => {
  afterEach(() => {
    mockGetNote.mockReset()
    mockSaveNote.mockReset()
  })

  it('loads the raw markdown and renders a Save button', async () => {
    mockGetNote.mockResolvedValue(note)

    renderEditPage('/edit/Leaf.md')

    expect(await screen.findByRole('button', { name: 'Save' })).toBeInTheDocument()
    expect(mockGetNote).toHaveBeenCalledWith('Leaf.md')
    await waitFor(() => expect(screen.getByText('Original text.')).toBeInTheDocument())
  })

  it('shows an error when loading fails', async () => {
    mockGetNote.mockRejectedValue(new Error('boom'))

    renderEditPage('/edit/Leaf.md')

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('saves with the loaded hash as baseHash and shows Saved', async () => {
    mockGetNote.mockResolvedValue(note)
    mockSaveNote.mockResolvedValue({ hash: 'new-hash' })

    renderEditPage('/edit/Leaf.md')
    await screen.findByRole('button', { name: 'Save' })

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await screen.findByText('Saved')
    expect(mockSaveNote).toHaveBeenCalledWith('Leaf.md', '# Leaf\n\nOriginal text.\n', 'original-hash')
  })

  it('shows a conflict banner with a link to the conflict copy on a 409', async () => {
    mockGetNote.mockResolvedValue(note)
    mockSaveNote.mockRejectedValue(new ConflictError('Leaf.sync-conflict-x-web.md'))

    renderEditPage('/edit/Leaf.md')
    await screen.findByRole('button', { name: 'Save' })

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent('Leaf.sync-conflict-x-web.md')
    expect(screen.getByRole('link', { name: 'Leaf.sync-conflict-x-web.md' })).toHaveAttribute(
      'href',
      '/note/Leaf.sync-conflict-x-web.md',
    )
  })

  it('reloads the current server version when asked to after a conflict', async () => {
    mockGetNote.mockResolvedValue(note)
    mockSaveNote.mockRejectedValue(new ConflictError('Leaf.sync-conflict-x-web.md'))

    renderEditPage('/edit/Leaf.md')
    await screen.findByRole('button', { name: 'Save' })
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await screen.findByRole('alert')

    const reloaded: NoteDetail = { ...note, raw: '# Leaf\n\nServer text.\n', hash: 'server-hash' }
    mockGetNote.mockResolvedValue(reloaded)

    await userEvent.click(screen.getByRole('button', { name: 'Reload current version' }))

    await waitFor(() => expect(screen.getByText('Server text.')).toBeInTheDocument())
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
