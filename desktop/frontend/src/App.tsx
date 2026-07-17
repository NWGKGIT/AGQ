import { useState } from 'react'

import { Sidebar } from '@/components/layout/sidebar'
import { AnalyticsPage } from '@/pages/analytics'
import { OverviewPage } from '@/pages/overview'
import { SettingsPage } from '@/pages/settings'
import type { Page } from '@/types/navigation'

const pages: Record<Page, () => React.JSX.Element> = {
  overview: OverviewPage,
  analytics: AnalyticsPage,
  settings: SettingsPage,
}

export default function App() {
  const [page, setPage] = useState<Page>('overview')
  const ActivePage = pages[page]

  return (
    <div className="flex h-full">
      <Sidebar page={page} onNavigate={setPage} />
      <main className="flex-1 overflow-y-auto">
        <ActivePage />
      </main>
    </div>
  )
}
