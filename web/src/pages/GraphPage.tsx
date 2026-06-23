import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { forceCenter, forceLink, forceManyBody, forceSimulation } from 'd3-force'
import { getGraph, type GraphEdge, type GraphNode } from '../api'

const WIDTH = 800
const HEIGHT = 600
const SIMULATION_TICKS = 300

interface SimNode extends GraphNode {
  x?: number
  y?: number
}

type Position = { x: number; y: number }

// layout runs the force simulation synchronously to a fixed number of
// ticks instead of animating frame-by-frame — for a single-user vault's
// note graph (tens to low hundreds of nodes) that settles fast enough to
// not need requestAnimationFrame, and it keeps the result deterministic
// and easy to test.
interface SimLink {
  source: string
  target: string
}

function layout(nodes: GraphNode[], edges: GraphEdge[]): Map<string, Position> {
  const simNodes: SimNode[] = nodes.map((n) => ({ ...n }))
  const simLinks: SimLink[] = edges.map((e) => ({ source: e.source, target: e.target }))

  const simulation = forceSimulation<SimNode>(simNodes)
    .force(
      'link',
      forceLink<SimNode, SimLink>(simLinks).id((d) => d.path).distance(80),
    )
    .force('charge', forceManyBody().strength(-150))
    .force('center', forceCenter(WIDTH / 2, HEIGHT / 2))
    .stop()

  for (let i = 0; i < SIMULATION_TICKS; i++) {
    simulation.tick()
  }

  return new Map(simNodes.map((n) => [n.path, { x: n.x ?? 0, y: n.y ?? 0 }]))
}

export default function GraphPage() {
  const navigate = useNavigate()
  const [nodes, setNodes] = useState<GraphNode[] | null>(null)
  const [edges, setEdges] = useState<GraphEdge[]>([])
  const [error, setError] = useState<string | null>(null)
  const [positions, setPositions] = useState<Map<string, Position>>(new Map())
  const [dragging, setDragging] = useState<string | null>(null)
  const didDragRef = useRef(false)

  useEffect(() => {
    getGraph()
      .then((res) => {
        setNodes(res.nodes)
        setEdges(res.edges)
        setPositions(layout(res.nodes, res.edges))
      })
      .catch((err) => setError(String(err)))
  }, [])

  function onNodePointerDown(path: string) {
    setDragging(path)
    didDragRef.current = false
  }

  function onSvgPointerMove(e: React.PointerEvent<SVGSVGElement>) {
    if (!dragging) return
    didDragRef.current = true
    const rect = e.currentTarget.getBoundingClientRect()
    setPositions((prev) => {
      const next = new Map(prev)
      next.set(dragging, { x: e.clientX - rect.left, y: e.clientY - rect.top })
      return next
    })
  }

  function onNodeClick(path: string) {
    if (didDragRef.current) return
    navigate(`/note/${path}`)
  }

  if (error) return <p role="alert">Failed to load graph: {error}</p>
  if (!nodes) return <p>Loading…</p>

  return (
    <div className="graph-page">
      <h1>Graph</h1>
      <svg
        width={WIDTH}
        height={HEIGHT}
        className="graph-svg"
        onPointerMove={onSvgPointerMove}
        onPointerUp={() => setDragging(null)}
        onPointerLeave={() => setDragging(null)}
      >
        {edges.map((edge) => {
          const from = positions.get(edge.source)
          const to = positions.get(edge.target)
          if (!from || !to) return null
          return (
            <line
              key={`${edge.source}->${edge.target}`}
              className="graph-edge"
              x1={from.x}
              y1={from.y}
              x2={to.x}
              y2={to.y}
              stroke="#999"
            />
          )
        })}
        {nodes.map((node) => {
          const pos = positions.get(node.path)
          if (!pos) return null
          return (
            <g
              key={node.path}
              role="link"
              aria-label={node.title}
              className="graph-node"
              transform={`translate(${pos.x}, ${pos.y})`}
              onPointerDown={() => onNodePointerDown(node.path)}
              onClick={() => onNodeClick(node.path)}
              style={{ cursor: 'pointer' }}
            >
              <circle r={8} fill="#4970e6" />
              <text x={12} y={4} fontSize={12}>
                {node.title}
              </text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}
