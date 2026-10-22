<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { TableInstance } from 'element-plus'

import {
  createAdminTool,
  deleteAdminTool,
  fetchAdminTools,
  updateAdminTool,
} from '@/api/admin-tool'
import type { Tool, ToolPayload } from '@/types/api'
import ToolFormDialog from '@/views/tool/components/ToolFormDialog.vue'

const loading = ref(false)
const list = ref<Tool[]>([])
const total = ref(0)
const query = reactive({
  page: 1,
  pageSize: 10,
})
const formVisible = ref(false)
const editing = ref<Tool | null>(null)
const tableRef = ref<TableInstance>()
const selected = ref<Tool[]>([])

const deletableSelected = computed(() => selected.value.filter((row) => row.editable && row.id))

async function loadList() {
  loading.value = true
  try {
    const data = await fetchAdminTools({
      page: query.page,
      pageSize: query.pageSize,
    })
    list.value = data.list ?? []
    total.value = data.total ?? 0
    selected.value = []
    tableRef.value?.clearSelection()
  } finally {
    loading.value = false
  }
}

function onPageChange(page: number) {
  query.page = page
  loadList()
}

function openCreate() {
  editing.value = null
  formVisible.value = true
}

function openEdit(row: Tool) {
  if (!row.editable) return
  editing.value = row
  formVisible.value = true
}

function isSelectable(row: Tool) {
  return !!row.editable
}

function onSelectionChange(rows: Tool[]) {
  selected.value = rows
}

async function onDelete(row: Tool) {
  if (!row.editable || !row.id) return
  await ElMessageBox.confirm(`确定删除工具「${row.displayName}」？`, '删除确认', {
    type: 'warning',
  })
  await deleteAdminTool(row.id)
  ElMessage.success('已删除')
  loadList()
}

async function onBatchDelete() {
  const rows = deletableSelected.value
  if (!rows.length) {
    ElMessage.warning('请先选择可删除的工具')
    return
  }
  await ElMessageBox.confirm(`确定删除选中的 ${rows.length} 个工具？`, '批量删除确认', {
    type: 'warning',
  })
  await Promise.all(rows.map((row) => deleteAdminTool(row.id!)))
  ElMessage.success(`已删除 ${rows.length} 个工具`)
  loadList()
}

async function submitTool(payload: ToolPayload) {
  if (editing.value?.id) {
    await updateAdminTool(editing.value.id, payload)
  } else {
    await createAdminTool(payload)
    query.page = 1
  }
}

onMounted(loadList)
</script>

<template>
  <div class="tool-page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>模型工具管理</span>
          <div class="header-actions">
            <el-button
              type="danger"
              plain
              :disabled="!deletableSelected.length"
              @click="onBatchDelete"
            >
              批量删除
            </el-button>
            <el-button type="primary" @click="openCreate">新增工具</el-button>
          </div>
        </div>
      </template>

      <el-table
        ref="tableRef"
        v-loading="loading"
        :data="list"
        stripe
        border
        row-key="id"
        @selection-change="onSelectionChange"
      >
        <el-table-column type="selection" width="48" :selectable="isSelectable" />
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="英文名称" min-width="120" />
        <el-table-column prop="displayName" label="中文名称" min-width="120" />
        <el-table-column prop="description" label="说明" min-width="180" show-overflow-tooltip />
        <el-table-column label="来源" width="100">
          <template #default="{ row }">
            <el-tag :type="row.type === 'http' ? 'primary' : 'info'" size="small">
              {{ row.type === 'builtin' ? '系统内置' : '管理员' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="90" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <template v-if="row.editable">
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button link type="danger" @click="onDelete(row)">删除</el-button>
            </template>
            <span v-else class="readonly-tip">不可编辑</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="total"
          :current-page="query.page"
          :page-size="query.pageSize"
          @current-change="onPageChange"
        />
      </div>
    </el-card>

    <ToolFormDialog
      v-model="formVisible"
      :editing="editing"
      :submit="submitTool"
      @success="loadList"
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
  gap: 8px;
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.readonly-tip {
  color: #909399;
  font-size: 13px;
}
</style>
