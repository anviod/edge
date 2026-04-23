<template>
  <a-card class="mb-4">
    <a-card-header>
      <div class="card-title"></div>
    </a-card-header>
    <a-card-body>
      <a-form :model="ldapConfig" layout="vertical" class="industrial-form">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item field="enabled" label="服务">
              <a-switch v-model="ldapConfig.enabled" type="round" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item field="port" label="端口">
              <a-input-number v-model="ldapConfig.port" class="rect-input" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item field="use_ssl" label="SSL/TLS">
              <a-switch v-model="ldapConfig.use_ssl" type="round" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item field="server" label="服务器地址">
          <a-input v-model="ldapConfig.server" placeholder="ldap://internal.edge-os.io" class="rect-input" />
        </a-form-item>
        <a-form-item field="base_dn" label="基础 DN">
          <a-input v-model="ldapConfig.base_dn" class="rect-input" />
        </a-form-item>
        <a-divider class="my-4" />
        <div class="form-footer">
          <a-button type="primary" long @click="$emit('save')">部署 LDAP 配置</a-button>
        </div>
      </a-form>
    </a-card-body>
  </a-card>
</template>

<script setup>
import { reactive } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({
      enabled: false,
      server: '',
      port: 389,
      base_dn: '',
      bind_dn: '',
      bind_password: '',
      user_filter: '(uid=%s)',
      attributes: '',
      use_ssl: false,
      skip_verify: false
    })
  }
})

const emit = defineEmits(['update:modelValue', 'save'])

const ldapConfig = reactive({
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

.form-footer {
  margin-top: 16px;
}
</style>
