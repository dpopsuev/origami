import { useCallback, useEffect, useState } from 'react'

export type ThemePreference = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'kami-theme'

function getStored(): ThemePreference {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'light' || v === 'dark') return v
  } catch {
    // SSR or restricted storage
  }
  return 'system'
}

function applyClass(pref: ThemePreference) {
  const root = document.documentElement
  root.classList.remove('light', 'dark')
  if (pref !== 'system') {
    root.classList.add(pref)
  }
}

export function useTheme() {
  const [preference, setPreference] = useState<ThemePreference>(getStored)

  useEffect(() => {
    applyClass(preference)
    try {
      localStorage.setItem(STORAGE_KEY, preference)
    } catch {
      // restricted storage
    }
  }, [preference])

  // Keep in sync with OS changes when preference is 'system'
  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => {
      if (preference === 'system') applyClass('system')
    }
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [preference])

  const cycle = useCallback(() => {
    setPreference((prev) => {
      const order: ThemePreference[] = ['system', 'light', 'dark']
      return order[(order.indexOf(prev) + 1) % order.length]
    })
  }, [])

  return { preference, setPreference, cycle } as const
}
