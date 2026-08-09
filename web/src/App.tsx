import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { ContainerDetailPage } from './pages/ContainerDetail'
import { ContainersPage } from './pages/Containers'
import { DashboardPage } from './pages/Dashboard'
import { GraphPage } from './pages/Graph'
import { ImagesPage } from './pages/Images'
import { NetworksPage } from './pages/Networks'
import { SettingsPage } from './pages/Settings'
import { StacksPage } from './pages/Stacks'
import { VolumesPage } from './pages/Volumes'

export default function App() {
  return (
    <BrowserRouter>
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
          <Route path="settings" element={<SettingsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
