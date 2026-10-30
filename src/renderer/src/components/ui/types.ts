export interface SelectOption {
  label: string
  value: string | number
  title?: string
  disabled?: boolean
  divided?: boolean
  /** 设了就在选项前显示头像；空字符串用名称首字。 */
  avatar?: string
}

export interface MenuOption {
  label?: string
  key: string | number
  type?: 'divider'
  danger?: boolean
}
