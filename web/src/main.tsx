// Every profile uses the same Modern Minimal Dark palette, with independent display preferences.
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import './theme/themes.css'
import { UsersProvider } from './users/UsersContext'
import { DisplaySettingsProvider } from './settings/DisplaySettingsContext'
import { GuideSettingsProvider } from './settings/GuideSettingsContext'
import App from './App.tsx'
import { AuthGate } from './users/AuthGate'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <AuthGate>
      <UsersProvider>
          <DisplaySettingsProvider>
            <GuideSettingsProvider>
              <App />
            </GuideSettingsProvider>
          </DisplaySettingsProvider>
      </UsersProvider>
      </AuthGate>
    </BrowserRouter>
  </StrictMode>,
)
