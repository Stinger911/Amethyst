import { useEffect, useState } from 'react'
import {
  getSettings,
  pairTelegram,
  saveSettings,
  type CaptureMode,
  type PairResponse,
  type Settings,
} from '../api'

export default function SettingsPage() {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    getSettings()
      .then(setSettings)
      .catch((err) => setLoadError(String(err)))
  }, [])

  async function onCaptureModeChange(captureMode: CaptureMode) {
    setSettings({ captureMode })
    setSaved(false)
    setSaveError(null)
    try {
      const result = await saveSettings({ captureMode })
      setSettings(result)
      setSaved(true)
    } catch (err) {
      setSaveError(String(err))
    }
  }

  if (loadError) return <p role="alert">Failed to load settings: {loadError}</p>
  if (!settings) return <p>Loading…</p>

  return (
    <div className="settings-page">
      <h1>Settings</h1>

      <fieldset>
        <legend>Telegram capture target</legend>
        <p>Where plain-text messages sent to the bot are saved.</p>

        <label>
          <input
            type="radio"
            name="captureMode"
            value="inbox"
            checked={settings.captureMode === 'inbox'}
            onChange={() => onCaptureModeChange('inbox')}
          />
          Inbox folder — each message becomes its own note under Inbox/
        </label>
        <label>
          <input
            type="radio"
            name="captureMode"
            value="daily"
            checked={settings.captureMode === 'daily'}
            onChange={() => onCaptureModeChange('daily')}
          />
          Today's daily note — appended to Daily/&lt;date&gt;.md
        </label>
      </fieldset>

      {saved && <p className="save-status">Saved</p>}
      {saveError && <p role="alert">Failed to save: {saveError}</p>}

      <TelegramPairing />
    </div>
  )
}

function TelegramPairing() {
  const [pairing, setPairing] = useState<PairResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [requesting, setRequesting] = useState(false)

  async function onGenerate() {
    setRequesting(true)
    setError(null)
    try {
      setPairing(await pairTelegram())
    } catch (err) {
      setError(String(err))
    } finally {
      setRequesting(false)
    }
  }

  return (
    <fieldset>
      <legend>Link Telegram</legend>
      <p>
        Generates a one-time code, valid for 10 minutes, that pairs your Telegram chat to
        this Amethyst instance for the bot and Telegram login — an alternative to setting
        TELEGRAM_OWNER_CHAT_ID by hand.
      </p>

      <button type="button" onClick={onGenerate} disabled={requesting}>
        {requesting ? 'Generating…' : 'Generate pairing code'}
      </button>

      {error && <p role="alert">Failed to generate pairing code: {error}</p>}

      {pairing && (
        <div className="telegram-pairing-result">
          <p>
            Send this to the bot: <code>/start {pairing.token}</code>
          </p>
          {pairing.botUsername && (
            <p>
              Or open it directly:{' '}
              <a
                href={`https://t.me/${pairing.botUsername}?start=${pairing.token}`}
                target="_blank"
                rel="noreferrer"
              >
                t.me/{pairing.botUsername}
              </a>
            </p>
          )}
          <p>Valid until {new Date(pairing.expiresAt).toLocaleTimeString()}.</p>
        </div>
      )}
    </fieldset>
  )
}
