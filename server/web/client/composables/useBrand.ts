import { onScopeDispose, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'

import { DEFAULT_BRAND, loadBrand, type Brand } from '../lib/brand'

/** 按接口前缀加载品牌。地址变化时重新读取，请求失败保持默认值。 */
export function useBrand(apiBase: MaybeRefOrGetter<string>) {
  const brand = ref<Brand>({ ...DEFAULT_BRAND })
  let ticket = 0
  let timer = 0

  watch(
    () => toValue(apiBase).trim(),
    (base) => {
      window.clearTimeout(timer)
      const current = ++ticket
      timer = window.setTimeout(() => {
        loadBrand(base)
          .then((next) => {
            if (current === ticket) brand.value = next
          })
          .catch(() => {
            if (current === ticket) brand.value = { ...DEFAULT_BRAND }
          })
      }, 300)
    },
    { immediate: true },
  )

  onScopeDispose(() => {
    ticket += 1
    window.clearTimeout(timer)
  })

  return { brand }
}
