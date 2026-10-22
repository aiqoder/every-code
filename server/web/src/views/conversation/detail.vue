<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import { fetchConversation } from '@/api/conversation'
import type { ChatMessage, ConversationDetail } from '@/types/api'

interface ToolCallMeta {
  id?: string
  name?: string
  arguments?: string
}

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const detail = ref<ConversationDetail | null>(null)

const conversationId = computed(() => Number(route.params.id))

const roleMeta: Record<string, { label: string; type: 'primary' | 'success' | 'warning' | 'info' | 'danger' }> = {
  user: { label: '用户', type: 'primary' },
  assistant: { label: '助手', type: 'success' },
  tool: { label: '工具', type: 'warning' },
  system: { label: '系统', type: 'info' },
}

function roleLabel(role: string) {
  return roleMeta[role]?.label ?? role
}

function roleTagType(role: string) {
  return roleMeta[role]?.type ?? 'info'
}

function parseToolCalls(metadata: string): ToolCallMeta[] {
  if (!metadata?.trim()) return []
  try {
    const parsed = JSON.parse(metadata)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function prettyJSON(raw: string) {
  if (!raw?.trim()) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

function formatArgs(args?: string) {
  if (!args) return ''
  return prettyJSON(args) || args
}

async function loadDetail() {
  if (!conversationId.value || Number.isNaN(conversationId.value)) {
    ElMessage.error('无效的会话 ID')
    return
  }
  loading.value = true
  try {
    detail.value = await fetchConversation(conversationId.value)
  } catch (e: any) {
    ElMessage.error(e?.message || '加载会话失败')
    detail.value = null
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push('/admin/conversations')
}

function messageSummary(msg: ChatMessage) {
  const parts: string[] = []
  if (msg.toolName) parts.push(`工具: ${msg.toolName}`)
  if (msg.toolCallId) parts.push(`CallID: ${msg.toolCallId}`)
  const calls = parseToolCalls(msg.metadata)
  if (calls.length) parts.push(`发起 ${calls.length} 个工具调用`)
  return parts.join(' · ')
}

onMounted(loadDetail)
watch(conversationId, loadDetail)
</script>

<template>
  <div v-loading="loading" class="detail-page">
    <el-card shadow="never" class="meta-card">
      <template #header>
        <div class="card-header">
          <div class="title-row">
            <el-button :icon="ArrowLeft" text @click="goBack">返回</el-button>
            <span class="title">会话详情 #{{ conversationId }}</span>
          </div>
          <el-button @click="loadDetail">刷新</el-button>
        </div>
      </template>

      <template v-if="detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="标题">{{ detail.title || '-' }}</el-descriptions-item>
          <el-descriptions-item label="消息数">{{ detail.messages?.length ?? 0 }}</el-descriptions-item>
          <el-descriptions-item label="模型">{{ detail.modelName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="供应商">{{ detail.providerName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="客户端">
            <el-tag v-if="detail.clientId" size="small" type="info">{{ detail.clientId }}</el-tag>
            <span v-else>未标记</span>
          </el-descriptions-item>
          <el-descriptions-item label="模型 ID">{{ detail.modelId }}</el-descriptions-item>
          <el-descriptions-item label="会话 ID">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ detail.createdAt }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ detail.updatedAt }}</el-descriptions-item>
        </el-descriptions>
      </template>
      <el-empty v-else-if="!loading" description="会话不存在或加载失败" />
    </el-card>

    <el-card v-if="detail" shadow="never" class="flow-card">
      <template #header>
        <span>全流程消息（按时间顺序）</span>
      </template>

      <el-empty v-if="!detail.messages?.length" description="暂无消息" />

      <div v-else class="timeline">
        <div
          v-for="(msg, index) in detail.messages"
          :key="msg.id"
          class="msg-item"
          :class="`role-${msg.role}`"
        >
          <div class="msg-head">
            <span class="seq">#{{ index + 1 }}</span>
            <el-tag size="small" :type="roleTagType(msg.role)">{{ roleLabel(msg.role) }}</el-tag>
            <span class="msg-id">msg {{ msg.id }}</span>
            <span class="time">{{ msg.createdAt }}</span>
          </div>

          <div v-if="messageSummary(msg)" class="msg-meta">{{ messageSummary(msg) }}</div>

          <div v-if="msg.content" class="msg-content">
            <pre>{{ msg.content }}</pre>
          </div>
          <div v-else class="msg-content empty">（无文本内容）</div>

          <div v-if="parseToolCalls(msg.metadata).length" class="tool-calls">
            <div class="section-title">工具调用（metadata）</div>
            <div
              v-for="(call, ci) in parseToolCalls(msg.metadata)"
              :key="call.id || ci"
              class="tool-call"
            >
              <div class="tool-call-head">
                <el-tag size="small" type="warning">{{ call.name || 'unknown' }}</el-tag>
                <span v-if="call.id" class="call-id">{{ call.id }}</span>
              </div>
              <details v-if="call.arguments" open>
                <summary>参数</summary>
                <pre>{{ formatArgs(call.arguments) }}</pre>
              </details>
            </div>
          </div>

          <details v-if="msg.metadata && !parseToolCalls(msg.metadata).length" class="raw-meta">
            <summary>原始 metadata</summary>
            <pre>{{ prettyJSON(msg.metadata) || msg.metadata }}</pre>
          </details>

          <details class="raw-json">
            <summary>原始消息 JSON</summary>
            <pre>{{ JSON.stringify(msg, null, 2) }}</pre>
          </details>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.detail-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.title {
  font-weight: 600;
}

.timeline {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.msg-item {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 12px 14px;
  background: #fff;
}

.msg-item.role-user {
  border-left: 3px solid #409eff;
}

.msg-item.role-assistant {
  border-left: 3px solid #67c23a;
}

.msg-item.role-tool {
  border-left: 3px solid #e6a23c;
  background: #fafafa;
}

.msg-item.role-system {
  border-left: 3px solid #909399;
}

.msg-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.seq {
  font-weight: 600;
  color: #303133;
  min-width: 28px;
}

.msg-id {
  color: #909399;
  font-size: 12px;
}

.time {
  margin-left: auto;
  color: #909399;
  font-size: 12px;
}

.msg-meta {
  margin-bottom: 8px;
  font-size: 12px;
  color: #606266;
}

.msg-content pre,
.tool-call pre,
.raw-meta pre,
.raw-json pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  line-height: 1.5;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.msg-content {
  background: #f5f7fa;
  border-radius: 6px;
  padding: 10px 12px;
}

.msg-content.empty {
  color: #c0c4cc;
  font-size: 13px;
}

.section-title {
  margin: 10px 0 6px;
  font-size: 13px;
  font-weight: 500;
  color: #606266;
}

.tool-call {
  border: 1px solid #f0e6d2;
  background: #fffbf0;
  border-radius: 6px;
  padding: 8px 10px;
  margin-bottom: 8px;
}

.tool-call-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.call-id {
  font-size: 12px;
  color: #909399;
}

.tool-call details,
.raw-meta,
.raw-json {
  margin-top: 6px;
}

.tool-call summary,
.raw-meta summary,
.raw-json summary {
  cursor: pointer;
  color: #606266;
  font-size: 12px;
}

.tool-call pre,
.raw-meta pre,
.raw-json pre {
  margin-top: 6px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 8px;
  max-height: 360px;
  overflow: auto;
}

.raw-json {
  margin-top: 8px;
}
</style>
