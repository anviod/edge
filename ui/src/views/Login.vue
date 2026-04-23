<template>
  <div class="login-container">
    <div class="login-scene">
      <div class="login-panel" 
        :class="{ 'shake-animation': isShaking, 'login-card-exit': isLoginSuccess, 'countdown-10': ctxData.countdown <= 10 }"
        :style="{ '--clip-path': getClipPath }"
      >

        <div class="panel-topbar">
          <div class="logo-box">
            <div class="logo-icon">
              <span>EDGEx</span>
            </div>
          </div>
          <div class="panel-header-side">
            <span class="version-tag">VER {{ ctxData.configInfo.softVer || '2.0' }}</span>
          </div>
        </div>

        <div class="panel-title">
          <div class="title-main">边缘计算网关</div>
          <div class="title-sub">IOT THINGS GATEWAY</div>
        </div>

        <div class="auth-row">
          <a-radio-group v-model="ctxData.loginMethod" type="button" class="industrial-radio">
            <a-radio value="local"><icon-user /> 本地登录</a-radio>
            <a-radio value="ldap"><icon-user-group /> LDAP 登录</a-radio>
          </a-radio-group>
          <span class="mode-indicator" :class="{ 'is-ldap': ctxData.loginMethod === 'ldap' }"></span>
        </div>

        <a-form :model="ctxData.loginForm" layout="vertical" @submit="handleLogin" class="custom-form">
          <div class="field">
            <div class="label">用户标识 / Username</div>
            <a-input v-model="ctxData.loginForm.userName" :placeholder="ctxData.loginMethod === 'ldap' ? 'LDAP 账号 / 邮箱' : '请输入用户名'" size="large" allow-clear>
              <template #prefix>
                <icon-user v-if="ctxData.loginMethod === 'local'" />
                <icon-at v-else />
              </template>
            </a-input>
          </div>

          <div class="field">
            <div class="label">访问密钥 / Password</div>
            <a-input-password v-model="ctxData.loginForm.password" :placeholder="ctxData.loginMethod === 'ldap' ? 'LDAP 域密码' : '请输入密码'" size="large" allow-clear @keyup.enter="handleLogin">
              <template #prefix><icon-lock /></template>
            </a-input-password>
          </div>

          <div class="options">
            <a-checkbox v-model="ctxData.rememberMe" class="remember-check">记住访问权限</a-checkbox>
            <a-link v-if="ctxData.loginMethod === 'local'" class="forgot-link" @click="handleForgotPassword">忘记密码?</a-link>
          </div>

          <div v-if="ctxData.loginMethod === 'ldap'" class="ldap-hint compact-hint">
            <icon-info-circle /> <span>通过企业域控服务进行身份验证</span>
          </div>
          <div v-else class="terminal-decorator compact-terminal">
            <span class="status-dot"></span>
            <span class="terminal-code">AUTH_MODE: LOCAL_DATABASE</span>
          </div>

          <div v-if="ctxData.errorMessage" class="error-message">
            <icon-close-circle-fill /> <span>{{ ctxData.errorMessage }}</span>
          </div>

          <a-button type="primary" html-type="submit" size="large" long :loading="ctxData.loading" :disabled="isLoginSuccess" class="login-submit-btn">
            <template #icon>
              <icon-check-circle-fill v-if="isLoginSuccess" />
              <icon-arrow-right v-else />
            </template>
            {{ isLoginSuccess ? '登录成功' : (ctxData.loginMethod === 'ldap' ? '域验证登录' : '立即登录') }}
          </a-button>

          <div class="copyright-text panel-copyright">© {{ new Date().getFullYear() }} {{ ctxData.configInfo.name || '系统' }}</div>
        </a-form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref, computed } from 'vue'
import {
  IconUser, IconLock, IconArrowRight,
  IconUserGroup, IconAt, IconInfoCircle,
  IconCloseCircleFill, IconCheckCircleFill
} from '@arco-design/web-vue/es/icon'
import LoginApi from 'api/login.js'
import router from '@/router'
import { userStore } from 'stores/user.js'
import { configStore } from '@/stores/app.js'
import { useI18n } from 'vue-i18n'
import sha256 from 'crypto-js/sha256'
import encHex from 'crypto-js/enc-hex'
import { showMessage } from '@/composables/useGlobalState'

