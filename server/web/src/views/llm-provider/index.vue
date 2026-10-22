<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'

import {
  createAdminLlmProvider,
  fetchAdminLlmProviders,
  updateAdminLlmProvider,
} from '@/api/admin-llm-provider'
import type { LlmProvider, LlmProviderPayload } from '@/types/api'
import ProviderFormDialog from './components/ProviderFormDialog.vue'

const loading = ref(false)
const list = ref<LlmProvider[]>([])
const total = ref(0)
const query = reactive({
  page: 1,
  pageSize: 10,
})
const formVisible = ref(false)
const editing = ref<LlmProvider | null>(null)

async function loadList() {
  loading.value = true
  try {
    const data = await fetchAdminLlmProviders({
      page: query.page,
      pageSize: query.pageSize,
    })
    list.value = data.list ?? []
    total.value = data.total ?? 0
  } finally {
    loading.value = false
  }
}

function onPageChange(page: number) {
  query.page = page
  loadList()
}

function onSizeChange(size: number) {
  query.pageSize = size
  query.page = 1
  loadList()
}

function openCreate() {
  editing.value = null
  formVisible.value = true
}

function openEdit(row: LlmProvider) {
  editing.value = row
  formVisible.value = true
}

function onSaved() {
  if (!editing.value) {
    query.page = 1
  }
  loadList()
}

async function submitProvider(payload: LlmProviderPayload) {
  if (editing.value?.id) {
    await updateAdminLlmProvider(editing.value.id, payload)
  } else {
    await createAdminLlmProvider(payload)
    query.page = 1
  }
}

function maskApiKey(key?: string) {
  if (!key) return '-'
  if (key.length <= 8) return '****'
  return `${key.slice(0, 4)}****${key.slice(-4)}`
}

onMounted(loadList)
</script>

<template>
  <div class="llm-provider-page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>大模型供应商</span>
          <el-button type="primary" @click="openCreate">新建供应商</el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="供应商名称" min-width="140" />
        <el-table-column prop="website" label="官方网站" min-width="200" show-overflow-tooltip />
        <el-table-column label="API Key" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            {{ maskApiKey(row.apiKey) }}
          </template>
        </el-table-column>
        <el-table-column
          prop="openaiCompatibleBaseUrl"
          label="OpenAI Compatible BaseUrl"
          min-width="220"
          show-overflow-tooltip
        />
        <el-table-column
          prop="anthropicBaseUrl"
          label="Anthropic BaseUrl"
          min-width="200"
          show-overflow-tooltip
        />
        <el-table-column prop="createdAt" label="创建时间" min-width="180" />
        <el-table-column prop="updatedAt" label="更新时间" min-width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          background
          layout="total, sizes, prev, pager, next"
          :total="total"
          :current-page="query.page"
          :page-size="query.pageSize"
          :page-sizes="[10, 20, 50]"
          @current-change="onPageChange"
          @size-change="onSizeChange"
        />
      </div>
    </el-card>

    <ProviderFormDialog v-model="formVisible" :editing="editing" :submit="submitProvider" @success="onSaved" />
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
