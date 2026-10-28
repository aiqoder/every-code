export const themeIds = ['github-dark', 'github-light'] as const

export type ThemeId = (typeof themeIds)[number]

export const appearanceIds = ['github-dark', 'github-light', 'system'] as const

export type AppearanceId = (typeof appearanceIds)[number]

export const themeStorageKey = 'everycode.theme'

export const defaultAppearance: AppearanceId = 'system'

export interface AppearanceOption {
  id: AppearanceId
  label: string
}

export const appearanceList: AppearanceOption[] = [
  { id: 'github-dark', label: '深色' },
  { id: 'github-light', label: '浅色' },
  { id: 'system', label: '跟随系统' }
]

export interface ThemeCssVars {
  '--ec-canvas': string
  '--ec-canvas-subtle': string
  '--ec-canvas-inset': string
  '--ec-border': string
  '--ec-fg': string
  '--ec-fg-muted': string
  '--ec-accent': string
  '--ec-success': string
  '--ec-danger': string
  '--ec-attention': string
}

export interface GithubTheme {
  id: ThemeId
  label: string
  dark: boolean
  cssVars: ThemeCssVars
}

const surfaceDark = 'rgba(255,255,255,0.05)'
const surfaceLight = 'rgba(17,17,19,0.04)'

function theme(id: ThemeId, label: string, dark: boolean, cssVars: ThemeCssVars): GithubTheme {
  return { id, label, dark, cssVars }
}

export const themes: Record<ThemeId, GithubTheme> = {
  'github-dark': theme('github-dark', '深色', true, {
    '--ec-canvas': '#09090b',
    '--ec-canvas-subtle': '#141418',
    '--ec-canvas-inset': surfaceDark,
    '--ec-border': 'rgba(255,255,255,0.10)',
    '--ec-fg': '#f4f4f5',
    '--ec-fg-muted': '#a1a1aa',
    '--ec-accent': '#6d8bff',
    '--ec-success': '#3ddc97',
    '--ec-danger': '#ff6b86',
    '--ec-attention': '#ffc14d'
  }),
  'github-light': theme('github-light', '浅色', false, {
    '--ec-canvas': '#f5f5f7',
    '--ec-canvas-subtle': '#ffffff',
    '--ec-border': 'rgba(17,17,19,0.08)',
    '--ec-canvas-inset': surfaceLight,
    '--ec-fg': '#111113',
    '--ec-fg-muted': '#6b6b76',
    '--ec-accent': '#3557f5',
    '--ec-success': '#0a7a56',
    '--ec-danger': '#e11d48',
    '--ec-attention': '#b45309'
  })
}

export function isAppearanceId(value: string): value is AppearanceId {
  return appearanceIds.some((id) => id === value)
}

export function resolveTheme(appearance: AppearanceId, prefersDark: boolean): ThemeId {
  if (appearance === 'system') return prefersDark ? 'github-dark' : 'github-light'
  return appearance
}
