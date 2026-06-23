import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { getNote, type NoteDetail } from '../api'
import { useExternalChangeNotice } from '../useExternalChangeNotice'

export default function NotePage() {
  const params = useParams()
  const path = params['*'] ?? ''
  // Remounting on path change (via key) gives each note a clean state
  // instead of manually resetting note/error inside an effect.
  return <NoteView key={path} path={path} />
}

function NoteView({ path }: { path: string }) {
  const navigate = useNavigate()
  const [note, setNote] = useState<NoteDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [changed, resetChanged] = useExternalChangeNotice(path)

  const load = useCallback(() => {
    getNote(path)
      .then(setNote)
      .catch((err) => setError(String(err)))
  }, [path])

  useEffect(() => {
    load()
  }, [load])

  function onReload() {
    resetChanged()
    load()
  }

  // Rendered wiki-links are plain <a href="/note/..."> from the Go-side
  // renderer (internal/render/wikilink.go) — intercept clicks on those so
  // navigation stays client-side instead of a full page reload.
  function onContentClick(e: React.MouseEvent<HTMLDivElement>) {
    const anchor = (e.target as HTMLElement).closest('a')
    if (!anchor) return
    const href = anchor.getAttribute('href')
    if (!href || !href.startsWith('/note/')) return
    e.preventDefault()
    navigate(href)
  }

  if (error) return <p role="alert">Failed to load note: {error}</p>
  if (!note) return <p>Loading…</p>

  return (
    <article>
      <h1>{note.title}</h1>
      <Link to={`/edit/${note.path}`} className="edit-link">
        Edit
      </Link>

      {changed && (
        <div role="alert" className="external-change-banner">
          <p>This note changed outside the app.</p>
          <button type="button" onClick={onReload}>
            Reload
          </button>
        </div>
      )}

      <div
        className="note-content"
        onClick={onContentClick}
        dangerouslySetInnerHTML={{ __html: note.html }}
      />

      {note.backlinks.length > 0 && (
        <div className="backlinks">
          <h2>Backlinks</h2>
          <ul>
            {note.backlinks.map((b) => (
              <li key={b.path}>
                <Link to={`/note/${b.path}`}>{b.title}</Link>
              </li>
            ))}
          </ul>
        </div>
      )}
    </article>
  )
}
