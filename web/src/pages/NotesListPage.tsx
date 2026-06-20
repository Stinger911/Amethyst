import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listNotes, type NoteSummary } from '../api'

export default function NotesListPage() {
  const [notes, setNotes] = useState<NoteSummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listNotes()
      .then((res) => setNotes(res.notes))
      .catch((err) => setError(String(err)))
  }, [])

  if (error) return <p role="alert">Failed to load notes: {error}</p>
  if (!notes) return <p>Loading…</p>

  return (
    <ul className="notes-list">
      {notes.map((note) => (
        <li key={note.path}>
          <Link to={`/note/${note.path}`}>{note.title}</Link>
          {note.tags.length > 0 && (
            <span className="note-tags">
              {note.tags.map((tag) => (
                <span className="note-tag" key={tag}>
                  {tag}
                </span>
              ))}
            </span>
          )}
        </li>
      ))}
    </ul>
  )
}
