import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getGraph, type GraphResponse } from '../api'
import GraphPage from './GraphPage'

vi.mock('../api', () => ({
  getGraph: vi.fn(),
}))

const mockGetGraph = vi.mocked(getGraph)

const graph: GraphResponse = {
  nodes: [
    { path: 'Hub.md', title: 'Hub' },
    { path: 'Leaf.md', title: 'Leaf' },
    { path: 'Island.md', title: 'Island' },
  ],
  edges: [{ source: 'Hub.md', target: 'Leaf.md' }],
}

function renderGraphPage() {
  return render(
    <MemoryRouter initialEntries={['/graph']}>
      <Routes>
        <Route path="/graph" element={<GraphPage />} />
        <Route path="/note/*" element={<p>note view</p>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('GraphPage', () => {
  afterEach(() => {
    mockGetGraph.mockReset()
  })

  it('renders one node per graph node and one line per edge', async () => {
    mockGetGraph.mockResolvedValue(graph)

    renderGraphPage()

    expect(await screen.findByRole('link', { name: 'Hub' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Leaf' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Island' })).toBeInTheDocument()
    expect(document.querySelectorAll('.graph-node')).toHaveLength(3)
    expect(document.querySelectorAll('.graph-edge')).toHaveLength(1)
  })

  it('shows an error when loading fails', async () => {
    mockGetGraph.mockRejectedValue(new Error('boom'))

    renderGraphPage()

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('navigates to the note on a plain click', async () => {
    mockGetGraph.mockResolvedValue(graph)

    renderGraphPage()
    const hubNode = await screen.findByRole('link', { name: 'Hub' })

    fireEvent.pointerDown(hubNode)
    fireEvent.click(hubNode)

    expect(await screen.findByText('note view')).toBeInTheDocument()
  })

  it('does not navigate when the pointer moved (a drag, not a click)', async () => {
    mockGetGraph.mockResolvedValue(graph)

    renderGraphPage()
    const hubNode = await screen.findByRole('link', { name: 'Hub' })
    const svg = document.querySelector('svg.graph-svg')
    if (!svg) throw new Error('expected the graph <svg> to be in the document')

    fireEvent.pointerDown(hubNode)
    fireEvent.pointerMove(svg, { clientX: 50, clientY: 50 })
    fireEvent.pointerUp(svg)
    fireEvent.click(hubNode)

    expect(screen.queryByText('note view')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Hub' })).toBeInTheDocument()
  })
})
