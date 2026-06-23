import { useCallback, useEffect, useState } from 'react'

// useExternalChangeNotice subscribes to GET /api/ws (internal/notify) and
// reports whether path has changed on disk outside the app since this
// hook mounted — e.g. edited directly in desktop Obsidian — per
// plan_amethyst-web-ui §5. The connection is per-mount (opened while a
// note is being viewed, closed when navigating away) rather than a single
// app-wide socket, which is simplest for a single-page-at-a-time SPA.
//
// Callers that show a different path over the hook's lifetime (NotePage
// doesn't — it remounts via a `key={path}`) should rely on that same
// remount-via-key pattern instead of expecting this hook to reset itself,
// per https://react.dev/learn/you-might-not-need-an-effect.
export function useExternalChangeNotice(path: string): [boolean, () => void] {
  const [changed, setChanged] = useState(false)

  useEffect(() => {
    if (typeof WebSocket === 'undefined') return

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${protocol}://${window.location.host}/api/ws`)
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as { path?: string }
        if (data.path === path) setChanged(true)
      } catch {
        // Malformed message — ignore rather than crash the page over a
        // best-effort notification.
      }
    }

    return () => ws.close()
  }, [path])

  const reset = useCallback(() => setChanged(false), [])
  return [changed, reset]
}