const { t } = useI18n()
const config = configStore()
const users = userStore()

const loginFormRef = ref(null)
const isShaking = ref(false)
const isLoginSuccess = ref(false)

const ctxData = reactive({
  loginForm: {
    userName: '',
    password: '',
  },
  loginMethod: 'local',
  loading: false,
  rememberMe: false,
  configInfo: config.configInfo || {},
  nonce: '',
  errorMessage: '',
  countdown: 60,
  countdownTimer: null
})

// 计算倒计时进度，用于外部轮廓动画
const countdownProgress = computed(() => {
  return (ctxData.countdown / 60) * 100
})

// 生成 clip-path 值，用于绘制外部轮廓
const getClipPath = computed(() => {
  const progress = countdownProgress.value
  if (progress >= 100) return 'polygon(0 0, 100% 0, 100% 100%, 0 100%)'
  if (progress <= 0) return 'polygon(0 0, 0 0, 0 100%, 0 100%)'
  
  // 计算进度对应的 clip-path
  const percentage = progress / 100
  if (percentage > 0.75) {
    // 右上角到右下角
    const y = 100 - (percentage - 0.75) * 400
    return `polygon(0 0, 100% 0, 100% ${y}%, 0 100%)`
  } else if (percentage > 0.5) {
    // 左上角到右上角
    const x = (percentage - 0.5) * 400
    return `polygon(0 0, ${x}% 0, 100% 100%, 0 100%)`
  } else if (percentage > 0.25) {
    // 左下角到左上角
    const y = (percentage - 0.25) * 400
    return `polygon(0 ${y}%, 100% 0, 100% 100%, 0 100%)`
  } else {
    // 右下角到左下角
    const x = 100 - (percentage * 400)
    return `polygon(0 0, 100% 0, 100% 100%, ${x}% 100%)`
  }
})

const clearCountdown = () => {
  if (ctxData.countdownTimer) {
    clearInterval(ctxData.countdownTimer)
    ctxData.countdownTimer = null
  }
}

const startCountdown = () => {
  if (ctxData.countdownTimer) {
    clearInterval(ctxData.countdownTimer)
  }

  ctxData.countdown = 60
  const interval = 20
  const step = 60 / (60 * 1000 / interval)

  ctxData.countdownTimer = setInterval(() => {
    ctxData.countdown -= step

    if (ctxData.countdown <= 0) {
      clearInterval(ctxData.countdownTimer)
      ctxData.countdownTimer = null
      showMessage('登录页面已过期，请刷新页面重新登录', 'warning')
      setTimeout(() => {
        window.location.reload()
      }, 3000)
    }
  }, interval)
}

onBeforeUnmount(() => {
  if (ctxData.countdownTimer) {
    clearInterval(ctxData.countdownTimer)
    ctxData.countdownTimer = null
  }
})

onMounted(() => {
  const logout = localStorage.getItem('logout')
  if (logout && logout !== '') {
    try {
      const lo = JSON.parse(logout)
      showMessage(lo.message || '您已成功退出登录', lo.type || 'info')
    } catch (error) {
      console.error('解析登出信息失败:', error)
    }
    localStorage.setItem('logout', '')
  }

  loadRememberedAccount()
  getSystemInfo()
  getNonce()
  startCountdown()
})

const loadRememberedAccount = () => {
  try {
    const saved = localStorage.getItem('rememberedAccount')
    if (saved) {
      const account = JSON.parse(saved)
      ctxData.loginForm.userName = account.userName || ''
      ctxData.rememberMe = true
    }
  } catch (e) {
    console.error('加载保存的账号失败:', e)
  }
}

const getSystemInfo = async () => {
  try {
    const res = await LoginApi.getSystemInfo()
    if (res.code === '0' && res.data) {
      // 只更新版本号等信息，不更新系统标题
      const newConfigInfo = {
        ...ctxData.configInfo,
        softVer: res.data.softVer
      }
      ctxData.configInfo = newConfigInfo
      config.setConfigInfo(newConfigInfo)
    }
  } catch (error) {
    console.error('获取系统信息失败:', error)
  }
}

