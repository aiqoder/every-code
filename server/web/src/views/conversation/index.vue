<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'

import { deleteConversation, fetchConversations } from '@/api/conversation'
import type { Conversation } from '@/types/api'

const router = useRouter()
const loading = ref(false)
const list = ref<Conversation[]>([])
const total = ref(0)
const query = reactive({
  page: 1,
  pageSize: 10,
})

async function loadList() {
  loading.value = true
  try {
    const data = await fetchConversations({
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

function openDetail(row: Conversation) {
  router.push(`/admin/conversations/${row.id}`)
}

async function onDelete(row: Conversation) {
  await ElMessageBox.confirm(`确定删除会话「${row.title || row.id}」？`, '提示', {
    type: 'warning',
  })
  await deleteConversation(row.id)
  ElMessage.success('已删除')
  if (list.value.length === 1 && query.page > 1) {
    query.page -= 1
  }
  await loadList()
}

onMounted(loadList)
</script>

<template>
  <div class="conversation-page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>会话查看</span>
          <el-button @click="loadList">刷新</el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
        <el-table-column label="客户端" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tag v-if="row.clientId" size="small" type="info">{{ row.clientId }}</el-tag>
            <span v-else class="muted">未标记</span>
          </template>
        </el-table-column>
        <el-table-column prop="modelName" label="模型" min-width="140" show-overflow-tooltip />
        <el-table-column prop="providerName" label="供应商" min-width="120" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="创建时间" min-width="170" />
        <el-table-column prop="updatedAt" label="更新时间" min-width="170" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">查看</el-button>
            <el-button link type="danger" @click="onDelete(row)">删除</el-button>
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

.muted {
  color: var(--el-text-color-placeholder);
}
</style>
