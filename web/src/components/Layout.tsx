import { useEffect, useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { useT } from '../i18n'
import { CommandPalette } from './CommandPalette'
import { HostPicker } from './HostPicker'
import { StatusBanner } from './StatusBanner'

export function Layout() {
  const t = useT()
  const [navOpen, setNavOpen] = useState(false)
  const [paletteOpen, setPaletteOpen] = useState(false)

  const links = [
    { to: '/', label: t('nav.dashboard'), end: true },
    { to: '/containers', label: t('nav.containers') },
    { to: '/stacks', label: t('nav.stacks') },
    { to: '/networks', label: t('nav.networks') },
    { to: '/volumes', label: t('nav.volumes') },
    { to: '/images', label: t('nav.images') },
    { to: '/graph', label: t('nav.graph') },
    { to: '/diagnostics', label: t('nav.diagnostics') },
    { to: '/snapshots', label: t('nav.snapshots') },
    { to: '/settings', label: t('nav.settings') },
  ]

  useEffect(() => {
    const onResize = () => {
      if (window.innerWidth > 900) setNavOpen(false)
    }
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setPaletteOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar-row">
          <div className="brand">
            <span className="brand-mark">DV</span>
            <div>
              <div className="brand-title">{t('brand.title')}</div>
              <div className="brand-sub">{t('brand.sub')}</div>
            </div>
          </div>
          <button
            type="button"
            className="btn palette-launch"
            title={t('palette.hint')}
            onClick={() => setPaletteOpen(true)}
          >
            {t('palette.hint')}
          </button>
          <button
            type="button"
            className="nav-toggle"
            aria-expanded={navOpen}
            aria-controls="main-nav"
            onClick={() => setNavOpen((v) => !v)}
          >
            {navOpen ? t('nav.close') : t('nav.menu')}
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
          <HostPicker />
        </nav>
      </header>
      <StatusBanner />
      <main className="main">
        <Outlet />
      </main>
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
    </div>
  )
}
