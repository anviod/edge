<template>
  <a-card class="mb-4">
    <a-card-header>
      <div class="card-title"></div>
    </a-card-header>
    <a-card-body>
      <a-form :model="timeConfig" layout="vertical" class="industrial-form">
        <a-form-item field="mode" label="同步模式">
          <a-radio-group v-model="timeConfig.mode" type="button" size="small">
            <a-radio value="manual">手动设置</a-radio>
            <a-radio value="ntp">NTP 服务器</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-row :gutter="16" v-if="timeConfig.mode === 'ntp'">
          <a-col :span="12">
            <a-form-item field="ntp.servers" label="NTP 服务器">
              <a-select 
                v-model="timeConfig.ntp.servers" 
                :options="ntpOptions" 
                mode="tags" 
                allow-create 
                placeholder="选择或输入NTP服务器地址" 
                class="rect-input" 
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item field="ntp.interval" label="同步间隔 (小时)">
              <a-input-number v-model="timeConfig.ntp.interval" :min="1" :max="24" class="rect-input" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16" v-if="timeConfig.mode === 'manual'">
          <a-col :span="12">
            <a-form-item field="manual.datetime" label="本地时间">
              <a-input v-model="timeConfig.manual.datetime" type="datetime-local" class="rect-input" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item field="manual.timezone" label="时区">
              <a-select v-model="timeConfig.manual.timezone" :options="[{label:'亚洲/上海', value:'Asia/Shanghai'}, {label:'UTC', value:'UTC'}]" class="rect-input" />
            </a-form-item>
          </a-col>
        </a-row>
        <div class="form-footer">
          <a-button type="primary" @click="$emit('save')">保存更改</a-button>
        </div>
      </a-form>
    </a-card-body>
  </a-card>
</template>

<script setup>
import { reactive } from 'vue'

const ntpOptions = [
  { label: 'pool.ntp.org', value: 'pool.ntp.org' },
  { label: 'time.nist.gov', value: 'time.nist.gov' },
  { label: 'ntp.aliyun.com', value: 'ntp.aliyun.com' },
  { label: 'ntp.tencent.com', value: 'ntp.tencent.com' },
  { label: 'time.windows.com', value: 'time.windows.com' },
  { label: 'asia.pool.ntp.org', value: 'asia.pool.ntp.org' }
]

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({
      mode: 'manual',
      manual: {
        datetime: '',
        timezone: 'Asia/Shanghai',
        sync_rtc: true
      },
      ntp: {
        servers: ['pool.ntp.org'],
        interval: 1,
        enabled: true
      }
    })
  }
})

const emit = defineEmits(['update:modelValue', 'save'])

const timeConfig = reactive({
  ...props.modelValue
})
</script>

<style scoped>
.card-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-gray-900);
  letter-spacing: 0.5px;
}

.industrial-form :deep(.arco-form-item) {
  margin-bottom: 16px;
}

.industrial-form :deep(.arco-form-item-label) {
  font-size: 11px;
  color: var(--color-gray-50);
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
