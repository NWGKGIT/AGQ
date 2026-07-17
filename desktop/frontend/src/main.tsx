import React from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import App from './App'
import { ThemeProvider } from '@/components/theme-provider'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // The data source is a local daemon: failures mean it is down, not
      // flaky, so surface them immediately instead of retrying.
      retry: false,
      refetchOnWindowFocus: true,
    },
  },
})

const container = document.getElementById('root')

createRoot(container!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="dark">
        <App />
      </ThemeProvider>
    </QueryClientProvider>
  </React.StrictMode>,
)
