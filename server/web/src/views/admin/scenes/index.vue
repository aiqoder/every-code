<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  createAdminScene,
  deleteAdminScene,
  fetchAdminScenes,
  updateAdminScene,
  type Scene,
} from '@/api/scene'

const loading = ref(false)
const saving = ref(false)
const list = ref<Scene[]>([])
const formVisible = ref(false)
const detail = ref<Scene | null>(null)
const editingId = ref(0)
const form = reactive({
  title: '',
  description: '',
})

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminScenes()
    list.value = data.list ?? []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = 0
  form.title = ''
  form.description = ''
  formVisible.value = true
}

function openEdit(row: Scene) {
  editingId.value = row.id
  form.title = row.title
  form.description = row.description
  formVisible.value = true
}

async function save() {
  const title = form.title.trim()
  const description = form.description.trim()
  if (!title) {
    ElMessage.warning('请填写标题')
    return
  }
  if (!description) {
    ElMessage.warning('请填写描述')
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await updateAdminScene(editingId.value, { title, description })
      ElMessage.success('已保存')
    } else {
      await createAdminScene({ title, description })
      ElMessage.success('已创建')
    }
    formVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: Scene) {
  await ElMessageBox.confirm(`确定删除场景「${row.title}」？已创建的项目仍保留当时的描述。`, '删除确认', {
    type: 'warning',
  })
  await deleteAdminScene(row.id)
  ElMessage.success('已删除')
  await load()
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="scene-page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>场景</span>
          <el-button type="primary" @click="openCreate">新建场景</el-button>
        </div>
      </template>
      <p class="hint">
        桌面端新建项目的第二步从这里选场景。场景只有标题和描述。用户在桌面端保存的自定义场景不会出现在这张表里。
      </p>
      <el-table :data="list" empty-text="还没有场景">
        <el-table-column prop="title" label="标题" min-width="160" />
        <el-table-column label="描述" min-width="320">
          <template #default="{ row }">
            <div class="desc-cell">
              <span class="desc-preview">{{ row.description }}</span>
              <el-button link type="primary" @click="detail = row">详情</el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="updatedAt" label="更新时间" width="180" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="formVisible" :title="editingId ? '编辑场景' : '新建场景'" width="640px">
      <el-form label-width="72px">
        <el-form-item label="标题">
          <el-input v-model="form.title" maxlength="64" show-word-limit placeholder="例如：内部管理系统" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="12"
            maxlength="20000"
            show-word-limit
            placeholder="写清这个场景要做什么、给谁用、有哪些约束。描述可以比较长。"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog :model-value="!!detail" :title="detail?.title || '场景详情'" width="640px" @close="detail = null">
      <p class="detail">{{ detail?.description }}</p>
    </el-dialog>
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.hint {
  margin: 0 0 16px;
  color: var(--admin-text-muted);
  font-size: 14px;
  line-height: 1.5;
}

.desc-cell {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.desc-preview {
  flex: 1;
  min-width: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  white-space: pre-wrap;
  line-height: 1.5;
}

.detail {
  margin: 0;
  white-space: pre-wrap;
  line-height: 1.7;
  max-height: 60vh;
  overflow: auto;
}
</style>
