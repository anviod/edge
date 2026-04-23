<template>
  <a-card class="system-maintenance-card mb-4">
    <a-card-header>
      <div class="card-title">
        <icon-settings :size="18" class="title-icon" />
        系统维护
      </div>
    </a-card-header>
    <a-card-body>
      <a-alert type="success" class="status-alert mb-8">
        <template #icon>
          <icon-check-circle-fill />
        </template>
        系统运行正常
      </a-alert>
      <a-row :gutter="20">
        <a-col :span="12" class="metrics-col">
          <a-card class="maintenance-card restart-card" @click="showRestartConfirm">
            <div class="card-icon">
              <icon-refresh :size="32" />
            </div>
            <div class="card-content">
              <div class="card-title-text">重启系统</div>
              <div class="card-description">立即重启 Edge Gateway 硬件终端</div>
            </div>
            <div class="card-actions">
              <a-button type="outline" status="warning" size="small" class="action-btn">
                <template #icon>
                  <icon-refresh />
                </template>
                执行重启
              </a-button>
            </div>
          </a-card>
        </a-col>
        <a-col :span="12" class="metrics-col">
          <a-card class="maintenance-card reset-card" @click="showResetConfirm">
            <div class="card-icon">
              <icon-delete :size="32" />
            </div>
            <div class="card-content">
              <div class="card-title-text">恢复出厂设置</div>
              <div class="card-description">清除所有本地配置并恢复出厂镜像</div>
            </div>
            <div class="card-actions">
              <a-button type="outline" status="danger" size="small" class="action-btn">
                <template #icon>
                  <icon-delete />
                </template>
                执行清除
              </a-button>
            </div>
          </a-card>
        </a-col>
      </a-row>
    </a-card-body>
  </a-card>

  <!-- 重启确认弹窗 -->
  <a-modal
    v-model:visible="restartModalVisible"
    title="重启系统"
    ok-text="确认重启"
    cancel-text="取消"
    status="warning"
    @ok="handleRestart"
  >
    <div class="modal-content">
      <p class="modal-message">确定要重启系统吗？</p>
      <p class="modal-warning">服务将暂时不可用，重启过程可能需要几分钟时间。</p>
    </div>
  </a-modal>

  <!-- 恢复出厂设置确认弹窗 -->
  <a-modal
    v-model:visible="resetModalVisible"
    title="恢复出厂设置"
    ok-text="确认恢复"
    cancel-text="取消"
    status="danger"
    @ok="handleReset"
  >
    <div class="modal-content">
      <p class="modal-message">确定要恢复出厂设置吗？</p>
      <p class="modal-warning">此操作将清除所有配置且无法撤销。系统将恢复到初始状态。</p>
    </div>
  </a-modal>
</template>

<script setup>
import { ref } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconSettings,
  IconCheckCircleFill,
  IconRefresh,
  IconDelete
} from '@arco-design/web-vue/es/icon'

const restartModalVisible = ref(false)
const resetModalVisible = ref(false)

const showRestartConfirm = () => {
  restartModalVisible.value = true
}

const showResetConfirm = () => {
  resetModalVisible.value = true
}

const handleRestart = () => {
  Message.loading({
    content: '正在重启系统...',
    duration: 0
  })
  // 这里可以添加实际的重启逻辑
  setTimeout(() => {
    Message.success('系统重启指令已发送')
  }, 2000)
  restartModalVisible.value = false
}

const handleReset = () => {
  Message.loading({
    content: '正在恢复出厂设置...',
    duration: 0
  })
  // 这里可以添加实际的重置逻辑
  setTimeout(() => {
    Message.success('出厂设置恢复指令已发送')
  }, 2000)
  resetModalVisible.value = false
}
</script>

<style scoped>
.modal-content {
  padding: 16px 0;
}

.modal-message {
  font-size: 16px;
  font-weight: 500;
  color: #111827;
  margin-bottom: 12px;
}

.modal-warning {
  font-size: 14px;
  color: #6b7280;
  line-height: 1.5;
}

.dark-theme .modal-message {
  color: #f9fafb;
}

