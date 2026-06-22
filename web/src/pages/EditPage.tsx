import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import CodeMirror from '@uiw/react-codemirror'
import { markdown } from '@codemirror/lang-markdown'
import { ConflictError, getNote, saveNote } from '../api'

const markdownExtensions = [markdown()]

export default function EditPage() {
  const params = useParams()
  const path = params['*'] ?? ''
  // Remounting on path change (via key) gives each note a clean state,
  // matching the pattern in NotePage.
  return <EditView key={path} path={path} />
}

type SaveStatus = 'idle' | 'saving' | 'saved'

function EditView({ path }: { path: string }) {
  const [content, setContent] = useState<string | null>(null)
  const [hash, setHash] = useState('')
  const [loadError, setLoadError] = useState<string | null>(null)
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const [saveError, setSaveError] = useState<string | null>(null)
  const [conflictPath, setConflictPath] = useState<string | null>(null)

  useEffect(() => {
    getNote(path)
      .then((note) => {
        setContent(note.raw)
        setHash(note.hash)
      })
      .catch((err) => setLoadError(String(err)))
  }, [path])

  const onSave = useCallback(async () => {
    if (content === null) return
    setSaveStatus('saving')
    setSaveError(null)
    setConflictPath(null)
    try {
      const result = await saveNote(path, content, hash)
      setHash(result.hash)
      setSaveStatus('saved')
    } catch (err) {
      setSaveStatus('idle')
      if (err instanceof ConflictError) {
        setConflictPath(err.conflictPath)
      } else {
        setSaveError(String(err))
      }
    }
  }, [path, content, hash])

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault()
        onSave()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onSave])

  async function onReloadFromServer() {
    setConflictPath(null)
    setLoadError(null)
    try {
      const note = await getNote(path)
      setContent(note.raw)
      setHash(note.hash)
    } catch (err) {
      setLoadError(String(err))
    }
  }

  if (loadError) return <p role="alert">Failed to load note: {loadError}</p>
  if (content === null) return <p>Loading…</p>

  return (
    <div className="edit-page">
      <div className="edit-toolbar">
        <Link to={`/note/${path}`}>← Cancel</Link>
        <button type="button" onClick={onSave} disabled={saveStatus === 'saving'}>
          {saveStatus === 'saving' ? 'Saving…' : 'Save'}
        </button>
        {saveStatus === 'saved' && <span className="save-status">Saved</span>}
      </div>

      {conflictPath && (
        <div role="alert" className="conflict-banner">
          <p>
            This note changed on disk since you opened it. Your edits were saved separately as{' '}
            <Link to={`/note/${conflictPath}`}>{conflictPath}</Link>.
          </p>
          <button type="button" onClick={onReloadFromServer}>
            Reload current version
          </button>
        </div>
      )}

      {saveError && <p role="alert">Failed to save: {saveError}</p>}

      <CodeMirror
        value={content}
        height="70vh"
        extensions={markdownExtensions}
        onChange={(value) => {
          setContent(value)
          setSaveStatus('idle')
        }}
      />
    </div>
  )
}
