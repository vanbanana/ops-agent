import { authFetch } from '../lib/auth'
import { type FC, useState, useEffect, useCallback } from 'react'

interface ProviderPublic {
  id: string
  name: string
  provider: string
  base_url: string
  api_key_masked: string
  model_id: string
  context_window: number
  max_output: number
  is_active: boolean
  can_reason: boolean
}

interface ProviderForm {
  id: string
  name: string
  provider: string
  base_url: string
  api_key: string
  model_id: string
  context_window: number
  max_output: number
  can_reason: boolean
}

type TestState =
  | { status: 'idle' }
  | { status: 'testing' }
  | { status: 'success'; latency: number }
  | { status: 'error'; message: string }

// 2026.05 最新预制模型（用户只需填写 API Key）
interface ModelPreset {
  id: string
  name: string
  provider: string
  base_url: string
  model_id: string
  context_window: number
  max_output: number
  can_reason: boolean
  desc: string
}

const MODEL_PRESETS: ModelPreset[] = [
  {
    id: 'mimo-v25-pro', name: 'MiMo V2.5 Pro', provider: 'xiaomi',
    base_url: 'https://token-plan-cn.xiaomimimo.com/v1',
    model_id: 'mimo-v2.5-pro', context_window: 1000000, max_output: 16384,
    can_reason: true, desc: '小米旗舰，1M上下文，Agent场景强',
  },
  {
    id: 'mimo-v2-flash', name: 'MiMo V2 Flash', provider: 'xiaomi',
    base_url: 'https://token-plan-cn.xiaomimimo.com/v1',
    model_id: 'mimo-v2-flash', context_window: 256000, max_output: 16384,
    can_reason: true, desc: '小米轻量，256K上下文，性价比高',
  },
  {
    id: 'ds-v4-pro', name: 'DeepSeek V4 Pro', provider: 'deepseek',
    base_url: 'https://api.deepseek.com',
    model_id: 'deepseek-v4-pro', context_window: 1000000, max_output: 8192,
    can_reason: true, desc: '深度求索旗舰，1M上下文，推理编码强',
  },
  {
    id: 'ds-v4-flash', name: 'DeepSeek V4 Flash', provider: 'deepseek',
    base_url: 'https://api.deepseek.com',
    model_id: 'deepseek-v4-flash', context_window: 1000000, max_output: 8192,
    can_reason: true, desc: '深度求索快速版，高吞吐低延迟',
  },
  {
    id: 'qwen36-plus', name: 'Qwen 3.6 Plus', provider: 'alibaba',
    base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    model_id: 'qwen3.6-plus', context_window: 1000000, max_output: 65536,
    can_reason: true, desc: '通义千问旗舰，1M上下文，65K输出',
  },
  {
    id: 'gpt-55', name: 'GPT-5.5', provider: 'openai',
    base_url: 'https://api.openai.com/v1',
    model_id: 'gpt-5.5', context_window: 1000000, max_output: 128000,
    can_reason: true, desc: 'OpenAI 旗舰，1M上下文，128K输出',
  },
  {
    id: 'claude-sonnet-46', name: 'Claude Sonnet 4.6', provider: 'anthropic',
    base_url: 'https://api.anthropic.com/v1',
    model_id: 'claude-sonnet-4-6', context_window: 1000000, max_output: 64000,
    can_reason: true, desc: 'Anthropic 编码旗舰，1M上下文',
  },
]

