import { lazy, Suspense } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { useT } from './i18n'

const DashboardPage = lazy(() =>
  import('./pages/Dashboard').then((m) => ({ default: m.DashboardPage })),
)
const ContainersPage = lazy(() =>
  import('./pages/Containers').then((m) => ({ default: m.ContainersPage })),
)
const ContainerDetailPage = lazy(() =>
  import('./pages/ContainerDetail').then((m) => ({ default: m.ContainerDetailPage })),
)
const StacksPage = lazy(() => import('./pages/Stacks').then((m) => ({ default: m.StacksPage })))
const NetworksPage = lazy(() =>
  import('./pages/Networks').then((m) => ({ default: m.NetworksPage })),
)
const VolumesPage = lazy(() =>
  import('./pages/Volumes').then((m) => ({ default: m.VolumesPage })),
)
const ImagesPage = lazy(() => import('./pages/Images').then((m) => ({ default: m.ImagesPage })))
const GraphPage = lazy(() => import('./pages/Graph').then((m) => ({ default: m.GraphPage })))
const DiagnosticsPage = lazy(() =>
  import('./pages/Diagnostics').then((m) => ({ default: m.DiagnosticsPage })),
)
const SnapshotsPage = lazy(() =>
  import('./pages/Snapshots').then((m) => ({ default: m.SnapshotsPage })),
)
const SettingsPage = lazy(() =>
  import('./pages/Settings').then((m) => ({ default: m.SettingsPage })),
)

function RouteFallback() {
  const t = useT()
  return <div className="page muted">{t('common.loading')}</div>
}

export default function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={<RouteFallback />}>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<DashboardPage />} />
            <Route path="containers" element={<ContainersPage />} />
            <Route path="containers/:id" element={<ContainerDetailPage />} />
            <Route path="stacks" element={<StacksPage />} />
            <Route path="networks" element={<NetworksPage />} />
            <Route path="volumes" element={<VolumesPage />} />
            <Route path="images" element={<ImagesPage />} />
            <Route path="graph" element={<GraphPage />} />
            <Route path="diagnostics" element={<DiagnosticsPage />} />
            <Route path="snapshots" element={<SnapshotsPage />} />
            <Route path="settings" element={<SettingsPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </Suspense>
    </BrowserRouter>
  )
}
