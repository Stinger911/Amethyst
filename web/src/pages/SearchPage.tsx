import { useState } from 'react'
import { Link } from 'react-router-dom'
import { search, type SearchResult } from '../api'

export default function SearchPage() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [searching, setSearching] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!query.trim()) {
      setResults([])
      return
    }
    setSearching(true)
    setError(null)
    try {
      const res = await search(query)
      setResults(res.results)
    } catch (err) {
      setError(String(err))
    } finally {
      setSearching(false)
    }
  }

  return (
    <div className="search-page">
      <h1>Search</h1>

      <form onSubmit={onSubmit} className="search-form">
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search notes…"
          autoFocus
        />
        <button type="submit" disabled={searching}>
          {searching ? 'Searching…' : 'Search'}
        </button>
      </form>

      {error && <p role="alert">Search failed: {error}</p>}

      {results !== null && (
        <ul className="search-results">
          {results.length === 0 && <li className="search-no-results">No results.</li>}
          {results.map((r) => (
            <li key={r.path} className="search-result">
              <Link to={`/note/${r.path}`}>{r.title}</Link>
              <p
                className="search-snippet"
                // The backend wraps matched terms in <mark> (internal/api/search.go) —
                // the only HTML this endpoint returns, deliberately narrow.
                dangerouslySetInnerHTML={{ __html: r.snippet }}
              />
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
