import { Link, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import NotesListPage from './pages/NotesListPage'
import NotePage from './pages/NotePage'
import EditPage from './pages/EditPage'
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
          {/* react-router splats must be the final path segment, so a note
              path containing slashes can't fit "/note/:path/edit" — this
              is the literal route for plan_amethyst-web-ui §6 step 5. */}
          <Route path="/edit/*" element={<EditPage />} />
        </Routes>
      </main>
    </div>
  )
}
