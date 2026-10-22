<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'

import { fetchAdminProviderRemoteModels, fetchAllAdminLlmProviders } from '@/api/admin-llm-provider'
import { createAdminLlmModel, fetchAdminLlmModels, updateAdminLlmModel } from '@/api/admin-llm-model'
import type { LlmModel, LlmModelPayload, LlmProvider } from '@/types/api'
import ModelFormDialog from './components/ModelFormDialog.vue'

const loading = ref(false)
const list = ref<LlmModel[]>([])
const total = ref(0)
const providers = ref<LlmProvider[]>([])
const query = reactive({
  page: 1,
  pageSize: 10,
  providerId: undefined as number | undefined,
})
const formVisible = ref(false)
const editing = ref<LlmModel | null>(null)

async function loadProviders() {
  const data = await fetchAllAdminLlmProviders()
  providers.value = data.list ?? []
}

async function loadList() {
  loading.value = true
  try {
    const data = await fetchAdminLlmModels({
      page: query.page,
      pageSize: query.pageSize,
      providerId: query.providerId,
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

function onProviderFilterChange() {
  query.page = 1
  loadList()
}

function openCreate() {
  editing.value = null
  formVisible.value = true
}

function openEdit(row: LlmModel) {
  editing.value = row
  formVisible.value = true
}

function onSaved() {
  if (!editing.value) {
    query.page = 1
  }
  loadList()
}

async function submitModel(payload: LlmModelPayload) {
  if (editing.value?.id) {
    await updateAdminLlmModel(editing.value.id, payload)
  } else {
    await createAdminLlmModel(payload)
    query.page = 1
  }
}

function formatCaps(caps: string[]) {
  return caps?.length ? caps.join(', ') : '-'
}

function formatOutputLimit(v: number) {
  return v === 0 ? '不限制' : String(v)
}

onMounted(async () => {
  await loadProviders()
  await loadList()
})
</script>

<template>
  <div class="llm-model-page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>大模型维护</span>
          <div class="header-actions">
            <el-select
              v-model="query.providerId"
              clearable
              placeholder="按供应商筛选"
              style="width: 200px"
              @change="onProviderFilterChange"
            >
              <el-option
                v-for="p in providers"
                :key="p.id"
                :label="p.name"
                :value="p.id"
              />
            </el-select>
            <el-button type="primary" @click="openCreate">新增大模型</el-button>
          </div>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe border>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="providerName" label="供应商" min-width="120" />
        <el-table-column prop="name" label="模型名称" min-width="140" />
        <el-table-column label="附件" width="80">
          <template #default="{ row }">
            <el-tag :type="row.attachment ? 'success' : 'info'" size="small">
              {{ row.attachment ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="推理" width="80">
          <template #default="{ row }">
            <el-tag :type="row.reasoning ? 'success' : 'info'" size="small">
              {{ row.reasoning ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="推理选项" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ formatCaps(row.reasoningOptions) }}
          </template>
        </el-table-column>
        <el-table-column label="温度" width="80">
          <template #default="{ row }">
            <el-tag :type="row.temperature ? 'success' : 'info'" size="small">
              {{ row.temperature ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="输入能力" min-width="120">
          <template #default="{ row }">
            {{ formatCaps(row.inputCapabilities) }}
          </template>
        </el-table-column>
        <el-table-column prop="inputContextLimit" label="输入上下文" min-width="110" />
        <el-table-column label="输出上下文" min-width="110">
          <template #default="{ row }">
            {{ formatOutputLimit(row.outputContextLimit) }}
          </template>
        </el-table-column>
        <el-table-column prop="inputPrice" label="输入价" min-width="100" />
        <el-table-column prop="outputPrice" label="输出价" min-width="100" />
        <el-table-column prop="cachePrice" label="缓存价" min-width="100" />
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

    <ModelFormDialog
      v-model="formVisible"
      :editing="editing"
      :providers="providers"
      :submit="submitModel"
      :fetch-remote-models="fetchAdminProviderRemoteModels"
      @success="onSaved"
    />
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