const getNonce = async () => {
  try {
    const res = await LoginApi.getNonce()
    if (res.code === '0' && res.data?.nonce) {
      ctxData.nonce = res.data.nonce
    } else {
      console.warn('获取nonce失败，使用本地生成')
      ctxData.nonce = Date.now().toString(36) + Math.random().toString(36).substr(2)
    }
  } catch (error) {
    console.error('获取nonce异常:', error)
    ctxData.nonce = Date.now().toString(36) + Math.random().toString(36).substr(2)
  }
}

const handleLogin = async () => {
  if (!ctxData.loginForm.userName) {
    ctxData.errorMessage = '请输入用户名'
    triggerShake()
    return
  }
  if (!ctxData.loginForm.password) {
    ctxData.errorMessage = '请输入密码'
    triggerShake()
    return
  }
  if (ctxData.loginForm.password.length < 8) {
    ctxData.errorMessage = '密码长度至少8位'
    triggerShake()
    return
  }

  ctxData.loading = true
  ctxData.errorMessage = ''

  try {
    if (!ctxData.nonce) {
      await getNonce()
    }

    let passwordToSend = ''
    if (ctxData.loginMethod === 'ldap') {
      passwordToSend = ctxData.loginForm.password
    } else {
      passwordToSend = sha256(ctxData.loginForm.password + ctxData.nonce).toString(encHex)
    }

    const loginData = {
      loginFlag: true,
      loginType: ctxData.loginMethod,
      data: {
        username: ctxData.loginForm.userName,
        password: passwordToSend,
        nonce: ctxData.nonce,
      },
      token: '',
    }

    const res = await LoginApi.login(loginData)

    if (res.code === '0') {
      await handleLoginSuccess(res)
    } else {
      handleLoginFailure(res)
      triggerShake()
      ctxData.loading = false
    }
  } catch (error) {
    handleLoginError(error)
    triggerShake()
    ctxData.loading = false
  }
}

const triggerShake = () => {
  isShaking.value = true
  setTimeout(() => {
    isShaking.value = false
  }, 500)
}

const handleLoginSuccess = async (res) => {
  clearCountdown()
  try {
    ctxData.errorMessage = ''
    isLoginSuccess.value = true

    const processedPermissions = processPermissions(res.data.permissions)

    users.setLoginInfo(
      { userName: res.data.username },
      processedPermissions,
      res.data.token
    )

    const storeData = {
      ...res.data,
      permissions: processedPermissions,
      loginTime: Date.now()
    }
    localStorage.setItem('loginInfo', JSON.stringify(storeData))

    if (ctxData.rememberMe) {
      localStorage.setItem('rememberedAccount', JSON.stringify({
        userName: ctxData.loginForm.userName,
        timestamp: Date.now()
      }))
    } else {
      localStorage.removeItem('rememberedAccount')
    }

    showMessage('登录成功')

    ctxData.loading = false
    await new Promise(resolve => setTimeout(resolve, 1000))
    await router.push('/')

  } catch (error) {
    console.error('处理登录成功数据失败:', error)
    ctxData.errorMessage = '处理用户数据失败，请稍后重试'
    ctxData.loading = false
  }
}

const processPermissions = (permissions) => {
  const perms = Array.isArray(permissions) ? [...permissions] : []

  const ensureTerminalGroup = (list) => {
    const edge = list.find(p =>
      p && (p.path === '/ruleEngine' || p.meta?.title === '边缘计算')
    )

    if (edge) {
      edge.children = edge.children || []
      const hasTerminalGroup = edge.children.some(c =>
        c && (c.path === '/terminalGroup' || c.meta?.title === '末端群控')
      )

      if (!hasTerminalGroup) {
        const terminalGroup = {
          path: '/terminalGroup',
          name: 'TerminalGroup',
          meta: { title: '末端群控', icon: 'terminal' }
        }

        const scriptIndex = edge.children.findIndex(c =>
          c && c.meta?.title === '规则脚本'
        )

        if (scriptIndex >= 0) {
          edge.children.splice(scriptIndex + 1, 0, terminalGroup)
        } else {
          edge.children.push(terminalGroup)
        }
      }
    } else {
      list.push({
        name: 'RuleEngine',
        path: '/ruleEngine',
        meta: { title: '边缘计算', icon: 'ruleEngine' },
        children: [{
          path: '/terminalGroup',
          name: 'TerminalGroup',
          meta: { title: '末端群控', icon: 'terminal' }
        }]
      })
    }

    return list
  }

  return ensureTerminalGroup(perms)
}

