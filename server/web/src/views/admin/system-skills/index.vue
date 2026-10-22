<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

import {
  fetchSystemSkills,
  saveSystemSkills,
  type AgentOption,
  type SystemSkill,
} from '@/api/system-skill'

const loading = ref(false)
const saving = ref(false)
const rows = reactive<SystemSkill[]>([])
const agents = ref<AgentOption[]>([])

async function load() {
  loading.value = true
  try {
    const data = await fetchSystemSkills()
    rows.splice(0, rows.length, ...(data.list ?? []))
    agents.value = data.agents ?? []
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const data = await saveSystemSkills(rows.map((row) => ({ id: row.id, agentId: row.agentId ?? '' })))
    rows.splice(0, rows.length, ...(data.list ?? []))
    ElMessage.success('已保存')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="skills-page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>系统技能</span>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        </div>
      </template>
      <p class="hint">
        每个技能只能绑定一个智能体。未选择智能体的技能不会进入任何 Agent 的 catalog，桌面端也不会加载。
      </p>
      <el-empty v-if="!rows.length && !loading" description="server/skills 下还没有技能目录" />
      <el-table v-else :data="rows" stripe>
        <el-table-column prop="id" label="技能 ID" min-width="140" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="description" label="说明" min-width="200" show-overflow-tooltip />
        <el-table-column label="绑定智能体" min-width="200">
          <template #default="{ row }">
            <el-select v-model="row.agentId" clearable placeholder="不使用" style="width: 100%">
              <el-option v-for="a in agents" :key="a.id" :label="`${a.name} (${a.id})`" :value="a.id" />
            </el-select>
          </template>
        </el-table-column>
      </el-table>
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
  margin: 0 0 16px;
  color: var(--admin-text-muted);
  font-size: 14px;
  line-height: 1.5;
}
</style>