export const ModelSettings: FC = () => {
  const [providers, setProviders] = useState<ProviderPublic[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [testStates, setTestStates] = useState<Record<string, TestState>>({})
  const [form, setForm] = useState<ProviderForm>({
    id: '', name: '', provider: '', base_url: '', api_key: '',
    model_id: '', context_window: 32768, max_output: 8192, can_reason: false,
  })

  const loadProviders = useCallback(async () => {
    try {
      const res = await authFetch('/api/v1/models/pool')
      const data = await res.json()
      if (data?.data?.providers) setProviders(data.data.providers)
    } catch { setError('加载模型列表失败') }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { loadProviders() }, [loadProviders])

  const handleSwitch = async (id: string) => {
    try {
      const res = await authFetch('/api/v1/models/switch', {
        body: JSON.stringify({ id }),
      })
      if (res.ok) await loadProviders()
      else { const b = await res.json().catch(() => ({})); setError(b.error || '切换失败') }
    } catch { setError('网络错误') }
  }

  const handleTest = async (id: string) => {
    setTestStates(p => ({ ...p, [id]: { status: 'testing' } }))
    try {
      const res = await authFetch('/api/v1/models/test', {
        body: JSON.stringify({ id }),
      })
      const data = await res.json()
      if (data?.data?.success) setTestStates(p => ({ ...p, [id]: { status: 'success', latency: data.data.latency_ms } }))
      else setTestStates(p => ({ ...p, [id]: { status: 'error', message: data?.data?.error || '未知错误' } }))
    } catch (e) { setTestStates(p => ({ ...p, [id]: { status: 'error', message: String(e) } })) }
  }

  const handleDelete = async (id: string) => {
    const remaining = providers.filter(p => p.id !== id)
    if (remaining.length === 0) { setError('至少需要保留一个模型'); return }
    try {
      const res = await authFetch('/api/v1/models/pool', {
        body: JSON.stringify({ providers: remaining.map(p => ({
          id: p.id, name: p.name, provider: p.provider, base_url: p.base_url,
          api_key: '', model_id: p.model_id, context_window: p.context_window,
          max_output: p.max_output, is_active: p.is_active, can_reason: p.can_reason,
        })) }),
      })
      if (res.ok) await loadProviders()
      else await loadProviders()
    } catch { setError('删除失败') }
  }

  const openAddForm = () => {
    setForm({ id: '', name: '', provider: '', base_url: '', api_key: '', model_id: '', context_window: 32768, max_output: 8192, can_reason: false })
    setEditingId(null)
    setShowForm(true)
    setError(null)
  }

  const applyPreset = (preset: ModelPreset) => {
    setForm({
      id: preset.id, name: preset.name, provider: preset.provider,
      base_url: preset.base_url, api_key: '', model_id: preset.model_id,
      context_window: preset.context_window, max_output: preset.max_output,
      can_reason: preset.can_reason,
    })
  }

  const openEditForm = (p: ProviderPublic) => {
    setForm({
      id: p.id, name: p.name, provider: p.provider, base_url: p.base_url,
      api_key: '', model_id: p.model_id, context_window: p.context_window,
      max_output: p.max_output, can_reason: p.can_reason,
    })
    setEditingId(p.id)
    setShowForm(true)
    setError(null)
  }

  const handleSubmitForm = async () => {
    if (!form.id || !form.name || !form.base_url || !form.model_id) {
      setError('ID、名称、Base URL、Model ID 为必填项'); return
    }
    // API Key 仅在新增且没有同 base_url 的已有模型时必填
    const hasSameProvider = providers.some(p => p.base_url === form.base_url)
    if (!editingId && !form.api_key && !hasSameProvider) {
      setError('API Key 为必填项（同供应商已有模型时可留空复用）'); return
    }
    const newProvider = {
      id: form.id, name: form.name, provider: form.provider,
      base_url: form.base_url, api_key: form.api_key, model_id: form.model_id,
      context_window: form.context_window, max_output: form.max_output,
      is_active: false, can_reason: form.can_reason,
    }
    let updatedList
    if (editingId) {
      updatedList = providers.map(p => p.id === editingId
        ? { ...p, ...newProvider, is_active: p.is_active, api_key: form.api_key || '' }
        : { ...p, api_key: '' })
    } else {
      updatedList = [...providers.map(p => ({ ...p, api_key: '' })), newProvider]
    }
    try {
      const res = await authFetch('/api/v1/models/pool', {
        body: JSON.stringify({ providers: updatedList }),
      })
      if (res.ok) { setShowForm(false); await loadProviders() }
      else { const b = await res.json().catch(() => ({})); setError(b.error || '保存失败') }
    } catch { setError('网络错误') }
  }

  if (loading) {
    return (
      <div style={{ padding: 48, display: 'flex', justifyContent: 'center' }}>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 13, color: 'var(--ops-fg-muted)' }}>加载中...</span>
      </div>
    )
  }

  return (
    <div style={{ flex: 1, overflow: 'auto', background: 'var(--ops-bg-canvas)', padding: '24px 48px' }}>
      <h2 style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 16, fontWeight: 600, color: 'var(--ops-fg-primary)', margin: '0 0 8px 0', display: 'flex', alignItems: 'center', gap: 8 }}>
        <span className="material-symbols-outlined" style={{ fontSize: 18 }}>smart_toy</span>
        模型管理
      </h2>
      <p style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)', margin: '0 0 24px 0' }}>
        管理 LLM 模型供应商，运行时切换激活模型无需重启服务。
      </p>

      {error && (
        <div style={{ padding: '8px 12px', borderRadius: 4, background: 'rgba(255,59,48,0.08)', border: '1px solid var(--ops-status-danger)', marginBottom: 16 }}>
          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-status-danger)' }}>{error}</span>
          <button onClick={() => setError(null)} style={{ marginLeft: 8, background: 'none', border: 'none', color: 'var(--ops-status-danger)', cursor: 'pointer', fontSize: 11 }}>关闭</button>
        </div>
      )}

      {/* Provider cards */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {providers.map(p => {
          const ts = testStates[p.id] || { status: 'idle' as const }
          return (
            <div key={p.id} style={{ padding: '16px 20px', borderRadius: 8, border: '1px solid var(--ops-border-subtle)', borderLeft: p.is_active ? '3px solid #34c759' : '3px solid transparent', background: 'var(--ops-bg-elevated)', display: 'flex', alignItems: 'center', gap: 16 }}>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                  <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 14, fontWeight: 600, color: 'var(--ops-fg-primary)' }}>{p.name}</span>
                  {p.is_active && <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 10, fontWeight: 500, color: '#34c759', background: 'rgba(52,199,89,0.1)', padding: '2px 6px', borderRadius: 3 }}>使用中</span>}
                  {p.can_reason && <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 10, color: 'var(--ops-fg-muted)', background: 'var(--ops-bg-canvas)', padding: '2px 6px', borderRadius: 3 }}>支持推理</span>}
                </div>
                <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
                  <InfoBit label="供应商" value={p.provider || '-'} />
                  <InfoBit label="模型" value={p.model_id} />
                  <InfoBit label="地址" value={truncUrl(p.base_url)} />
                  <InfoBit label="密钥" value={p.api_key_masked} mono />
                  <InfoBit label="上下文" value={fmtTokens(p.context_window)} />
                </div>
                {ts.status === 'testing' && <div style={{ marginTop: 6, fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>正在测试连通性...</div>}
                {ts.status === 'success' && <div style={{ marginTop: 6, fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: '#34c759' }}><span className="material-symbols-outlined" style={{ fontSize: 12, verticalAlign: 'middle', marginRight: 4 }}>check_circle</span>连通成功 ({ts.latency}ms)</div>}
                {ts.status === 'error' && <div style={{ marginTop: 6, fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-status-danger)' }}><span className="material-symbols-outlined" style={{ fontSize: 12, verticalAlign: 'middle', marginRight: 4 }}>cancel</span>{ts.message}</div>}
              </div>
              <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
                {!p.is_active && <ActionBtn icon="play_arrow" title="设为激活" onClick={() => handleSwitch(p.id)} color="#34c759" />}
                <ActionBtn icon="wifi_tethering" title="测试连通" onClick={() => handleTest(p.id)} />
                <ActionBtn icon="edit" title="编辑" onClick={() => openEditForm(p)} />
                {!p.is_active && <ActionBtn icon="delete" title="删除" onClick={() => handleDelete(p.id)} color="var(--ops-status-danger)" />}
              </div>
            </div>
          )
        })}
      </div>

      {/* Add button */}
      <button onClick={openAddForm} style={{ marginTop: 16, width: '100%', padding: '12px', borderRadius: 8, border: '1px dashed var(--ops-border-default)', background: 'transparent', fontFamily: 'var(--ops-font-ui)', fontSize: 13, color: 'var(--ops-fg-secondary)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>
        <span className="material-symbols-outlined" style={{ fontSize: 16 }}>add</span>
        添加新模型
      </button>

      {/* Add/Edit form modal */}
      {showForm && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div style={{ background: 'var(--ops-bg-elevated)', border: '1px solid var(--ops-border-default)', borderRadius: 10, padding: 24, width: 480, maxHeight: '85vh', overflow: 'auto' }}>
            <h3 style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 15, fontWeight: 600, color: 'var(--ops-fg-primary)', margin: '0 0 16px 0' }}>
              {editingId ? '编辑模型' : '添加模型'}
            </h3>

            {/* Preset quick select (only show in add mode) */}
            {!editingId && (
              <div style={{ marginBottom: 16 }}>
                <label style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', display: 'block', marginBottom: 6 }}>快速选择预制模型（选择后仅需填写 API Key）</label>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                  {MODEL_PRESETS.map(preset => (
                    <button
                      key={preset.id}
                      onClick={() => applyPreset(preset)}
                      title={preset.desc}
                      style={{
                        padding: '5px 10px',
                        borderRadius: 5,
                        border: form.id === preset.id ? '1px solid var(--ops-fg-primary)' : '1px solid var(--ops-border-default)',
                        background: form.id === preset.id ? 'var(--ops-bg-canvas)' : 'transparent',
                        fontFamily: 'var(--ops-font-ui)',
                        fontSize: 11,
                        color: form.id === preset.id ? 'var(--ops-fg-primary)' : 'var(--ops-fg-secondary)',
                        cursor: 'pointer',
                        fontWeight: form.id === preset.id ? 500 : 400,
                      }}
                    >
                      {preset.name}
                    </button>
                  ))}
                </div>
                {form.id && MODEL_PRESETS.find(p => p.id === form.id) && (
                  <div style={{ marginTop: 6, fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>
                    {MODEL_PRESETS.find(p => p.id === form.id)?.desc}
                  </div>
                )}
              </div>
            )}

            <FormField label="ID *" placeholder="如 mimo-pro" value={form.id} onChange={v => setForm(f => ({ ...f, id: v }))} disabled={!!editingId} mono />
            <FormField label="显示名称 *" placeholder="如 MiMo V2.5 Pro" value={form.name} onChange={v => setForm(f => ({ ...f, name: v }))} />
            <FormField label="供应商" placeholder="如 xiaomi, deepseek, alibaba" value={form.provider} onChange={v => setForm(f => ({ ...f, provider: v }))} />
            <FormField label="Base URL *" placeholder="https://api.example.com/v1" value={form.base_url} onChange={v => setForm(f => ({ ...f, base_url: v }))} mono />
            <FormField label={editingId ? 'API Key (留空保持不变)' : 'API Key *'} placeholder="sk-..." value={form.api_key} onChange={v => setForm(f => ({ ...f, api_key: v }))} mono type="password" />
            <FormField label="Model ID *" placeholder="如 mimo-v2.5-pro" value={form.model_id} onChange={v => setForm(f => ({ ...f, model_id: v }))} mono />

            <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
              <div style={{ flex: 1 }}>
                <label style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', display: 'block', marginBottom: 4 }}>上下文窗口</label>
                <input type="number" value={form.context_window} onChange={e => setForm(f => ({ ...f, context_window: parseInt(e.target.value) || 0 }))} style={{ width: '100%', padding: '6px 10px', fontFamily: 'var(--ops-font-mono)', fontSize: 12, border: '1px solid var(--ops-border-default)', borderRadius: 4, background: 'var(--ops-bg-input)', color: 'var(--ops-fg-primary)', outline: 'none' }} />
              </div>
              <div style={{ flex: 1 }}>
                <label style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', display: 'block', marginBottom: 4 }}>最大输出</label>
                <input type="number" value={form.max_output} onChange={e => setForm(f => ({ ...f, max_output: parseInt(e.target.value) || 0 }))} style={{ width: '100%', padding: '6px 10px', fontFamily: 'var(--ops-font-mono)', fontSize: 12, border: '1px solid var(--ops-border-default)', borderRadius: 4, background: 'var(--ops-bg-input)', color: 'var(--ops-fg-primary)', outline: 'none' }} />
              </div>
            </div>

            <label style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 20, cursor: 'pointer' }}>
              <input type="checkbox" checked={form.can_reason} onChange={e => setForm(f => ({ ...f, can_reason: e.target.checked }))} />
              <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-primary)' }}>支持推理/思考链</span>
            </label>

            {error && <div style={{ marginBottom: 12, fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-status-danger)' }}>{error}</div>}

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
              <button onClick={() => setShowForm(false)} style={{ padding: '8px 18px', borderRadius: 6, border: '1px solid var(--ops-border-default)', background: 'transparent', fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-secondary)', cursor: 'pointer' }}>取消</button>
              <button onClick={handleSubmitForm} style={{ padding: '8px 18px', borderRadius: 6, border: 'none', background: 'var(--ops-fg-primary)', fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-bg-canvas)', cursor: 'pointer', fontWeight: 500 }}>{editingId ? '保存' : '添加'}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// Sub-components

const InfoBit: FC<{ label: string; value: string; mono?: boolean }> = ({ label, value, mono }) => (
  <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>
    {label}: <span style={{ fontFamily: mono ? 'var(--ops-font-mono)' : 'var(--ops-font-ui)', color: 'var(--ops-fg-secondary)' }}>{value}</span>
  </span>
)

const ActionBtn: FC<{ icon: string; title: string; onClick: () => void; color?: string }> = ({ icon, title, onClick, color }) => (
  <button onClick={onClick} title={title} style={{ width: 30, height: 30, display: 'flex', alignItems: 'center', justifyContent: 'center', borderRadius: 6, border: '1px solid var(--ops-border-subtle)', background: 'var(--ops-bg-canvas)', cursor: 'pointer', color: color || 'var(--ops-fg-secondary)' }}>
    <span className="material-symbols-outlined" style={{ fontSize: 16 }}>{icon}</span>
  </button>
)

const FormField: FC<{ label: string; placeholder?: string; value: string; onChange: (v: string) => void; disabled?: boolean; mono?: boolean; type?: string }> = ({ label, placeholder, value, onChange, disabled, mono, type }) => (
  <div style={{ marginBottom: 12 }}>
    <label style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', display: 'block', marginBottom: 4 }}>{label}</label>
    <input type={type || 'text'} value={value} onChange={e => onChange(e.target.value)} disabled={disabled} placeholder={placeholder} style={{ width: '100%', padding: '6px 10px', fontFamily: mono ? 'var(--ops-font-mono)' : 'var(--ops-font-ui)', fontSize: 12, border: '1px solid var(--ops-border-default)', borderRadius: 4, background: disabled ? 'var(--ops-bg-canvas)' : 'var(--ops-bg-input)', color: 'var(--ops-fg-primary)', outline: 'none', opacity: disabled ? 0.6 : 1 }} />
  </div>
)

function truncUrl(url: string): string {
  try { return new URL(url).host } catch { return url.length > 30 ? url.slice(0, 30) + '...' : url }
}

function fmtTokens(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(0) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(0) + 'K'
  return String(n)
}