.dark-theme .modal-warning {
  color: #d1d5db;
}

/* Original styles */
.system-maintenance-card {
  border-radius: 0;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  border: 1px solid #e5e7eb;
  overflow: hidden;
  transition: all 0.3s ease;
}

.system-maintenance-card:hover {
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #111827;
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}

.title-icon {
  color: #3b82f6;
}

.status-alert {
  border-radius: 0;
  border: 1px solid #10b981;
  background: linear-gradient(135deg, #ecfdf5 0%, #f0fdf4 100%);
}

.status-alert :deep(.arco-alert-content) {
  font-weight: 500;
}

.metrics-col {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.maintenance-card {
  border-radius: 0;
  border: 1px solid #e5e7eb;
  background: linear-gradient(135deg, #ffffff 0%, #f9fafb 100%);
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06);
  transition: all 0.3s ease;
  cursor: pointer;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}

.maintenance-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  border-color: #d1d5db;
}

.restart-card:hover {
  border-color: #f59e0b;
  box-shadow: 0 10px 25px -5px rgba(245, 158, 11, 0.15), 0 10px 10px -5px rgba(245, 158, 11, 0.04);
}

.reset-card:hover {
  border-color: #ef4444;
  box-shadow: 0 10px 25px -5px rgba(239, 68, 68, 0.15), 0 10px 10px -5px rgba(239, 68, 68, 0.04);
}

.card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 60px;
  height: 60px;
  border-radius: 0;
  margin: 20px auto 16px;
  background: linear-gradient(135deg, #f3f4f6 0%, #e5e7eb 100%);
  color: #6b7280;
  transition: all 0.3s ease;
}

.restart-card .card-icon {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  color: #d97706;
}

.reset-card .card-icon {
  background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
  color: #dc2626;
}

.maintenance-card:hover .card-icon {
  transform: scale(1.1);
}

.card-content {
  padding: 0 20px 16px;
  flex: 1;
}

.card-title-text {
  font-size: 18px;
  font-weight: 600;
  color: #111827;
  margin-bottom: 8px;
  text-align: center;
}

.card-description {
  font-size: 14px;
  color: #6b7280;
  line-height: 1.5;
  text-align: center;
  margin-bottom: 16px;
}

.card-actions {
  padding: 0 20px 20px;
  text-align: center;
}

.action-btn {
  border-radius: 0;
  font-weight: 500;
  transition: all 0.3s ease;
  min-width: 120px;
}

.action-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

/* 深色主题支持 */
.dark-theme .system-maintenance-card {
  background: #1f2937;
  border-color: #374151;
  color: #f9fafb;
}

.dark-theme .card-title {
  color: #f9fafb;
}

.dark-theme .status-alert {
  background: linear-gradient(135deg, #064e3b 0%, #065f46 100%);
  border-color: #10b981;
}

.dark-theme .maintenance-card {
  background: linear-gradient(135deg, #374151 0%, #4b5563 100%);
  border-color: #4b5563;
}

.dark-theme .restart-card:hover {
  border-color: #f59e0b;
  box-shadow: 0 10px 25px -5px rgba(245, 158, 11, 0.25), 0 10px 10px -5px rgba(245, 158, 11, 0.1);
}

.dark-theme .reset-card:hover {
  border-color: #ef4444;
  box-shadow: 0 10px 25px -5px rgba(239, 68, 68, 0.25), 0 10px 10px -5px rgba(239, 68, 68, 0.1);
}

.dark-theme .card-icon {
  background: linear-gradient(135deg, #4b5563 0%, #6b7280 100%);
}

.dark-theme .restart-card .card-icon {
  background: linear-gradient(135deg, #451a03 0%, #92400e 100%);
  color: #fbbf24;
}

.dark-theme .reset-card .card-icon {
  background: linear-gradient(135deg, #450a0a 0%, #991b1b 100%);
  color: #fca5a5;
}

.dark-theme .card-title-text {
  color: #f9fafb;
}

.dark-theme .card-description {
  color: #d1d5db;
}
</style>
