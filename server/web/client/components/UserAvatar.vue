<script setup lang="ts">
import { computed } from 'vue'

/**
 * 用户头像组件：基于用户名 hash 生成稳定的双色渐变头像，
 * 首字符作为显示字母，无需后端提供图片资源。
 */
const props = withDefaults(
  defineProps<{
    /** 用户名，用于计算颜色与显示字母 */
    name?: string
    /** 头像尺寸（px） */
    size?: number
  }>(),
  {
    name: '',
    size: 36,
  },
)

/** 色相配对表：挑饱和度/亮度适中的组合，暗亮主题下均可读 */
const HUE_PAIRS = [
  [14, 36],
  [92, 160],
  [174, 210],
  [220, 258],
  [280, 320],
  [336, 18],
  [44, 74],
  [196, 240],
]

/** 简单字符串 hash，保证同名用户颜色稳定 */
function hashCode(str: string): number {
  let h = 0
  for (let i = 0; i < str.length; i++) {
    h = (h << 5) - h + str.charCodeAt(i)
    h |= 0
  }
  return Math.abs(h)
}

const style = computed(() => {
  const label = (props.name || '?').trim()
  const pair = HUE_PAIRS[hashCode(label) % HUE_PAIRS.length] ?? [14, 36]
  const [h1, h2] = pair
  return {
    width: `${props.size}px`,
    height: `${props.size}px`,
    fontSize: `${Math.round(props.size * 0.42)}px`,
    background: `linear-gradient(135deg, hsl(${h1} 72% 55%), hsl(${h2} 68% 48%))`,
  }
})

const initial = computed(() => {
  const label = (props.name || '?').trim()
  return label ? label.slice(0, 1).toUpperCase() : '?'
})
</script>

<template>
  <span class="user-avatar" :style="style" :title="name || undefined" aria-hidden="true">
    {{ initial }}
  </span>
</template>

<style scoped>
.user-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 50%;
  color: rgba(255, 255, 255, 0.94);
  font-weight: 700;
  font-family: var(--display);
  letter-spacing: 0;
  user-select: none;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.18);
}
</style>
