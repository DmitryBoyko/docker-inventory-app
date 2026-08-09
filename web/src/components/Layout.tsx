import { useEffect, useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { StatusBanner } from './StatusBanner'

const links = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/containers', label: 'Containers' },
  { to: '/stacks', label: 'Stacks' },
  { to: '/networks', label: 'Networks' },
  { to: '/volumes', label: 'Volumes' },
  { to: '/images', label: 'Images' },
  { to: '/graph', label: 'Graph' },
  { to: '/settings', label: 'Settings' },
]

export function Layout() {
  const [navOpen, setNavOpen] = useState(false)

  useEffect(() => {
    const onResize = () => {
      if (window.innerWidth > 900) setNavOpen(false)
    }
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar-row">
          <div className="brand">
            <span className="brand-mark">DV</span>
            <div>
              <div className="brand-title">Docker Visualizer</div>
              <div className="brand-sub">read-only inventory</div>
            </div>
          </div>
          <button
            type="button"
            className="nav-toggle"
            aria-expanded={navOpen}
            aria-controls="main-nav"
            onClick={() => setNavOpen((v) => !v)}
          >
            {navOpen ? 'Close' : 'Menu'}
          </button>
        </div>
        <nav id="main-nav" className={navOpen ? 'nav nav-open' : 'nav'}>
          {links.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              end={l.end}
              className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
              onClick={() => setNavOpen(false)}
            >
              {l.label}
            </NavLink>
          ))}
        </nav>
      </header>
      <StatusBanner />
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
