<template>
  <div class="app-background">
    <!-- Navigation Drawer -->
    <aside 
        v-if="!isLoginPage"
        class="industrial-sidebar" 
        :class="{ 'is-collapsed': drawerRail }"
    >
        <div class="sidebar-header">
            <div class="logo-icon">
                <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
                </svg>
            </div>
            <span v-if="!drawerRail" class="logo-text">edgex</span>
        </div>
        
        <nav class="sidebar-nav">
            <router-link to="/" class="nav-item" active-class="nav-item-active" exact>
                <span class="nav-icon">
                    <icon-apps />
                </span>
                <span v-if="!drawerRail" class="nav-text">首页监控</span>
            </router-link>
            <router-link to="/channels" class="nav-item" active-class="nav-item-active">
                <span class="nav-icon">
                    <icon-link />
                </span>
                <span v-if="!drawerRail" class="nav-text">采集通道</span>
            </router-link>
            <router-link to="/edge-compute" class="nav-item" active-class="nav-item-active">
                <span class="nav-icon">
                    <icon-tool />
                </span>
                <span v-if="!drawerRail" class="nav-text">边缘计算</span>
            </router-link>
            <router-link to="/northbound" class="nav-item" active-class="nav-item-active">
                <span class="nav-icon">
                    <icon-arrow-up />
                </span>
                <span v-if="!drawerRail" class="nav-text">北向上报</span>
            </router-link>
            <router-link to="/logs" class="nav-item" active-class="nav-item-active">
                <span class="nav-icon">
                    <icon-file />
                </span>
                <span v-if="!drawerRail" class="nav-text">系统日志</span>
            </router-link>
            <router-link to="/system" class="nav-item" active-class="nav-item-active">
                <span class="nav-icon">
                    <icon-settings />
                </span>
                <span v-if="!drawerRail" class="nav-text">系统设置</span>
            </router-link>
        </nav>

        <div class="sidebar-footer">
            <button class="collapse-btn" @click="drawerRail = !drawerRail">
                <icon-arrow-left v-if="!drawerRail" :size="14" />
                <icon-arrow-right v-else :size="14" />
                <span v-if="!drawerRail">收起</span>
                <span v-else>展开</span>
            </button>
        </div>
    </aside>

    <!-- App Bar -->
    <header v-if="!isLoginPage" class="industrial-header" :class="{ 'is-collapsed': drawerRail }">
        <div class="header-title">
            <span class="title-main">边缘计算网关</span>
            <span v-if="$route.meta.title" class="title-sub">
                / {{ $route.meta.title }}
            </span>
            <span v-if="globalState.navTitle" class="title-sub">
                / {{ globalState.navTitle }}
            </span>
        </div>
        <div class="header-actions">
            <button class="theme-toggle" @click="toggleTheme" title="切换主题">
              <icon-sun-fill v-if="isDarkTheme" :size="20" />
              <icon-moon-fill v-else :size="20" />
            </button>
            <div class="user-menu" @click="toggleUserMenu" ref="userMenuRef">
                <div class="user-avatar">
                    <span>{{ userInitials }}</span>
                </div>
                <span class="user-name">{{ user.username || 'Admin' }}</span>
                <icon-caret-down class="dropdown-icon" :class="{ 'is-open': userMenuOpen }" :size="14" />
                
                <!-- Dropdown Menu -->
                <div v-if="userMenuOpen" class="dropdown-menu">
                    <div class="dropdown-item" @click.stop="openChangePassword">
                        <icon-lock :size="14" />
                        <span>修改密码</span>
                    </div>
                    <div class="dropdown-divider"></div>
                    <div class="dropdown-item text-warning" @click.stop="handleRestart">
                        <icon-refresh :size="14" />
                        <span>软件重启</span>
                    </div>
                    <div class="dropdown-item text-error" @click.stop="handleLogout">
                        <icon-arrow-right :size="14" />
                        <span>退出登录</span>
                    </div>
                </div>
            </div>
        </div>
    </header>

    <!-- Main Content -->
    <main class="main-content" :class="{ 'has-sidebar': !isLoginPage, 'is-collapsed': drawerRail }">
        <div class="page-container" v-if="!isLoginPage">
            <router-view v-slot="{ Component }">
                <transition name="fade" mode="out-in">
                    <component v-if="Component" :is="Component" :key="$route.fullPath" />
                </transition>
            </router-view>
        </div>
        <router-view v-else v-slot="{ Component }">
            <transition name="fade" mode="out-in">
                <component v-if="Component" :is="Component" :key="$route.fullPath" />
            </transition>
        </router-view>
    </main>

    <!-- Dialogs -->
    <change-password-dialog ref="changePwdRef" />

    <!-- Global Snackbar -->
    <a-notification
        v-model:visible="snackbar.show"
        :type="snackbar.color === 'error' ? 'error' : snackbar.color === 'warning' ? 'warning' : snackbar.color === 'success' ? 'success' : 'info'"
        :title="snackbar.text"
        :duration="3000"
        style="position: fixed; top: 20px; right: 20px; z-index: 1000"
    >
        <template #extra>
            <a-button type="text" size="small" @click="snackbar.show = false">关闭</a-button>
        </template>
    </a-notification>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { globalState, showMessage } from './composables/useGlobalState'
