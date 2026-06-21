import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { listNotes } from '../api'
import NotesListPage from './NotesListPage'

vi.mock('../api', () => ({
  listNotes: vi.fn(),
}))

const mockListNotes = vi.mocked(listNotes)

function renderPage() {
  return render(
    <MemoryRouter>
      <NotesListPage />
    </MemoryRouter>,
  )
}

describe('NotesListPage', () => {
  afterEach(() => {
    mockListNotes.mockReset()
  })

  it('renders a flat note as a top-level link with its tags', async () => {
    mockListNotes.mockResolvedValue({
      notes: [{ path: 'Hub.md', title: 'Hub', tags: ['demo', 'intro'] }],
    })

    renderPage()

    expect(await screen.findByRole('link', { name: 'Hub' })).toHaveAttribute(
      'href',
      '/note/Hub.md',
    )
    expect(screen.getByText('demo')).toBeInTheDocument()
    expect(screen.getByText('intro')).toBeInTheDocument()
  })

  it('groups notes under a folder button and toggles visibility on click', async () => {
    mockListNotes.mockResolvedValue({
      notes: [{ path: 'Folder/Leaf.md', title: 'Leaf', tags: [] }],
    })

    renderPage()

    const folderButton = await screen.findByRole('button', { name: /Folder/ })
    expect(folderButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: 'Leaf' })).toBeInTheDocument()

    await userEvent.click(folderButton)

    expect(folderButton).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('link', { name: 'Leaf' })).not.toBeInTheDocument()

    await userEvent.click(folderButton)

    expect(screen.getByRole('link', { name: 'Leaf' })).toBeInTheDocument()
  })

  it('shows an error message when the request fails', async () => {
    mockListNotes.mockRejectedValue(new Error('network down'))

    renderPage()

    expect(await screen.findByRole('alert')).toHaveTextContent('network down')
  })
})