const handleLoginFailure = (res) => {
  ctxData.errorMessage = res.message || '登录失败，请检查用户名和密码'
  getNonce()
}

const handleLoginError = (error) => {
  console.error('登录错误:', error)

  if (error.code === 'ECONNABORTED' || error.code === 'ERR_NETWORK') {
    ctxData.errorMessage = '网络连接失败，请检查网络后重试'
  } else {
    ctxData.errorMessage = '登录异常，请稍后重试'
  }

  getNonce()
}

const handleForgotPassword = () => {
  showMessage('请联系系统管理员重置密码', 'info')
}
</script>

<style scoped>
/* ===== 容器：全屏数据背景 ===== */
.login-container {
  position: fixed;
  inset: 0;
  background: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'JetBrains Mono', monaco, monospace, sans-serif;
}

/* 登录 UI 层 */
.login-scene {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.login-panel {
  width: 580px;
  padding: 32px 60px;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 0;
  box-shadow: 0 8px 20px -5px rgba(0, 0, 0, 0.05), 0 6px 8px -6px rgba(0, 0, 0, 0.05);
  transition: box-shadow 0.15s ease, border-color 0.15s ease;
  position: relative;
  overflow: hidden;
}

.login-panel::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  border: 2px solid transparent;
  border-radius: 0;
  pointer-events: none;
  z-index: 1;
}

.login-panel::after {
  content: '';
  position: absolute;
  top: -2px;
  left: -2px;
  right: -2px;
  bottom: -2px;
  border: 2px solid #0ea5e9;
  border-radius: 0;
  pointer-events: none;
  z-index: 0;
  clip-path: var(--clip-path, polygon(0 100%, 100% 100%, 100% 100%, 0 100%));
  transition: clip-path 0.1s linear, border-color 0.3s ease;
}

.login-panel.countdown-10::after {
  border-color: #ef4444;
  box-shadow: 0 0 10px rgba(239, 68, 68, 0.5);
}

.login-panel:hover {
  box-shadow: 0 15px 30px -5px rgba(0, 0, 0, 0.1), 0 10px 15px -6px rgba(0, 0, 0, 0.1);
  border-color: #cbd5e1;
}

.panel-topbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 8px;
}

.panel-title {
  text-align: center;
  margin-bottom: 24px;
}

.title-main {
  font-size: 20px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: 0.5px;
  margin: 0;
}

.title-sub {
  font-size: 12px;
  color: #64748b;
  letter-spacing: 1.4px;
  font-family: monaco, monospace;
  margin-top: 4px;
}

.panel-header-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}

.auth-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 20px;
}

/* 其他样式参考前述代码，保持 Arco 组件自定义效果 */
.field { margin-bottom: 20px; }
.label { font-size: 12px; font-weight: 700; color: #475569; margin-bottom: 8px; }

.options {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 14px 0 10px;
}

.mode-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: rgba(34, 197, 94, 0.7);
  box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.12);
}

.mode-indicator.is-ldap {
  background: rgba(139, 92, 246, 0.9);
  box-shadow: 0 0 0 4px rgba(139, 92, 246, 0.12);
}

.panel-error {
  margin-top: 12px;
  margin-bottom: 0;
}

.compact-hint {
  margin: 0 0 10px;
  padding: 0;
  border: none;
  background: transparent;
  font-size: 11px;
}

.compact-terminal {
  justify-content: center;
  gap: 8px;
  margin-bottom: 10px;
}

.panel-copyright {
  margin-top: 14px;
  text-align: center;
}

