import type { SelectGroupOption } from 'naive-ui'

export const DEFAULT_MODEL = 'gemini-4-neon-xhigh'

export const MODEL_OPTIONS: SelectGroupOption[] = [
  {
    type: 'group',
    label: 'Gemini 4 Neon 系列',
    key: 'group-gemini-4-neon',
    children: [
      { label: 'Gemini 4 Neon (XHigh) - 旗舰深度思考', value: 'gemini-4-neon-xhigh' },
      { label: 'Gemini 4 Neon (High)', value: 'gemini-4-neon-high' },
      { label: 'Gemini 4 Neon (Medium)', value: 'gemini-4-neon-medium' },
      { label: 'Gemini 4 Neon (Low)', value: 'gemini-4-neon-low' }
    ]
  },
  {
    type: 'group',
    label: 'Gemini 4 Hydrogen / Argon / Carbon 系列',
    key: 'group-gemini-4-elements',
    children: [
      { label: 'Gemini 4 Hydrogen (XHigh)', value: 'gemini-4-hydrogen-xhigh' },
      { label: 'Gemini 4 Hydrogen (High)', value: 'gemini-4-hydrogen-high' },
      { label: 'Gemini 4 Argon (High)', value: 'gemini-4-argon-high' },
      { label: 'Gemini 4 Argon (Medium)', value: 'gemini-4-argon-medium' },
      { label: 'Gemini 4 Carbon (High)', value: 'gemini-4-carbon-high' }
    ]
  },
  {
    type: 'group',
    label: 'Gemini 3.8 Flash & Antigravity 编排模型',
    key: 'group-gemini-38-orchestrator',
    children: [
      { label: 'Gemini 3.8 Flash (High) - 极速响应', value: 'gemini-3.8-flash-high' },
      { label: 'Gemini 3.8 Flash (Medium)', value: 'gemini-3.8-flash-medium' },
      { label: 'Orchestrator (Neon)', value: 'gemini-pro-orchestrator-neon' },
      { label: 'Claude Opus 5.5 (High)', value: 'claude-opus-5-5-high' },
      { label: 'Claude Sonnet 5.5 (High)', value: 'claude-sonnet-5-5-high' }
    ]
  },
  {
    type: 'group',
    label: '经典兼容模型 (Legacy)',
    key: 'group-legacy',
    children: [
      { label: 'Gemini 2.5 Pro (Legacy)', value: 'gemini-2.5-pro' },
      { label: 'Gemini 2.5 Flash (Legacy)', value: 'gemini-2.5-flash' },
      { label: 'Gemini 2.5 Flash-Lite (Legacy)', value: 'gemini-2.5-flash-lite' }
    ]
  }
]
