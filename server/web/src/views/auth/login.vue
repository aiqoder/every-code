<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'

import BrandLogo from '@client/components/BrandLogo.vue'
import { useBrand } from '@client/composables/useBrand'

import { oauthAuthorize } from '@/api/auth'
import { webBrandApiBase } from '@/lib/brand'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const { brand } = useBrand(webBrandApiBase)
const router = useRouter()
const route = useRoute()

const formRef = ref<FormInstance>()
const loading = ref(false)
const oauthLoading = ref(false)
const form = reactive({
  username: '',
  password: '',
})

const rules: FormRules = {
  username: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function onSubmit() {
  await formRef.value?.validate()
  loading.value = true
  try {
    const user = await auth.login(form.username, form.password)
    ElMessage.success('欢迎回来')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    if (user.role === 'admin') {
      router.replace(redirect.startsWith('/') ? redirect : '/admin/dashboard')
    } else {
      router.replace('/guest')
    }
  } finally {
    loading.value = false
  }
}

async function onOAuthLogin() {
  oauthLoading.value = true
  try {
    const res = await oauthAuthorize()
    window.location.href = res.url
  } finally {
    oauthLoading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="panel">
      <div class="intro">
        <p class="eyebrow-label">
          <BrandLogo class="eyebrow-logo" :logo="brand.logo" :alt="brand.name" />
          <span>{{ brand.name }}</span>
        </p>
        <h1>登录</h1>
        <p class="desc">管理员进入后台。日常工作请使用 ADE 桌面端。</p>
      </div>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent>
        <el-form-item label="账号" prop="username">
          <el-input v-model="form.username" placeholder="请输入账号" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="密码" />
        </el-form-item>
        <el-button type="primary" class="submit" :loading="loading" @click="onSubmit">登录</el-button>
        <el-button class="oauth" :loading="oauthLoading" @click="onOAuthLogin">
          使用统一账号登录
        </el-button>
      </el-form>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100%;
  display: grid;
  place-items: center;
  padding: 32px 16px;
  background: var(--admin-main-bg);
}

.panel {
  width: min(400px, 100%);
  padding: 24px;
  border-radius: 6px;
  background: var(--admin-header-bg);
  border: 1px solid var(--admin-border);
  box-shadow: var(--shadow, 0 1px 0 rgba(27, 31, 36, 0.04));
}

.intro {
  margin-bottom: 20px;
}

.eyebrow-label {
  margin: 0 0 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.eyebrow-logo {
  font-size: 16px;
}

h1 {
  margin: 0 0 8px;
  font-size: 24px;
  font-weight: 600;
  line-height: 1.25;
  letter-spacing: -0.01em;
  color: var(--admin-text);
}

.desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--admin-text-muted);
}

.submit {
  width: 100%;
  margin-top: 4px;
}

.oauth {
  width: 100%;
  margin: 8px 0 0;
}
</style>
