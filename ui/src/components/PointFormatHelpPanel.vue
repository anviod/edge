<template>
  <v-card variant="outlined" class="mb-4">
    <v-card-title class="d-flex align-center py-2 px-4" @click="toggle">
      <v-icon icon="mdi-help-circle-outline" color="primary" class="mr-2"></v-icon>
      <span class="text-subtitle-2 font-weight-bold">{{ t('panel.panelTitle') }}</span>
      <v-spacer></v-spacer>
      <v-chip size="x-small" variant="outlined" class="mr-2">
        {{ flatFormats.length }}
      </v-chip>
      <v-btn icon variant="text" size="small">
        <v-icon :icon="expanded ? 'mdi-chevron-up' : 'mdi-chevron-down'"></v-icon>
      </v-btn>
    </v-card-title>
    <v-expand-transition>
      <div v-show="expanded">
        <v-divider></v-divider>
        <v-card-text class="pa-3">
          <v-table density="compact" class="text-body-2">
            <thead>
              <tr>
                <th class="text-left" style="width: 26%;">{{ t('panel.columnFormat') }}</th>
                <th class="text-left" style="width: 26%;">{{ t('panel.columnRange') }}</th>
                <th class="text-left" style="width: 18%;">{{ t('panel.columnShortcut') }}</th>
                <th class="text-left">{{ t('panel.columnExample') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="key in flatFormats" :key="key.name">
                <td>
                  <div class="font-weight-medium">
                    <span class="text-caption text-grey mr-1">{{ key.groupLabel }}</span>
                    <span v-html="sanitize(t(`formats.${key.name}.title`))"></span>
                  </div>
                  <div class="text-caption text-grey" v-html="sanitize(t(`formats.${key.name}.subtitle`))"></div>
                </td>
                <td>
                  <div v-html="sanitize(t(`formats.${key.name}.range`))"></div>
                  <div class="text-caption text-grey" v-html="sanitize(t(`formats.${key.name}.registers`))"></div>
                </td>
                <td>
                  <span class="font-mono" v-html="sanitize(t(`formats.${key.name}.shortcut`))"></span>
                </td>
                <td>
                  <div v-html="sanitize(t(`formats.${key.name}.example`))"></div>
                </td>
              </tr>
            </tbody>
          </v-table>
        </v-card-text>
      </div>
    </v-expand-transition>
  </v-card>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import helpDefs from '@/i18n/pointFormatHelp.json'
import { sanitizeHtml } from '@/utils/sanitizeHtml'

const props = defineProps({
  lang: {
    type: String,
    default: 'zh'
  }
})

const buildMessages = (defs) => {
  const en = { panel: {}, formats: {} }
  const zh = { panel: {}, formats: {} }

  const metaFields = [
    'panelTitle',
    'columnFormat',
    'columnRange',
    'columnShortcut',
    'columnExample',
    'quickSwitchMenu',
    'resetButton',
    'resetTooltip',
    'recentToggle',
    'recentToggleTooltip'
  ]

  metaFields.forEach((field) => {
    const enVal = defs._meta?.en?.[field] || ''
    const zhVal = defs._meta?.zh?.[field]
    en.panel[field] = enVal
    if (!zhVal && enVal) {
      console.warn(`Missing zh translation for panel.${field}, fallback to en`)
      zh.panel[field] = enVal
    } else {
      zh.panel[field] = zhVal || ''
    }
  })

  const formatFields = ['title', 'subtitle', 'range', 'registers', 'shortcut', 'example']

  Object.keys(defs).forEach((key) => {
    if (key === '_meta') return
    const item = defs[key]
    en.formats[key] = {}
    zh.formats[key] = {}
    formatFields.forEach((field) => {
      const enVal = item.en?.[field] || ''
      const zhVal = item.zh?.[field]
      en.formats[key][field] = enVal
      if (!zhVal && enVal && field !== 'subtitle') {
        console.warn(`Missing zh translation for ${key}.${field}, fallback to en`)
        zh.formats[key][field] = enVal
      } else {
        zh.formats[key][field] = zhVal || ''
      }
    })
  })

  return { en, zh }
}

const messages = buildMessages(helpDefs)

const { t, locale } = useI18n({
  legacy: false,
  useScope: 'local',
  messages,
  inheritLocale: false,
  locale: props.lang || 'zh'
})

watch(
  () => props.lang,
  (val) => {
    locale.value = val || 'zh'
  }
)

const groupMeta = {
  one: {
    bytes: 1,
    label: '1 字节'
  },
  two: {
    bytes: 2,
    label: '2 字节 / 1 寄存器'
  },
  four: {
    bytes: 4,
    label: '4 字节 / 2 寄存器'
  },
  eight: {
    bytes: 8,
    label: '8 字节 / 4 寄存器'
  }
}

const formatGroups = [
  {
    key: 'two',
    names: ['Signed', 'Unsigned', 'Hex', 'Binary']
  },
  {
    key: 'four',
    names: ['Long AB CD', 'Long CD AB', 'Long BA DC', 'Long DC BA', 'Float AB CD', 'Float CD AB', 'Float BA DC', 'Float DC BA']
  },
  {
    key: 'eight',
    names: ['Double AB CDEF GH', 'Double GH EFCD AB', 'Double BA DC FE HG', 'Double HG FE DC BA']
  }
]

const flatFormats = computed(() => {
  const result = []
  formatGroups.forEach((g) => {
    const meta = groupMeta[g.key]
    const groupLabel = meta ? meta.label : ''
    g.names.forEach((name) => {
      if (helpDefs[name]) {
        result.push({ name, groupLabel })
      }
    })
  })
  return result
})

const expanded = ref(false)

const toggle = () => {
  expanded.value = !expanded.value
}

const sanitize = (html) => sanitizeHtml(html)
</script>

