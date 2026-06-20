import { Link, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import NotesListPage from './pages/NotesListPage'
import NotePage from './pages/NotePage'
import LoginPage from './pages/LoginPage'
import { logout } from './api'

export default function App() {
  const location = useLocation()

  // The login page has no session yet, so it can't share the authenticated
  // shell (nav, logout button) with the rest of the app.
  if (location.pathname === '/login') {
    return (
      <Routes>
        <Route path="/login" element={<LoginPage />} />
      </Routes>
    )
  }

  return <AuthenticatedShell />
}

function AuthenticatedShell() {
  const navigate = useNavigate()

  async function onLogout() {
    await logout()
    navigate('/login')
  }

  return (
    <div className="app-shell">
      <nav className="app-nav">
        <Link to="/notes">Notes</Link>
        <button type="button" className="logout-button" onClick={onLogout}>
          Log out
        </button>
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
