import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'

let perms: string[] = ['*']
vi.mock('@/stores/auth', () => ({
  useCan: () => (p: string) => perms.includes('*') || perms.includes(p),
}))

import { PageHead } from './PageHead'
import { hasInAppHistory } from '@/lib/goBack'

// MemoryRouter keeps its own stack, but hasInAppHistory() reads the real
// BrowserRouter stamp in window.history.state — set it to simulate "the tab
// opened here" (idx 0) vs "navigated here inside the console" (idx > 0).
function setIdx(idx: number | undefined) {
  window.history.replaceState(idx === undefined ? null : { idx }, '')
}

let path = ''
function Probe() {
  path = useLocation().pathname
  return null
}

function renderStack(entries: string[], back: string) {
  return render(
    <MemoryRouter initialEntries={entries} initialIndex={entries.length - 1}>
      <Routes>
        <Route path="*" element={<><PageHead title="T" back={back} /><Probe /></>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  perms = ['*']
  setIdx(undefined)
})

describe('hasInAppHistory', () => {
  it('is false on the entry the tab opened on, or with no router stamp', () => {
    setIdx(undefined)
    expect(hasInAppHistory()).toBe(false)
    setIdx(0)
    expect(hasInAppHistory()).toBe(false)
  })
  it('is true once the app has pushed an entry', () => {
    setIdx(2)
    expect(hasInAppHistory()).toBe(true)
  })
})

describe('BackButton (via PageHead back=)', () => {
  it('goes to the previous in-app page when there is one (keeps its filters)', () => {
    setIdx(1)
    renderStack(['/orders?status=failed', '/orders/abc'], '/orders')
    fireEvent.click(screen.getByRole('button', { name: /back/i }))
    expect(path).toBe('/orders')
  })

  it('falls back to the logical parent when opened directly (pasted link / new tab)', () => {
    setIdx(0)
    renderStack(['/orders/abc'], '/orders')
    fireEvent.click(screen.getByRole('button', { name: /back/i }))
    expect(path).toBe('/orders')
  })

  it('top-level pages fall back to the Dashboard', () => {
    renderStack(['/orders'], '/')
    fireEvent.click(screen.getByRole('button', { name: /back/i }))
    expect(path).toBe('/')
  })

  it("a role that can't view the Dashboard falls back to its first permitted page", () => {
    // /todos has no RBAC domain, so it is every admin's first page after the Dashboard.
    perms = ['orders.view', 'users.view']
    renderStack(['/users'], '/')
    fireEvent.click(screen.getByRole('button', { name: /back/i }))
    expect(path).toBe('/todos')
  })

  it('is hidden when the fallback is the current page and there is no history', () => {
    perms = ['orders.view']
    renderStack(['/todos'], '/')
    expect(screen.queryByRole('button', { name: /back/i })).toBeNull()
  })

  it('is a keyboard-focusable button', () => {
    renderStack(['/orders'], '/')
    const b = screen.getByRole('button', { name: /back/i })
    expect(b.tagName).toBe('BUTTON')
    expect(b.getAttribute('type')).toBe('button')
  })
})
