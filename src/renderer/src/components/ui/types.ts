export interface SelectOption {
  label: string
  value: string | number
  title?: string
  disabled?: boolean
  divided?: boolean
}

export interface MenuOption {
  label?: string
  key: string | number
  type?: 'divider'
  danger?: boolean
}
