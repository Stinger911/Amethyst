import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getAuthConfig, login } from '../api'
import LoginPage from './LoginPage'

vi.mock('../api', () => ({
  getAuthConfig: vi.fn(),
  login: vi.fn(),
}))

const mockGetAuthConfig = vi.mocked(getAuthConfig)
const mockLogin = vi.mocked(login)

function renderLoginPage(initialPath = '/login') {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<div>Notes home</div>} />
        <Route path="/note/Hub.md" element={<div>Hub note</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('LoginPage', () => {
  afterEach(() => {
    mockGetAuthConfig.mockReset()
    mockLogin.mockReset()
  })

  it('does not render the Telegram widget when telegram login is unconfigured', async () => {
    mockGetAuthConfig.mockResolvedValue({ telegramBotUsername: '' })

    const { container } = renderLoginPage()

    await screen.findByLabelText('Password')
    expect(container.querySelector('.login-telegram')).not.toBeInTheDocument()
  })

  it('injects the Telegram widget script when a bot username is configured', async () => {
    mockGetAuthConfig.mockResolvedValue({ telegramBotUsername: 'AmethystBot' })

    const { container } = renderLoginPage()

    const script = await waitFor(() => {
      const el = container.querySelector('.login-telegram script')
      if (!el) throw new Error('script not injected yet')
      return el
    })
    expect(script).toHaveAttribute('data-telegram-login', 'AmethystBot')
    expect(script).toHaveAttribute('src', 'https://telegram.org/js/telegram-widget.js?22')
  })

  it('shows a Telegram failure message from the ?error=telegram query param', async () => {
    mockGetAuthConfig.mockResolvedValue({ telegramBotUsername: '' })

    renderLoginPage('/login?error=telegram')

    expect(await screen.findByRole('alert')).toHaveTextContent('Telegram login failed')
  })

  it('shows the server error on a failed password submit', async () => {
    mockGetAuthConfig.mockResolvedValue({ telegramBotUsername: '' })
    mockLogin.mockRejectedValue(new Error('invalid password'))

    renderLoginPage()

    await userEvent.type(await screen.findByLabelText('Password'), 'wrong')
    await userEvent.click(screen.getByRole('button', { name: 'Log in' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid password')
    expect(mockLogin).toHaveBeenCalledWith('wrong')
  })

  it('navigates to / after a successful login', async () => {
    mockGetAuthConfig.mockResolvedValue({ telegramBotUsername: '' })
    mockLogin.mockResolvedValue(undefined)

    renderLoginPage()

    await userEvent.type(await screen.findByLabelText('Password'), 's3cret')
    await userEvent.click(screen.getByRole('button', { name: 'Log in' }))

    expect(await screen.findByText('Notes home')).toBeInTheDocument()
  })

  it('navigates to the ?next= path after a successful login', async () => {
    mockGetAuthConfig.mockResolvedValue({ telegramBotUsername: '' })
    mockLogin.mockResolvedValue(undefined)

    renderLoginPage('/login?next=%2Fnote%2FHub.md')

    await userEvent.type(await screen.findByLabelText('Password'), 's3cret')
    await userEvent.click(screen.getByRole('button', { name: 'Log in' }))

    expect(await screen.findByText('Hub note')).toBeInTheDocument()
  })
})
