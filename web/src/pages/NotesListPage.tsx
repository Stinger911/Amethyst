import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { listNotes, type NoteSummary } from '../api'
import { buildNotesTree, type TreeNode } from '../notesTree'

export default function NotesListPage() {
  const [notes, setNotes] = useState<NoteSummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listNotes()
      .then((res) => setNotes(res.notes))
      .catch((err) => setError(String(err)))
  }, [])

  const tree = useMemo(() => (notes ? buildNotesTree(notes) : null), [notes])

  if (error) return <p role="alert">Failed to load notes: {error}</p>
  if (!tree) return <p>Loading…</p>

  return <NotesTree nodes={tree} />
}

function NotesTree({ nodes }: { nodes: TreeNode[] }) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())

  function toggle(path: string) {
    setCollapsed((prev) => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }

  function renderNodes(nodes: TreeNode[]) {
    return (
      <ul className="notes-tree">
        {nodes.map((node) =>
          node.type === 'folder' ? (
            <li key={node.path}>
              <button
                type="button"
                className="tree-folder"
                onClick={() => toggle(node.path)}
                aria-expanded={!collapsed.has(node.path)}
              >
                <span className="tree-folder-arrow">
                  {collapsed.has(node.path) ? '▸' : '▾'}
                </span>
                {node.name}
              </button>
              {!collapsed.has(node.path) && renderNodes(node.children)}
            </li>
          ) : (
            <li key={node.path}>
              <Link to={`/note/${node.path}`}>{node.title}</Link>
              {node.tags.length > 0 && (
                <span className="note-tags">
                  {node.tags.map((tag) => (
                    <span className="note-tag" key={tag}>
                      {tag}
                    </span>
                  ))}
                </span>
              )}
            </li>
          ),
        )}
      </ul>
    )
  }

  return renderNodes(nodes)
}
