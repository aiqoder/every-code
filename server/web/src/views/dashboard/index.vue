<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchHealth } from '@/api/health'

const loading = ref(false)
const status = ref('-')
const error = ref('')

async function loadHealth() {
  loading.value = true
  error.value = ''
  try {
    const data = await fetchHealth()
    status.value = data.status
  } catch (e) {
    status.value = 'unavailable'
    error.value = e instanceof Error ? e.message : '检查失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadHealth)
</script>

<template>
  <div class="dashboard">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>服务状态</span>
          <el-button size="small" :loading="loading" @click="loadHealth">刷新</el-button>
        </div>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="API Health">
          <el-tag :type="status === 'ok' ? 'success' : 'danger'">{{ status }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="error" label="说明">{{ error }}</el-descriptions-item>
      </el-descriptions>
      <p class="hint">对接后端 <code>GET /api/health</code>，开发环境经 Vite 代理到 :8182。</p>
    </el-card>
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.hint {
  margin-top: 16px;
  color: #909399;
  font-size: 13px;
}

.hint code {
  background: #f2f3f5;
  padding: 2px 6px;
  border-radius: 4px;
}
</style>