import { userStore } from '@/stores/user'
import LoginApi from '@/api/login'
import ChangePasswordDialog from '@/components/ChangePasswordDialog.vue'
import {
  IconApps, IconLink, IconSettings, IconArrowUp, 
  IconFile, IconTool, IconSunFill, IconMoonFill,
  IconUser, IconLock, IconRefresh, IconArrowRight,
  IconArrowLeft, IconCaretDown
} from '@arco-design/web-vue/es/icon'


const route = useRoute()
const router = useRouter()
const drawerRail = ref(false)
const snackbar = globalState.snackbar
const wsStatus = globalState.wsStatus
const user = userStore()
const changePwdRef = ref(null)
const isDarkTheme = ref(false)

const isLoginPage = computed(() => {
    return route.path === '/login'
})

const userInitials = computed(() => {
    return (user.username || 'A').charAt(0).toUpperCase()
})

// Theme toggle
const toggleTheme = () => {
    isDarkTheme.value = !isDarkTheme.value
    localStorage.setItem('theme', isDarkTheme.value ? 'dark' : 'light')
    document.body.classList.toggle('dark-theme', isDarkTheme.value)
    document.documentElement.classList.toggle('dark-theme', isDarkTheme.value)
    document.documentElement.classList.toggle('arco-theme-dark', isDarkTheme.value)
}

// User menu
const userMenuOpen = ref(false)
const userMenuRef = ref(null)

const toggleUserMenu = () => {
    userMenuOpen.value = !userMenuOpen.value
}

// Close menu when clicking outside
const handleClickOutside = (event) => {
    if (userMenuRef.value && !userMenuRef.value.contains(event.target)) {
        userMenuOpen.value = false
    }
}

onMounted(() => {
    document.addEventListener('click', handleClickOutside)
    // Load theme
    const savedTheme = localStorage.getItem('theme')
    if (savedTheme === 'dark') {
        isDarkTheme.value = true
        document.body.classList.add('dark-theme')
        document.documentElement.classList.add('dark-theme')
        document.documentElement.classList.add('arco-theme-dark')
    }
    // Restore user info from localStorage if not present
    if (!user.username) {
        try {
            const loginInfo = localStorage.getItem('loginInfo')
            if (loginInfo) {
                const parsed = JSON.parse(loginInfo)
                // parsed is the storeData from Login.vue, which has 'username' (lowercase)
                if (parsed && parsed.username) {
                    user.setLoginInfo({ userName: parsed.username }, parsed.permissions || [], parsed.token || '')
                }
            }
        } catch (e) {
            console.error('Failed to restore user info', e)
        }
    }
    // Test showMessage function
    console.log('Testing showMessage function')
    showMessage('测试提示信息', 'success')
})

onUnmounted(() => {
    document.removeEventListener('click', handleClickOutside)
})

const openChangePassword = () => {
    changePwdRef.value?.open()
}

const handleLogout = async () => {
    try {
        await LoginApi.logout()
    } catch (e) {
        console.error(e)
    }
    localStorage.removeItem('loginInfo')
    // Keep 'rememberedAccount'
    user.setLoginInfo({}, [], '')
    router.push('/login')
    showMessage('已退出登录')
}

const handleRestart = () => {
    if (confirm('确定要重启系统吗？服务将暂时不可用。')) {
        LoginApi.restartSystem().then(() => {
            showMessage('系统正在重启...', 'warning')
            // Wait a bit then reload to show login page or error (since server is down)
            setTimeout(() => {
                window.location.reload()
            }, 5000)
        }).catch(e => {
            showMessage('重启指令发送失败: ' + e.message, 'error')
        })
    }
}
</script>

<style>
.industrial-menu {
    background: rgba(255, 255, 255, 0.95) !important;
    border: 1px solid #cbd5e1 !important;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1) !important;
}

:root {
    /* Fonts */
    --font-sans: 'JetBrains Mono', ui-sans-serif, system-ui, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
    --font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    
    /* Colors */
    --color-gray-50: #f8fafc;
    --color-gray-900: #1e293b;
    --color-blue-50: #f8fafc;
    --color-purple-50: #f8fafc;
    
    /* Spacing & Radius */
    --spacing: 0.25rem;
    --radius-2xl: 2px;
    
    /* Animations */
    --animate-float: float 6s ease-in-out infinite;
}

@keyframes float {
    0% { transform: translateY(0px); }
    50% { transform: translateY(-10px); }
    100% { transform: translateY(0px); }
}

@keyframes blink {
    0% { opacity: 1; }
    50% { opacity: 0.5; }
    100% { opacity: 1; }
}

.blink {
    animation: blink 1s linear infinite;
}

