import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { search } from '../api'
import SearchPage from './SearchPage'

vi.mock('../api', () => ({
  search: vi.fn(),
}))

const mockSearch = vi.mocked(search)

function renderSearchPage() {
  return render(
    <MemoryRouter>
      <SearchPage />
    </MemoryRouter>,
  )
}

describe('SearchPage', () => {
  afterEach(() => {
    mockSearch.mockReset()
  })

  it('shows results with a link and highlighted snippet after submitting', async () => {
    mockSearch.mockResolvedValue({
      query: 'banana',
      results: [{ path: 'Banana.md', title: 'Banana', snippet: 'Banana <mark>bread</mark> is great' }],
    })

    renderSearchPage()
    await userEvent.type(screen.getByPlaceholderText('Search notes…'), 'banana')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(mockSearch).toHaveBeenCalledWith('banana')
    expect(await screen.findByRole('link', { name: 'Banana' })).toHaveAttribute(
      'href',
      '/note/Banana.md',
    )
    expect(screen.getByText('bread')).toBeInTheDocument()
    expect(document.querySelector('mark')).toHaveTextContent('bread')
  })

  it('shows "No results." for an empty result set', async () => {
    mockSearch.mockResolvedValue({ query: 'nonexistent', results: [] })

    renderSearchPage()
    await userEvent.type(screen.getByPlaceholderText('Search notes…'), 'nonexistent')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(await screen.findByText('No results.')).toBeInTheDocument()
  })

  it('shows an error message when the search fails', async () => {
    mockSearch.mockRejectedValue(new Error('boom'))

    renderSearchPage()
    await userEvent.type(screen.getByPlaceholderText('Search notes…'), 'banana')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('does not call search for a blank query', async () => {
    renderSearchPage()
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(mockSearch).not.toHaveBeenCalled()
    expect(await screen.findByText('No results.')).toBeInTheDocument()
  })
})
