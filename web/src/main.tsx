import { QueryClientProvider } from '@tanstack/react-query'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import { queryClient } from './api/queryClient'
import { I18nProvider } from './i18n'
import { applyTheme, getLocale } from './lib/prefs'
import { RealtimeProvider } from './realtime/RealtimeProvider'
import './index.css'

applyTheme()
document.documentElement.lang = getLocale()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <RealtimeProvider>
          <App />
        </RealtimeProvider>
      </I18nProvider>
    </QueryClientProvider>
  </StrictMode>,
)