body {
    font-family: var(--font-sans);
    margin: 0;
    color: var(--color-gray-900);
}

.app-background {
    background: #f8fafc;
    background-size: cover;
    background-attachment: fixed;
    min-height: 100vh;
}

/* Dark theme */
.dark-theme {
    --color-gray-50: #1e1e1e;
    --color-gray-900: #f8fafc;
}

.dark-theme .app-background {
    background: #1e1e1e;
}

.dark-theme .industrial-sidebar {
    background: rgba(30, 30, 30, 0.98);
    border-color: #333;
}

.dark-theme .industrial-header {
    background: rgba(30, 30, 30, 0.98);
    border-color: #333;
}

.dark-theme .sidebar-header {
    border-color: #333;
}

.dark-theme .sidebar-nav .nav-item {
    color: #ccc;
}

.dark-theme .sidebar-nav .nav-item:hover {
    background: #333;
    color: #0ea5e9;
}

.dark-theme .sidebar-nav .nav-item-active {
    background: #333;
    color: #0ea5e9;
    border-left-color: #0ea5e9;
}

.dark-theme .sidebar-footer {
    border-color: #333;
    background: rgba(30, 30, 30, 0.98);
}

.dark-theme .status-text {
    color: #999;
}

.dark-theme .version-info {
    color: #666;
}

.dark-theme .collapse-btn {
    color: #999;
}

.dark-theme .collapse-btn:hover {
    color: #0ea5e9;
    background: #333;
}

.dark-theme .industrial-card {
    background: rgba(30, 30, 30, 0.85) !important;
    border-color: #333 !important;
    color: #f8fafc !important;
}



.dark-theme .user-name {
    color: #f8fafc;
}

.dark-theme .dropdown-menu {
    background: rgba(30, 30, 30, 0.98);
    border-color: #333;
}

.dark-theme .dropdown-item {
    color: #ccc;
}

.dark-theme .dropdown-item:hover {
    color: #0ea5e9;
}

.dark-theme .industrial-table :deep(.arco-table-th) {
    background: #333;
    border-color: #444;
    color: #f8fafc;
}

.dark-theme .industrial-table :deep(.arco-table-td) {
    border-color: #444;
    color: #f8fafc;
}

.dark-theme .industrial-table :deep(.arco-table-tr:hover .arco-table-td) {
    background: #333;
}

.dark-theme .rect-input {
    background: #333;
    border-color: #444;
    color: #f8fafc;
}

.dark-theme .rect-input:focus {
    border-color: #0ea5e9;
}

