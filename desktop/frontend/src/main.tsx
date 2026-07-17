import React from 'react'
import { createRoot } from 'react-dom/client'

import App from './App'
import { ThemeProvider } from '@/components/theme-provider'
import './index.css'

const container = document.getElementById('root')

createRoot(container!).render(
  <React.StrictMode>
    <ThemeProvider defaultTheme="dark">
      <App />
    </ThemeProvider>
  </React.StrictMode>,
)