/* 保持 Logo 工业感样式 */
.logo-icon {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 2px solid #0ea5e9;
  border-radius: 0;
  padding: 6px 12px;
  margin-right: 16px;
}
.logo-icon span { font-weight: 800; color: #0ea5e9; font-size: 16px; }
.logo-icon small { color: #64748b; font-size: 10px; margin-left: 2px; }

.version-tag {
  font-size: 10px;
  font-family: monaco, monospace;
  color: #94a3b8;
  letter-spacing: 1px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 0;
  padding: 1px 6px;
}

.top-progress {
  width: 88px;
}

.industrial-radio {
  width: 100%;
  display: flex;
}

:deep(.industrial-radio .arco-radio-button) {
  flex: 1;
  justify-content: center;
  border-radius: 0 !important;
  font-weight: 500;
  font-size: 12px;
  white-space: nowrap;
}

/* ===== 表单 ===== */
.custom-form {
  display: flex;
  flex-direction: column;
  gap: 0;
}

:deep(.arco-form-item) {
  margin-bottom: 14px;
}

:deep(.arco-input-wrapper),
:deep(.arco-input-password) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border-color: #cbd5e1 !important;
}

:deep(.arco-input-wrapper:hover),
:deep(.arco-input-password:hover) {
  border-color: #0ea5e9 !important;
}

:deep(.arco-input-wrapper.arco-input-focus),
:deep(.arco-input-password.arco-input-focus) {
  border-color: #0ea5e9 !important;
  box-shadow: none !important;
}

/* ===== LDAP 提示 ===== */
.ldap-hint {
  font-size: 11px;
  color: #64748b;
  margin-bottom: 14px;
  padding: 8px 10px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-left: 3px solid #0ea5e9;
  border-radius: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: monaco, monospace;
}

.remember-check {
  font-size: 13px;
  color: #64748b;
}

:deep(.remember-check .arco-checkbox-label) {
  color: #64748b;
  font-size: 13px;
}

.forgot-link {
  font-size: 13px;
  color: #94a3b8 !important;
}

.forgot-link:hover {
  color: #0ea5e9 !important;
}

/* ===== 错误提示 ===== */
.error-message {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: rgba(239, 68, 68, 0.04);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-left: 3px solid #ef4444;
  border-radius: 0;
  color: #ef4444;
  font-size: 13px;
  margin-bottom: 14px;
}

/* ===== 提交按钮 ===== */
.login-submit-btn { height: 50px !important; margin-top: 10px; }

:deep(.login-submit-btn.arco-btn-primary) {
  border-radius: 0 !important;
  background: #0ea5e9 !important;
  border: none !important;
  box-shadow: none !important;
  transition: all 0.2s ease !important;
}

:deep(.login-submit-btn.arco-btn-primary:hover) {
  transform: none !important;
  box-shadow: none !important;
  background: #0284c7 !important;
}

:deep(.login-submit-btn.arco-btn-primary:active) {
  transform: none !important;
  box-shadow: none !important;
}

.terminal-decorator {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-dot { width: 6px; height: 6px; background: #22c55e; animation: blink 1.5s infinite; }

@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }

.status-dot.is-ldap {
  background: #0ea5e9;
  box-shadow: 0 0 6px rgba(14, 165, 233, 0.6);
}

.terminal-code {
  font-size: 10px;
  font-family: monaco, monospace;
  color: #94a3b8;
  letter-spacing: 0.5px;
}

.copyright-text {
  font-size: 11px;
  color: #94a3b8;
  font-family: monaco, monospace;
}

/* ===== 动画 ===== */
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}

.shake-animation {
  animation: shake 0.3s ease-out both;
}

.login-card-exit {
  transform: scale(0.95) translateY(-20px);
  opacity: 0;
  transition: all 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  pointer-events: none;
}

/* ===== 响应式 ===== */
@media (max-width: 1199px) {
  .login-panel {
    width: min(580px, 90vw);
  }
}

@media (max-width: 767px) {
  .login-panel {
    width: calc(100vw - 24px);
    padding: 24px;
    border-radius: 0;
  }

  .panel-topbar,
  .options,
  .auth-row {
    flex-direction: column;
    align-items: stretch;
  }

  .panel-header-side {
    align-items: flex-start;
  }

  .login-submit-btn {
    width: 100%;
  }

  .top-progress {
    width: 82px;
  }
}
</style>