.dark-theme .dashboard-container,
.dark-theme .edge-compute-container,
.dark-theme .stats-grid,
.dark-theme .section,
.dark-theme .channel-card,
.dark-theme .northbound-card,
.dark-theme .edge-compute-card,
.dark-theme .empty-card,
.dark-theme .metrics-card {
    background: #1f2937 !important;
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .dashboard-title,
.dark-theme .section-title,
.dark-theme .card-title,
.dark-theme .toolbar-title,
.dark-theme .status-text,
.dark-theme .title-sub,
.dark-theme .title-main,
.dark-theme .metric-label,
.dark-theme .metric-value,
.dark-theme .sub-item,
.dark-theme .ip-group-label,
.dark-theme .channel-meta,
.dark-theme .channel-name,
.dark-theme .status-badge,
.dark-theme .quality-score {
    color: #f8fafc !important;
}

.dark-theme .btn-primary,
.dark-theme .btn-outline,
.dark-theme .theme-toggle,
.dark-theme .collapse-btn,
.dark-theme .user-menu {
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .btn-primary {
    background: #2563eb !important;
}

.dark-theme .btn-outline {
    color: #60a5fa !important;
    border-color: #60a5fa !important;
}

/* 强制深色模式覆盖：Arco 组件与自定义卡片 */
.dark-theme .arco-card,
.dark-theme .arco-card-body,
.dark-theme .arco-card-header,
.dark-theme .arco-form,
.dark-theme .arco-form-item,
.dark-theme .arco-form-item-label,
.dark-theme .arco-form-item-content,
.dark-theme .arco-input,
.dark-theme .arco-input-wrapper,
.dark-theme .arco-select,
.dark-theme .arco-select-dropdown,
.dark-theme .arco-table,
.dark-theme .arco-table-th,
.dark-theme .arco-table-td,
.dark-theme .arco-tabs,
.dark-theme .arco-tabs-nav,
.dark-theme .arco-tabs-tab {
    background: #1f2937 !important;
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .arco-btn,
.dark-theme .arco-button,
.dark-theme .arco-tag,
.dark-theme .arco-badge,
.dark-theme .arco-badge-status,
.dark-theme .arco-badge-dot,
.dark-theme .arco-dropdown-menu {
    background: #1f2937 !important;
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .arco-badge .arco-badge-text,
.dark-theme .arco-badge-status,
.dark-theme .arco-badge-dot {
    color: #f8fafc !important;
}

.dark-theme .arco-input,
.dark-theme .arco-select,
.dark-theme .arco-textarea,
.dark-theme .arco-datepicker,
.dark-theme .arco-time-picker {
    background: #1f2937 !important;
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .arco-form-item-label,
.dark-theme .arco-form-item-content,
.dark-theme .arco-table-th,
.dark-theme .arco-table-td,
.dark-theme .arco-tabs-tab {
    color: #f8fafc !important;
}

.dark-theme .arco-table,
.dark-theme .arco-table-wrapper,
.dark-theme .arco-table thead,
.dark-theme .arco-table tfoot,
.dark-theme .arco-table th,
.dark-theme .arco-table td,
.dark-theme .arco-table-pagination {
    background: #0f172a !important;
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .arco-table-cells th,
.dark-theme .arco-table-cells td {
    background: #111827 !important;
    color: #f8fafc !important;
}

.dark-theme .arco-table-pagination,
.dark-theme .arco-table-pagination .arco-pagination,
.dark-theme .arco-table-pagination .arco-pagination-total,
.dark-theme .arco-table-pagination .arco-pagination-list,
.dark-theme .arco-table-pagination .arco-pagination-item,
.dark-theme .arco-table-pagination .arco-pagination-options,
.dark-theme .arco-select-view,
.dark-theme .arco-select-view-single,
.dark-theme .arco-select-view-value,
.dark-theme .arco-pagination {
    background: #0f172a !important;
    color: #f8fafc !important;
    border-color: #334155 !important;
}

.dark-theme .arco-table-pagination .arco-pagination-item,
.dark-theme .arco-table-pagination .arco-pagination-item-active,
.dark-theme .arco-table-pagination .arco-pagination-item-disabled,
.dark-theme .arco-pagination-item,
.dark-theme .arco-pagination-item-active,
.dark-theme .arco-pagination-item-disabled {
    background: #1f2937 !important;
    color: #f8fafc !important;
}

.dark-theme .arco-table-pagination,
.dark-theme .arco-pagination,
.dark-theme .arco-select-view,
.dark-theme .arco-select-view-single,
.dark-theme .arco-select-view-value,
.dark-theme .arco-select-view-input-hidden,
.dark-theme .arco-select-view-input {
    background: #0f172a !important;
    color: #f8fafc !important;
}

.dark-theme .arco-table-pagination *,
.dark-theme .arco-pagination *,
.dark-theme .arco-select-view * {
    background: transparent !important;
    color: #f8fafc !important;
}


.dark-theme .arco-pagination,
.dark-theme .arco-pagination .arco-pagination-total,
.dark-theme .arco-pagination .arco-pagination-jumper,
.dark-theme .arco-pagination .arco-select,
.dark-theme .arco-pagination .arco-select-selection,
.dark-theme .arco-pagination .arco-select-dropdown,
.dark-theme .arco-pagination .arco-select-selection-value {
    background: #0f172a !important;
    color: #f8fafc !important;
    border-color: #334155 !important;
}

.dark-theme .arco-select-dropdown .arco-select-item,
.dark-theme .arco-select-dropdown .arco-select-item-label {
    background: #1f2937 !important;
    color: #f8fafc !important;
}



/* 北向数据上报、卡片按钮、状态补丁 */
.dark-theme .northbound-card,
.dark-theme .northbound-card * {
    color: #f8fafc !important;
}

.dark-theme .northbound-card .status-badge,
.dark-theme .northbound-card .arco-tag,
.dark-theme .northbound-card .arco-badge,
.dark-theme .northbound-card .arco-button,
.dark-theme .northbound-card .btn-outline,
.dark-theme .northbound-card .btn-primary {
    color: #f8fafc !important;
    border-color: #334155 !important;
    background: #1f2937 !important;
}

/* Channel list / system overview / collection channels dark mode */
.dark-theme .channel-list-container,
.dark-theme .channel-header,
.dark-theme .title-text,
.dark-theme .title-subtitle,
.dark-theme .header-actions,
.dark-theme .minimal-line-card,
.dark-theme .card-title-content,
.dark-theme .protocol-tag,
.dark-theme .name-text,
.dark-theme .card-info-body,
.dark-theme .info-item,
.dark-theme .info-item .label,
.dark-theme .info-item .value,
.dark-theme .arco-table-small .arco-table-th,
.dark-theme .arco-table-small .arco-table-td,
.dark-theme .arco-table-small .arco-table-tr,
.dark-theme .config-section,
.dark-theme .section-header,
.dark-theme .section-title,
.dark-theme .stats-grid,
.dark-theme .stat-card,
.dark-theme .channel-card,
.dark-theme .channel-header,
.dark-theme .channel-info {
    background: #0f172a !important;
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .channel-list-container {
    background: #0b1223 !important;
}

.dark-theme .channel-header,
.dark-theme .stats-grid,
.dark-theme .stat-card,
.dark-theme .channel-card,
.dark-theme .channel-info {
    background: #111827 !important;
}

.dark-theme .stat-card:hover,
.dark-theme .minimal-line-card:hover {
    border-color: #3b82f6 !important;
    box-shadow: 0 0 14px rgba(14, 165, 233, 0.4) !important;
}

.dark-theme .protocol-tag,
.dark-theme .name-text,
.dark-theme .info-item .label,
.dark-theme .info-item .value,
.dark-theme .title-text,
.dark-theme .title-subtitle,
.dark-theme .section-title,
.dark-theme .stat-label,
.dark-theme .stat-value,
.dark-theme .status-text,
.dark-theme .quality-score {
    color: #f8fafc !important;
}

.dark-theme .arco-table-small .arco-table-th {
    background: #1f2937 !important;
    color: #f8fafc !important;
}

.dark-theme .arco-table-small .arco-table-td {
    background: #111827 !important;
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .arco-table-small .arco-table-tr:hover,
.dark-theme .arco-table-small .arco-table-tr.arco-table-tr-selected {
    background: #1e2938 !important;
}


.dark-theme .northbound-card .arco-tag-disabled,
.dark-theme .northbound-card .arco-tag-plain {
    color: #f8fafc !important;
    background: #334155 !important;
    border-color: #4b5563 !important;
}

.dark-theme .stats-grid,
.dark-theme .stat-card,
.dark-theme .section-header,
.dark-theme .channel-card,
.dark-theme .channel-header,
.dark-theme .channel-info {
    background: #1e293b !important;
    border-color: #334155 !important;
    color: #f8fafc !important;
}

.dark-theme .stat-card {
    box-shadow: none !important;
}

.dark-theme .stat-bar,
.dark-theme .channel-stats,
.dark-theme .channel-metrics,
.dark-theme .quality-bar-container,
.dark-theme .edge-stats {
    background: #23303f !important;
    border-color: #334155 !important;
}

.dark-theme .stat-label,
.dark-theme .stat-value,
.dark-theme .channel-name,
.dark-theme .channel-meta,
.dark-theme .status-text,
.dark-theme .quality-score,
.dark-theme .section-title,
.dark-theme .dashboard-title {
    color: #f8fafc !important;
}

.industrial-card {
    background: rgba(255, 255, 255, 0.85) !important;
    border: 1px solid #cbd5e1 !important;
    border-radius: 2px !important;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1) !important;
    transition: all 0.15s ease;
}

.industrial-card::before {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    height: 2px;
    width: 100%;
    background: #0ea5e9;
    opacity: 0.3;
}

.industrial-card:hover {
    border-color: #0ea5e9;
    box-shadow: 0 0 0 1px rgba(14, 165, 233, 0.1);
}

/* Page Transition */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.industrial-app-bar {
    background: rgba(255, 255, 255, 0.85) !important;
    border-bottom: 1px solid #cbd5e1 !important;
}

/* 工业风格侧边栏 - 按照规范重构 */
.industrial-sidebar {
    position: fixed;
    top: 0;
    left: 0;
    height: 100vh;
    width: 144px;
    background: white;
    border-right: 1px solid #e2e8f0;
    display: flex;
    flex-direction: column;
    z-index: 100;
    transition: width 0.2s ease;
    outline: none;
}

.industrial-sidebar.is-collapsed {
    width: 56px;
}

/* 侧边栏头部 */
.sidebar-header {
    display: flex;
    align-items: center;
    padding: 0 16px;
    height: 56px;
    border-bottom: 1px solid #e2e8f0;
    flex-shrink: 0;
    box-sizing: border-box;
}

.logo-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    color: #0ea5e9;
    flex-shrink: 0;
}

.logo-text {
    font-size: 16px;
    font-weight: 600;
    color: #0ea5e9;
    font-family: 'JetBrains Mono', monospace;
    margin-left: 12px;
    white-space: nowrap;
}

/* 导航区域 */
.sidebar-nav {
    flex: 1;
    padding: 8px 0;
    overflow-y: auto;
}

.nav-item {
    display: flex;
    align-items: center;
    padding: 8px 16px;
    color: #334155;
    text-decoration: none;
    transition: all 0.15s ease;
    border-left: 2px solid transparent;
    min-height: 40px;
    position: relative;
    outline: none;
    border-radius: 0;
}

.nav-item:hover {
    background: #f8fafc;
    color: #334155;
}

.nav-item-active {
    background: #f8fafc;
    color: #0f172a;
    border-left: 2px solid #0ea5e9;
    font-weight: 500;
}

.nav-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    color: #64748b;
    flex-shrink: 0;
}

.nav-item:hover .nav-icon,
.nav-item-active .nav-icon {
    color: #0ea5e9;
}

.nav-text {
    font-size: 14px;
    font-weight: 400;
    margin-left: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

/* 侧边栏底部 - 状态区 */
.sidebar-footer {
    padding: 16px;
    border-top: 1px solid #e2e8f0;
    flex-shrink: 0;
    background: white;
}

.sidebar-status {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 0;
}

.status-indicator {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #22c55e;
    animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
    0%, 100% {
        opacity: 1;
    }
    50% {
        opacity: 0.5;
    }
}

.status-text {
    font-size: 12px;
    color: #64748b;
}

.version-info {
    font-size: 11px;
    color: #94a3b8;
    margin-top: 4px;
}

.collapse-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    padding: 8px;
    color: #64748b;
    font-size: 12px;
    background: transparent;
    border: none;
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: inherit;
    border-radius: 0;
    margin-top: 8px;
}

.collapse-btn:hover {
    color: #0ea5e9;
    background: #f8fafc;
}

.collapse-btn svg {
    margin-right: 6px;
    flex-shrink: 0;
    width: 14px;
    height: 14px;
}

/* 顶部导航栏 */
.industrial-header {
    position: fixed;
    top: 0;
    right: 0;
    left: 144px;
    height: 56px;
    background: rgba(255, 255, 255, 0.98);
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px 0 0;
    z-index: 99;
    transition: left 0.2s ease;
    outline: none;
    box-sizing: border-box;
}

.industrial-header.is-collapsed {
    left: 56px;
}

.industrial-header::before {
    content: "";
    position: absolute;
    bottom: 0;
    left: 0;
    width: 100%;
    height: 1px;
    background: linear-gradient(to right, #0ea5e9 0%, #0ea5e9 100%);
    opacity: 0.3;
}

.industrial-header::after {
    content: "";
    position: absolute;
    bottom: 0;
    left: 0;
    width: 100%;
    height: 1px;
    background: linear-gradient(to right, transparent 0%, transparent 100%);
    opacity: 0.5;
}

.header-title {
    display: flex;
    align-items: center;
    font-family: 'JetBrains Mono', monospace;
}

.title-main {
    font-size: 16px;
    font-weight: 600;
    color: #0ea5e9;
}

.title-sub {
    font-size: 13px;
    font-weight: 400;
    color: #64748b;
    margin-left: 6px;
}

.header-actions {
    display: flex;
    align-items: center;
}

.theme-toggle {
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 8px;
    border-radius: 0;
    color: #64748b;
    transition: all 0.15s ease;
    margin-right: 8px;
}

.theme-toggle:hover {
    color: #0ea5e9;
    background: rgba(14, 165, 233, 0.05);
}

.user-menu {
    position: relative;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    cursor: pointer;
    border-radius: 0;
    transition: background 0.15s ease;
}

.user-menu:hover {
    background: rgba(14, 165, 233, 0.05);
}

.user-avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: #0ea5e9;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-size: 12px;
    font-weight: 600;
}

.user-name {
    font-size: 13px;
    font-weight: 500;
    color: #1e293b;
}

.dropdown-icon {
    color: #64748b;
    transition: transform 0.2s ease;
    width: 14px;
    height: 14px;
}

.dropdown-icon.is-open {
    transform: rotate(180deg);
}

.dropdown-menu {
    position: absolute;
    top: calc(100% + 6px);
    right: 0;
    min-width: 140px;
    background: rgba(255, 255, 255, 0.98);
    border: 1px solid #cbd5e1;
    border-radius: 0;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    z-index: 1000;
    padding: 4px 0;
}

.dropdown-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    color: #475569;
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s ease;
}

.dropdown-item:hover {
    background: rgba(14, 165, 233, 0.05);
    color: #0ea5e9;
}

.dropdown-item:hover svg {
    color: #0ea5e9;
}

.dropdown-item.text-warning {
    color: #f59e0b;
}

.dropdown-item.text-warning:hover {
    background: rgba(245, 158, 11, 0.05);
}

.dropdown-item.text-error {
    color: #ef4444;
}

.dropdown-item.text-error:hover {
    background: rgba(239, 68, 68, 0.05);
}

.dropdown-divider {
    height: 1px;
    background: #e2e8f0;
    margin: 4px 0;
}

/* 主内容区域 */
.main-content {
    flex: 1;
    min-height: 100vh;
    padding-top: 56px;
    transition: margin-left 0.2s ease;
}

.main-content.has-sidebar {
    margin-left: 144px;
}

.main-content.has-sidebar.is-collapsed {
    margin-left: 56px;
}

.page-container {
    min-height: calc(100vh - 56px);
}

.channel-icon {
    background: rgba(255, 255, 255, 0.6);
    border: 1px solid #cbd5e1;
    border-radius: 0;
    padding: 12px;
    display: inline-flex;
    margin-bottom: 12px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

/* ===== 工业级表单组件样式重构 ===== */
/* 1. 强制所有交互组件取消 Ring 偏移，改用内边框 */
*:focus {
    outline: none !important;
}

:focus-visible {
    --tw-ring-offset-width: 0px !important;
    --tw-ring-width: 0px !important;
    --tw-ring-offset-color: transparent !important;
    --tw-ring-color: transparent !important;
    box-shadow: none !important;
}

/* 2. 按钮样式重写 - 直角设计，无圆角，无阴影 */
.arco-btn,
.arco-button {
    border-radius: 0 !important;
    box-shadow: none !important;
}

.arco-btn:focus,
.arco-btn:focus-visible,
.arco-button:focus,
.arco-button:focus-visible {
    box-shadow: none !important;
    outline: none !important;
}

/* 默认按钮高度：32px */
.arco-btn-size-medium {
    height: 32px !important;
    line-height: 30px !important;
    font-size: 13px !important;
}

/* small 按钮高度：28px */
.arco-btn-size-small {
    height: 28px !important;
    line-height: 26px !important;
    font-size: 12px !important;
}

/* large 按钮高度：40px */
.arco-btn-size-large {
    height: 40px !important;
    line-height: 38px !important;
    font-size: 14px !important;
}

/* Primary 按钮样式 */
.arco-btn-primary,
.arco-button[type="primary"] {
    background: #0ea5e9 !important;
    border-color: #0ea5e9 !important;
    color: white !important;
    box-shadow: none !important;
}

.arco-btn-primary:hover,
.arco-button[type="primary"]:hover {
    background: #0284c7 !important;
    border-color: #0284c7 !important;
    box-shadow: none !important;
}

.arco-btn-primary:active,
.arco-button[type="primary"]:active {
    background: #0369a1 !important;
    border-color: #0369a1 !important;
    box-shadow: none !important;
}

/* Outline 按钮样式 */
.arco-btn-outline,
.arco-button[type="outline"] {
    background: transparent !important;
    border-color: #cbd5e1 !important;
    color: #475569 !important;
    box-shadow: none !important;
}

.arco-btn-outline:hover,
.arco-button[type="outline"]:hover {
    border-color: #0ea5e9 !important;
    color: #0ea5e9 !important;
    background: rgba(14, 165, 233, 0.05) !important;
    box-shadow: none !important;
}

/* Text 按钮样式 */
.arco-btn-text {
    background: transparent !important;
    border-color: transparent !important;
    color: #64748b !important;
    box-shadow: none !important;
}

.arco-btn-text:hover {
    color: #0ea5e9 !important;
    background: rgba(14, 165, 233, 0.05) !important;
}

/* 3. Input 组件样式重写 - 直角设计，焦点边框变色 */
.arco-input-wrapper,
.arco-input,
.arco-textarea,
.arco-input-password {
    border-radius: 0 !important;
    box-shadow: none !important;
    background-color: #ffffff;
    border-color: #cbd5e1 !important;
}

/* Input 焦点状态 */
.arco-input-wrapper.arco-input-focus,
.arco-input:focus,
.arco-textarea:focus,
.arco-input-password.arco-input-focus {
    border-color: #0ea5e9 !important;
    box-shadow: none !important;
    outline: none !important;
}

/* Input 悬停状态 */
.arco-input-wrapper:hover,
.arco-input:hover,
.arco-textarea:hover,
.arco-input-password:hover {
    border-color: #94a3b8 !important;
}

.arco-input-wrapper.arco-input-focus:hover,
.arco-input:focus:hover,
.arco-textarea:focus:hover,
.arco-input-password.arco-input-focus:hover {
    border-color: #0ea5e9 !important;
}

/* Input 默认高度：32px */
.arco-input-size-medium {
    height: 32px !important;
}

.arco-input-size-medium .arco-input {
    height: 32px !important;
    line-height: 30px !important;
}

/* Input small 高度：28px */
.arco-input-size-small {
    height: 28px !important;
}

.arco-input-size-small .arco-input {
    height: 28px !important;
    line-height: 26px !important;
}

/* Input large 高度：40px */
.arco-input-size-large {
    height: 40px !important;
}

.arco-input-size-large .arco-input {
    height: 40px !important;
    line-height: 38px !important;
}

/* 4. Select 组件样式重写 */
.arco-select-view,
.arco-select-view-single {
    border-radius: 0 !important;
    box-shadow: none !important;
    background-color: #ffffff;
    border-color: #cbd5e1 !important;
}

.arco-select-view.arco-select-view-focus,
.arco-select-view-single.arco-select-view-focus {
    border-color: #0ea5e9 !important;
    box-shadow: none !important;
    outline: none !important;
}

.arco-select-view:hover,
.arco-select-view-single:hover {
    border-color: #94a3b8 !important;
}

/* Select 下拉选项 */
.arco-select-dropdown {
    border-radius: 0 !important;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    border: 1px solid #e2e8f0;
}

.arco-select-option {
    border-radius: 0 !important;
}

.arco-select-option:hover {
    background: rgba(14, 165, 233, 0.05);
}

.arco-select-option-selected {
    background: rgba(14, 165, 233, 0.1);
    color: #0ea5e9;
}

/* 5. Textarea 组件 */
.arco-textarea {
    border-radius: 0 !important;
    box-shadow: none !important;
    background-color: #ffffff;
    border-color: #cbd5e1 !important;
}

.arco-textarea:focus {
    border-color: #0ea5e9 !important;
    box-shadow: none !important;
    outline: none !important;
}

/* 6. Checkbox 和 Radio 组件 */
.arco-checkbox,
.arco-radio {
    border-radius: 0 !important;
}

.arco-checkbox-checked .arco-checkbox-icon {
    background: #0ea5e9 !important;
    border-color: #0ea5e9 !important;
}

.arco-radio-checked .arco-radio-icon {
    border-color: #0ea5e9 !important;
}

.arco-radio-checked .arco-radio-icon::after {
    background: #0ea5e9 !important;
}

/* 7. Switch 组件 */
.arco-switch {
    border-radius: 0 !important;
}

.arco-switch-checked {
    background: #0ea5e9 !important;
}

/* 8. Form Item 样式优化 */
.arco-form-item {
    margin-bottom: 16px;
}

.arco-form-item-label-col {
    padding-right: 12px;
}

.arco-form-item-label {
    font-weight: 500;
    color: #475569;
    font-size: 13px;
    line-height: 32px;
}

.arco-form-item-message {
    font-size: 12px;
    color: #ef4444;
}

/* 9. Modal/Dialog 组件 */
.arco-modal {
    border-radius: 0 !important;
    box-shadow: 0 8px 20px rgba(0, 0, 0, 0.1);
}

.arco-modal-header {
    border-bottom: 1px solid #e2e8f0;
    padding: 16px 20px;
    background: #f8fafc;
}

.arco-modal-body {
    padding: 20px;
}

.arco-modal-footer {
    border-top: 1px solid #e2e8f0;
    padding: 16px 20px;
}

/* 10. Tabs 组件 */
.arco-tabs-nav {
    border-bottom: 1px solid #e2e8f0;
}

.arco-tabs-tab {
    border-radius: 0 !important;
    margin-right: 0 !important;
    padding: 0 16px;
    height: 40px;
    line-height: 39px;
}

.arco-tabs-tab-active {
    border-bottom: 2px solid #0ea5e9;
    color: #0ea5e9;
}

/* 11. InputNumber 组件 */
.arco-input-number {
    border-radius: 0 !important;
    box-shadow: none !important;
    background-color: #ffffff;
    border-color: #cbd5e1 !important;
}

.arco-input-number:focus,
.arco-input-number.arco-input-focus {
    border-color: #0ea5e9 !important;
    box-shadow: none !important;
}

/* 12. DatePicker 和 TimePicker 组件 */
.arco-picker,
.arco-timepicker {
    border-radius: 0 !important;
    box-shadow: none !important;
    background-color: #ffffff;
    border-color: #cbd5e1 !important;
}

.arco-picker:focus,
.arco-picker.arco-picker-focus,
.arco-timepicker:focus,
.arco-timepicker.arco-input-focus {
    border-color: #0ea5e9 !important;
    box-shadow: none !important;
}

/* 13. Dropdown 组件 */
.arco-dropdown {
    border-radius: 0 !important;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    border: 1px solid #e2e8f0;
}

.arco-dropdown-option {
    border-radius: 0 !important;
}

.arco-dropdown-option:hover {
    background: rgba(14, 165, 233, 0.05);
}

/* 14. Tag 组件 */
.arco-tag {
    border-radius: 0 !important;
}

/* 15. Tooltip 组件 */
.arco-tooltip {
    border-radius: 0 !important;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* 16. Upload 组件 */
.arco-upload-list-item {
    border-radius: 0 !important;
}

.arco-upload-list-picture-card {
    border-radius: 0 !important;
}

/* 17. Transfer 组件 */
.arco-transfer-list {
    border-radius: 0 !important;
}

/* 18. Tree 组件 */
.arco-tree-node-title:hover {
    background: rgba(14, 165, 233, 0.05);
}

.arco-tree-node-selected .arco-tree-node-title {
    background: rgba(14, 165, 233, 0.1);
    color: #0ea5e9;
}

/* 19. Card 组件 */
.arco-card {
    border-radius: 0 !important;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1) !important;
}

.arco-card-header {
    border-bottom: 1px solid #e2e8f0;
    padding: 16px 20px;
}

.arco-card-body {
    padding: 20px;
}

/* 20. Table 组件相关表单输入 */
.arco-table .arco-input-wrapper,
.arco-table .arco-select-view {
    border-radius: 0 !important;
}

/* 工业级滚动条 */
::-webkit-scrollbar {
    width: 6px;
    height: 6px;
}

::-webkit-scrollbar-track {
    background: transparent;
}

::-webkit-scrollbar-thumb {
    background: #CBD5E1;
    border-radius: 0;
}

::-webkit-scrollbar-thumb:hover {
    background: #94A3B8;
}

/* Firefox 滚动条 */
* {
    scrollbar-width: thin;
    scrollbar-color: #CBD5E1 transparent;
}
</style>
