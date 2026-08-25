import React from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import App from './App'
import { ThemeProvider } from '@/components/theme-provider'
import './index.css'
import { subscribeToUpdates } from './lib/api'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // The monitor is local, so surface connection failures immediately.
      retry: false,
      refetchOnWindowFocus: true,
    },
  },
})

const container = document.getElementById('root')

const unsubscribe = subscribeToUpdates(() => queryClient.invalidateQueries())
window.addEventListener('beforeunload', unsubscribe)

createRoot(container!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="dark">
        <App />
      </ThemeProvider>
    </QueryClientProvider>
  </React.StrictMode>,
)
