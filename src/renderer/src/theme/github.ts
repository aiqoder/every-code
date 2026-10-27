import type { GlobalThemeOverrides } from 'naive-ui'

export const themeIds = ['github-dark', 'github-light', 'github-dimmed'] as const

export type ThemeId = (typeof themeIds)[number]

export const themeStorageKey = 'everycode.theme'

export const defaultThemeId: ThemeId = 'github-dark'

const fontFamily =
  "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif"

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
  overrides: GlobalThemeOverrides
}

interface ThemeSource {
  id: ThemeId
  label: string
  dark: boolean
  canvas: string
  canvasSubtle: string
  canvasInset: string
  border: string
  fg: string
  fgMuted: string
  fgSubtle: string
  accent: string
  accentHover: string
  accentPressed: string
  onAccent: string
  success: string
  danger: string
  attention: string
  hover: string
}

function createTheme(source: ThemeSource): GithubTheme {
  return {
    id: source.id,
    label: source.label,
    dark: source.dark,
    cssVars: {
      '--ec-canvas': source.canvas,
      '--ec-canvas-subtle': source.canvasSubtle,
      '--ec-canvas-inset': source.canvasInset,
      '--ec-border': source.border,
      '--ec-fg': source.fg,
      '--ec-fg-muted': source.fgMuted,
      '--ec-accent': source.accent,
      '--ec-success': source.success,
      '--ec-danger': source.danger,
      '--ec-attention': source.attention
    },
    overrides: {
      common: {
        primaryColor: source.accent,
        primaryColorHover: source.accentHover,
        primaryColorPressed: source.accentPressed,
        primaryColorSuppl: source.accentHover,
        bodyColor: source.canvas,
        cardColor: source.canvasSubtle,
        modalColor: source.canvasSubtle,
        popoverColor: source.canvasSubtle,
        textColor1: source.fg,
        textColor2: source.fgMuted,
        textColor3: source.fgSubtle,
        textColorBase: source.fg,
        placeholderColor: source.fgMuted,
        borderColor: source.border,
        dividerColor: source.border,
        hoverColor: source.hover,
        inputColor: source.canvasInset,
        codeColor: source.canvasInset,
        tableColor: source.canvas,
        successColor: source.success,
        errorColor: source.danger,
        warningColor: source.attention,
        fontFamily
      },
      Button: {
        textColorPrimary: source.onAccent,
        textColorHoverPrimary: source.onAccent,
        textColorPressedPrimary: source.onAccent,
        textColorFocusPrimary: source.onAccent,
        textColorDisabledPrimary: source.onAccent
      }
    }
  }
}

export const themes: Record<ThemeId, GithubTheme> = {
  'github-dark': createTheme({
    id: 'github-dark',
    label: '深色',
    dark: true,
    canvas: '#0d1117',
    canvasSubtle: '#161b22',
    canvasInset: '#010409',
    border: '#30363d',
    fg: '#e6edf3',
    fgMuted: '#8b949e',
    fgSubtle: '#6e7681',
    accent: '#1f6feb',
    accentHover: '#388bfd',
    accentPressed: '#1158c7',
    onAccent: '#ffffff',
    success: '#3fb950',
    danger: '#f85149',
    attention: '#d29922',
    hover: 'rgba(177, 186, 196, 0.12)'
  }),
  'github-light': createTheme({
    id: 'github-light',
    label: '浅色',
    dark: false,
    canvas: '#ffffff',
    canvasSubtle: '#f6f8fa',
    canvasInset: '#f6f8fa',
    border: '#d0d7de',
    fg: '#1f2328',
    fgMuted: '#656d76',
    fgSubtle: '#6e7781',
    accent: '#0969da',
    accentHover: '#0860ca',
    accentPressed: '#0550ae',
    onAccent: '#ffffff',
    success: '#1a7f37',
    danger: '#d1242f',
    attention: '#9a6700',
    hover: 'rgba(208, 215, 222, 0.48)'
  }),
  'github-dimmed': createTheme({
    id: 'github-dimmed',
    label: '暗淡',
    dark: true,
    canvas: '#22272e',
    canvasSubtle: '#2d333b',
    canvasInset: '#1c2128',
    border: '#444c56',
    fg: '#adbac7',
    fgMuted: '#768390',
    fgSubtle: '#636e7b',
    accent: '#316dca',
    accentHover: '#4184e4',
    accentPressed: '#2559a7',
    onAccent: '#ffffff',
    success: '#57ab5a',
    danger: '#e5534b',
    attention: '#c69026',
    hover: 'rgba(173, 186, 199, 0.14)'
  })
}

export const themeList: GithubTheme[] = themeIds.map((id) => themes[id])

export function isThemeId(value: string): value is ThemeId {
  return themeIds.some((id) => id === value)
}
