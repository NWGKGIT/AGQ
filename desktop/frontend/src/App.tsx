import { lazy, Suspense, useState } from 'react'

import { Sidebar } from '@/components/layout/sidebar'
import type { Page } from '@/types/navigation'

const OverviewPage = lazy(() => import('@/pages/overview').then((module) => ({ default: module.OverviewPage })))
const AnalyticsPage = lazy(() => import('@/pages/analytics').then((module) => ({ default: module.AnalyticsPage })))
const SettingsPage = lazy(() => import('@/pages/settings').then((module) => ({ default: module.SettingsPage })))

const pages: Record<Page, React.LazyExoticComponent<() => React.JSX.Element>> = {
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
		<Suspense
			fallback={
				<div className="flex h-full items-center justify-center text-sm text-muted-foreground">
					Loading…
				</div>
			}
		>
			<ActivePage />
		</Suspense>
      </main>
    </div>
  )
}
