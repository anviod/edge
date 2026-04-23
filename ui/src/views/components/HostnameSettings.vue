<template>
  <a-card class="mb-4">
    <a-card-header>
      <div class="card-title"></div>
    </a-card-header>
    <a-card-body>
      <a-form :model="hostnameConfig" layout="vertical" class="industrial-form">
        <a-form-item field="name" label="设备名称">
          <a-input v-model="hostnameConfig.name" placeholder="输入设备名称" class="rect-input" />
          <div class="form-hint">访问地址: http://device-name</div>
        </a-form-item>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item field="http_port" label="HTTP 端口">
              <a-input-number v-model="hostnameConfig.http_port" :min="1" :max="65535" class="rect-input" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item field="https_port" label="HTTPS 端口">
              <a-input-number v-model="hostnameConfig.https_port" :min="1" :max="65535" class="rect-input" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item field="interfaces" label="绑定接口">
          <a-select
            v-model="hostnameConfig.interfaces"
            :options="networkInterfaces.map(i => ({ label: i.name, value: i.name }))"
            mode="multiple"
            placeholder="留空则绑定所有可用接口"
            class="rect-input"
          />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item field="enable_mdns" label="mDNS 服务">
              <a-switch v-model="hostnameConfig.enable_mdns" type="round" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item field="enable_bare" label="裸主机名">
              <a-switch v-model="hostnameConfig.enable_bare" type="round" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <div class="form-footer">
          <a-button type="primary" @click="$emit('save')">应用设置</a-button>
        </div>
      </a-form>

      <a-divider class="my-4" />

      <div class="card-title">访问状态</div>
      <a-tag color="success" size="small" class="mb-4">广播状态: 正常</a-tag>
      <div class="access-list">
        <div class="access-item">
          <div class="access-title">HTTP 访问</div>
          <div class="access-subtitle mono-text">{{ `http://${hostnameConfig.name}:${hostnameConfig.http_port}` }}</div>
        </div>
        <div class="access-item">
          <div class="access-title">HTTPS 访问</div>
          <div class="access-subtitle mono-text">{{ `https://${hostnameConfig.name}:${hostnameConfig.https_port}` }}</div>
        </div>
        <div class="access-item">
          <div class="access-title">mDNS 访问</div>
          <div class="access-subtitle mono-text">{{ `http://${hostnameConfig.name}.local:${hostnameConfig.http_port}` }}</div>
        </div>
      </div>
    </a-card-body>
  </a-card>
</template>

<script setup>
import { reactive } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({
      name: 'edge-gateway',
      enable_mdns: true,
      enable_bare: true,
      http_port: 8082,
      https_port: 443,
      interfaces: []
    })
  },
  networkInterfaces: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue', 'save'])

const hostnameConfig = reactive({
  ...props.modelValue
})
</script>

<style scoped>
.card-title {
  font-size: 12px;
  font-weight: 600;
  color: #374151;
  letter-spacing: 0.5px;
}

.industrial-form :deep(.arco-form-item) {
  margin-bottom: 16px;
}

.industrial-form :deep(.arco-form-item-label) {
  font-size: 11px;
  color: #6b7280;
  font-weight: 500;
}

.rect-input {
  border-radius: 0 !important;
  font-family: 'JetBrains Mono', monospace;
}

.form-hint {
  font-size: 11px;
  color: #6b7280;
  margin-top: 4px;
}

.form-footer {
  margin-top: 16px;
}

.access-list {
  margin-top: 16px;
}

.access-item {
  margin-bottom: 12px;
}

.access-title {
  font-size: 12px;
  font-weight: bold;
  margin-bottom: 4px;
}

.access-subtitle {
  margin-left: 16px;
}

.mono-text { font-family: 'JetBrains Mono', monospace; font-size: 12px; }
</style>
