<template>
  <a-modal v-model:visible="visible" title="修改密码" width="500px">
    <a-form ref="formRef" @submit.prevent="handleSubmit">
      <a-form-item field="oldPassword" label="旧密码" :rules="[{ required: true, message: '请输入旧密码' }]">
        <a-input-password v-model="formData.oldPassword" placeholder="请输入旧密码">
          <template #prefix><icon-lock /></template>
        </a-input-password>
      </a-form-item>
      
      <a-form-item field="newPassword" label="新密码" :rules="newPasswordRules">
        <a-input-password v-model="formData.newPassword" placeholder="请输入新密码">
          <template #prefix><icon-lock /></template>
        </a-input-password>
      </a-form-item>
      
      <a-form-item field="confirmPassword" label="确认新密码" :rules="confirmPasswordRules">
        <a-input-password v-model="formData.confirmPassword" placeholder="请确认新密码">
          <template #prefix><icon-lock /></template>
        </a-input-password>
      </a-form-item>
    </a-form>
    
    <template #footer>
      <a-space>
        <a-button @click="visible = false">取消</a-button>
        <a-button type="primary" :loading="loading" @click="handleSubmit">确认修改</a-button>
      </a-space>
    </template>
  </a-modal>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { IconLock } from '@arco-design/web-vue/es/icon'
import LoginApi from '@/api/login'
import { showMessage } from '@/composables/useGlobalState'
import sha256 from 'crypto-js/sha256'
import encHex from 'crypto-js/enc-hex'
import { userStore } from '@/stores/user'
import { useRouter } from 'vue-router'

const visible = ref(false)
const loading = ref(false)
const formRef = ref(null)
const users = userStore()
const router = useRouter()

const formData = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const newPasswordRules = [
  { required: true, message: '请输入新密码' },
  { minLength: 8, message: '密码长度至少8位' }
]

const confirmPasswordRules = [
  { required: true, message: '请确认新密码' },
  (value) => value === formData.newPassword || '两次输入的密码不一致'
]

const open = () => {
  formData.oldPassword = ''
  formData.newPassword = ''
  formData.confirmPassword = ''
  visible.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value.validate()
  if (!valid) return

  loading.value = true
  try {
    // 1. Get Nonce
    const nonceRes = await LoginApi.getNonce()
    let nonce = ''
    if (nonceRes.code === '0' && nonceRes.data?.nonce) {
      nonce = nonceRes.data.nonce
    } else {
      throw new Error('获取安全令牌失败')
    }

    // 2. Hash Old Password
    const hashedOld = sha256(formData.oldPassword + nonce).toString(encHex)

    // 3. Send Request
    const res = await LoginApi.changePassword({
      oldPassword: hashedOld,
      newPassword: formData.newPassword,
      nonce: nonce
    })

    if (res.code === '0') {
      showMessage('密码修改成功，请重新登录', 'success')
      visible.value = false
      // Logout logic
      localStorage.removeItem('loginInfo')
      router.push('/login')
    } else {
      showMessage(res.msg || '修改失败', 'error')
    }
  } catch (error) {
    console.error(error)
    showMessage(error.message || '系统异常', 'error')
  } finally {
    loading.value = false
  }
}

defineExpose({ open })
</script>

<style scoped>
/* 修改密码对话框样式 */
:deep(.arco-form-item-label) {
  font-weight: 500;
  white-space: nowrap !important;
}

:deep(.arco-form-item-control) {
  min-height: 32px;
}

:deep(.arco-input-wrapper) {
  height: 32px;
}

:deep(.arco-input) {
  height: 32px;
  line-height: 32px;
}
</style>

