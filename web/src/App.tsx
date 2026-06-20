import { Link, Route, Routes } from 'react-router-dom'
import NotesListPage from './pages/NotesListPage'
import NotePage from './pages/NotePage'

export default function App() {
  return (
    <div className="app-shell">
      <nav className="app-nav">
        <Link to="/notes">Notes</Link>
      </nav>
      <main className="app-main">
        <Routes>
          <Route path="/" element={<NotesListPage />} />
          <Route path="/notes" element={<NotesListPage />} />
          <Route path="/note/*" element={<NotePage />} />
        </Routes>
      </main>
    </div>
  )
}
