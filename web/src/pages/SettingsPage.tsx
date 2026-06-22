import { useEffect, useState } from 'react'
import { getSettings, saveSettings, type CaptureMode, type Settings } from '../api'

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
    </div>
  )
}
