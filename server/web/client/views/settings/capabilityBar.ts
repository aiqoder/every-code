import { inject, provide, reactive, type InjectionKey } from 'vue'

export interface CapabilityBarState {
  query: string
  placeholder: string
  installedCount: number
  installedOnly: boolean
  addLabel: string
  addDisabled: boolean
  onAdd: (() => void) | null
}

const key: InjectionKey<CapabilityBarState> = Symbol('capabilityBar')

export function provideCapabilityBar() {
  const state = reactive<CapabilityBarState>({
    query: '',
    placeholder: '搜索',
    installedCount: 0,
    installedOnly: false,
    addLabel: '添加',
    addDisabled: false,
    onAdd: null,
  })
  provide(key, state)
  return state
}

export function useCapabilityBar() {
  const state = inject(key)
  if (!state) throw new Error('未提供能力栏')
  return state
}
