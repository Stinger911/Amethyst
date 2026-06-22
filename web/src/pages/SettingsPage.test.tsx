import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getSettings, saveSettings } from '../api'
import SettingsPage from './SettingsPage'

vi.mock('../api', () => ({
  getSettings: vi.fn(),
  saveSettings: vi.fn(),
}))

const mockGetSettings = vi.mocked(getSettings)
const mockSaveSettings = vi.mocked(saveSettings)

describe('SettingsPage', () => {
  afterEach(() => {
    mockGetSettings.mockReset()
    mockSaveSettings.mockReset()
  })

  it('loads the current capture mode and checks the matching radio', async () => {
    mockGetSettings.mockResolvedValue({ captureMode: 'daily' })

    render(<SettingsPage />)

    expect(await screen.findByLabelText(/today's daily note/i)).toBeChecked()
    expect(screen.getByLabelText(/inbox folder/i)).not.toBeChecked()
  })

  it('shows an error when loading fails', async () => {
    mockGetSettings.mockRejectedValue(new Error('boom'))

    render(<SettingsPage />)

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('saves the new mode when a different radio is picked', async () => {
    mockGetSettings.mockResolvedValue({ captureMode: 'inbox' })
    mockSaveSettings.mockResolvedValue({ captureMode: 'daily' })

    render(<SettingsPage />)
    await screen.findByLabelText(/inbox folder/i)

    await userEvent.click(screen.getByLabelText(/today's daily note/i))

    expect(mockSaveSettings).toHaveBeenCalledWith({ captureMode: 'daily' })
    expect(await screen.findByText('Saved')).toBeInTheDocument()
    expect(screen.getByLabelText(/today's daily note/i)).toBeChecked()
  })

  it('shows an error when saving fails', async () => {
    mockGetSettings.mockResolvedValue({ captureMode: 'inbox' })
    mockSaveSettings.mockRejectedValue(new Error('save boom'))

    render(<SettingsPage />)
    await screen.findByLabelText(/inbox folder/i)

    await userEvent.click(screen.getByLabelText(/today's daily note/i))

    expect(await screen.findByRole('alert')).toHaveTextContent('save boom')
  })
})
