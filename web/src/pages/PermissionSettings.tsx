import { type FC, useState, useEffect } from 'react'

interface CommandEntry {
  name: string
  type: 'readonly' | 'readwrite' | 'write'
  requiresPathCheck: boolean
  allowedSubcommands: string[]
  forbiddenArgs: string[]
}

// Permanent blacklist — cannot be added to whitelist
const PERMANENT_BLACKLIST = [
  'rm', 'dd', 'mkfs', 'fdisk', 'parted', 'wipefs', 'shred',
  'reboot', 'shutdown', 'init', 'halt', 'poweroff',
]

export const PermissionSettings: FC = () => {
  const [mode, setMode] = useState<'default' | 'auto_approve'>('default')
  const [commands, setCommands] = useState<CommandEntry[]>([])
  const [search, setSearch] = useState('')
  const [showAddForm, setShowAddForm] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  // Form state
  const [formName, setFormName] = useState('')
  const [formType, setFormType] = useState<'readonly' | 'readwrite' | 'write'>('readonly')
  const [formPathCheck, setFormPathCheck] = useState(false)
  const [formSubcommands, setFormSubcommands] = useState('')
  const [formForbiddenArgs, setFormForbiddenArgs] = useState('')

  // Load current settings
  useEffect(() => {
    fetch('/api/v1/permission/mode')
      .then(r => r.json())
      .then(d => { if (d?.data?.mode) setMode(d.data.mode) })
      .catch(() => {})

    // Load whitelist from safety rules (served from backend)
    fetch('/api/v1/tools')
      .then(r => r.json())
      .then(() => {
        // For now, load hardcoded defaults matching rules.go
        setCommands(getDefaultCommands())
      })
      .catch(() => setCommands(getDefaultCommands()))
  }, [])

  const handleModeChange = async (newMode: 'default' | 'auto_approve') => {
    const res = await fetch('/api/v1/permission/mode', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode: newMode }),
    })
    if (res.ok) setMode(newMode)
  }

  const filtered = commands.filter(c =>
    c.name.toLowerCase().includes(search.toLowerCase())
  )

  const openAddForm = () => {
    setFormName('')
    setFormType('readonly')
    setFormPathCheck(false)
    setFormSubcommands('')
    setFormForbiddenArgs('')
    setEditIndex(null)
    setShowAddForm(true)
    setError(null)
  }

  const openEditForm = (idx: number) => {
    const cmd = commands[idx]
    setFormName(cmd.name)
    setFormType(cmd.type)
    setFormPathCheck(cmd.requiresPathCheck)
    setFormSubcommands(cmd.allowedSubcommands.join(', '))
    setFormForbiddenArgs(cmd.forbiddenArgs.join(', '))
    setEditIndex(idx)
    setShowAddForm(true)
    setError(null)
  }

  const validateForm = (): string | null => {
    if (!formName.trim()) return '命令名不能为空'
    if (!/^[a-zA-Z0-9_-]+$/.test(formName.trim())) return '命令名只能包含字母数字下划线短横'
    if (PERMANENT_BLACKLIST.includes(formName.trim())) return `命令 "${formName}" 在永久黑名单中，不允许添加`
    if (editIndex === null && commands.some(c => c.name === formName.trim())) return '命令已存在'
    return null
  }

  const handleSubmitForm = () => {
    const err = validateForm()
    if (err) { setError(err); return }

    const entry: CommandEntry = {
      name: formName.trim(),
      type: formType,
      requiresPathCheck: formPathCheck,
      allowedSubcommands: formSubcommands ? formSubcommands.split(',').map(s => s.trim()).filter(Boolean) : [],
      forbiddenArgs: formForbiddenArgs ? formForbiddenArgs.split(',').map(s => s.trim()).filter(Boolean) : [],
    }

    if (editIndex !== null) {
      const updated = [...commands]
      updated[editIndex] = entry
      setCommands(updated)
    } else {
      setCommands([...commands, entry])
    }
    setShowAddForm(false)
  }

  const handleDelete = (idx: number) => {
    setCommands(commands.filter((_, i) => i !== idx))
    setShowAddForm(false)
  }

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      const whitelist = commands.map(c => ({
        name: c.name,
        type: c.type,
        requires_path_check: c.requiresPathCheck,
        allowed_subcommands: c.allowedSubcommands,
        forbidden_args: c.forbiddenArgs,
      }))
      const res = await fetch('/api/v1/configs', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ command_whitelist: JSON.stringify(whitelist) }),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        setError(body.error || '保存失败')
      }
    } catch {
      setError('网络错误，保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleResetDefaults = () => {
    setCommands(getDefaultCommands())
  }

  const typeLabel = (t: string) => {
    switch (t) {
      case 'readonly': return '只读'
      case 'readwrite': return '读写'
      case 'write': return '写'
      default: return t
    }
  }

  return (
    <div style={{ flex: 1, overflow: 'auto', background: 'var(--ops-bg-canvas)', padding: '24px 48px' }}>
      <h2 style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 16, fontWeight: 600, color: 'var(--ops-fg-primary)', margin: '0 0 24px 0' }}>
        <span className="material-symbols-outlined" style={{ fontSize: 18, verticalAlign: 'middle', marginRight: 6 }}>shield</span>
        权限设置
      </h2>

      {/* Permission Mode */}
      <div style={{ marginBottom: 32, padding: 16, background: 'var(--ops-bg-elevated)', borderRadius: 6, border: '1px solid var(--ops-border-subtle)' }}>
        <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 13, fontWeight: 500, color: 'var(--ops-fg-primary)', marginBottom: 12 }}>
          权限模式
        </div>
        <label style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, cursor: 'pointer' }}>
          <input type="radio" name="mode" checked={mode === 'default'} onChange={() => handleModeChange('default')} />
          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-primary)' }}>
            <span className="material-symbols-outlined" style={{ fontSize: 14, verticalAlign: 'middle', marginRight: 4 }}>lock</span>
            标准模式 — 写操作需要手动确认
          </span>
        </label>
        <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
          <input type="radio" name="mode" checked={mode === 'auto_approve'} onChange={() => handleModeChange('auto_approve')} />
          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-status-warn)' }}>
            <span className="material-symbols-outlined" style={{ fontSize: 14, verticalAlign: 'middle', marginRight: 4 }}>lock_open</span>
            全权限模式 — 自动执行所有操作（谨慎）
          </span>
        </label>
      </div>

      {/* Command Whitelist */}
      <div style={{ padding: 16, background: 'var(--ops-bg-elevated)', borderRadius: 6, border: '1px solid var(--ops-border-subtle)' }}>
        <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 13, fontWeight: 500, color: 'var(--ops-fg-primary)', marginBottom: 4 }}>
          命令白名单
        </div>
        <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', marginBottom: 12 }}>
          白名单中的命令允许 Agent 直接执行，不在白名单中的命令将被拦截。
        </div>

        {/* Search */}
        <input
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder="搜索命令..."
          style={{
            width: '100%',
            padding: '6px 10px',
            fontFamily: 'var(--ops-font-ui)',
            fontSize: 12,
            border: '1px solid var(--ops-border-default)',
            borderRadius: 4,
            background: 'var(--ops-bg-input)',
            color: 'var(--ops-fg-primary)',
            marginBottom: 12,
            outline: 'none',
          }}
        />

        {/* Table */}
        <div style={{ border: '1px solid var(--ops-border-subtle)', borderRadius: 4, overflow: 'hidden', marginBottom: 12 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '100px 80px 1fr 60px', padding: '6px 10px', background: 'var(--ops-bg-canvas)', borderBottom: '1px solid var(--ops-border-subtle)' }}>
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 600, color: 'var(--ops-fg-muted)' }}>命令</span>
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 600, color: 'var(--ops-fg-muted)' }}>类型</span>
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 600, color: 'var(--ops-fg-muted)' }}>限制</span>
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 600, color: 'var(--ops-fg-muted)' }}>操作</span>
          </div>
          {filtered.map((cmd, i) => (
            <div key={cmd.name} style={{ display: 'grid', gridTemplateColumns: '100px 80px 1fr 60px', padding: '6px 10px', borderBottom: '1px solid var(--ops-border-subtle)' }}>
              <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 12, color: 'var(--ops-fg-primary)' }}>{cmd.name}</span>
              <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-secondary)' }}>{typeLabel(cmd.type)}</span>
              <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>
                {cmd.allowedSubcommands.length > 0 && `子命令: ${cmd.allowedSubcommands.join(', ')}`}
                {cmd.forbiddenArgs.length > 0 && ` 禁止: ${cmd.forbiddenArgs.join(', ')}`}
                {cmd.requiresPathCheck && ' 需路径检查'}
                {!cmd.allowedSubcommands.length && !cmd.forbiddenArgs.length && !cmd.requiresPathCheck && '无'}
              </span>
              <button
                onClick={() => openEditForm(commands.indexOf(cmd))}
                style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-secondary)', background: 'none', border: 'none', cursor: 'pointer', textAlign: 'left' }}
              >
                编辑
              </button>
            </div>
          ))}
          {filtered.length === 0 && (
            <div style={{ padding: '12px 10px', fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)', textAlign: 'center' }}>
              无匹配命令
            </div>
          )}
        </div>

        {/* Add button */}
        <button
          onClick={openAddForm}
          style={{
            width: '100%',
            padding: '8px',
            borderRadius: 4,
            border: '1px dashed var(--ops-border-default)',
            background: 'transparent',
            fontFamily: 'var(--ops-font-ui)',
            fontSize: 12,
            color: 'var(--ops-fg-secondary)',
            cursor: 'pointer',
            marginBottom: 12,
          }}
        >
          + 添加新命令
        </button>

        {/* Error display */}
        {error && (
          <div style={{ padding: '8px 12px', borderRadius: 4, background: 'rgba(255,59,48,0.1)', border: '1px solid var(--ops-status-danger)', marginBottom: 12 }}>
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-status-danger)' }}>
              {error}
            </span>
          </div>
        )}

        {/* Action buttons */}
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <button
            onClick={handleResetDefaults}
            style={{
              padding: '6px 16px',
              borderRadius: 4,
              border: '1px solid var(--ops-border-default)',
              background: 'transparent',
              fontFamily: 'var(--ops-font-ui)',
              fontSize: 12,
              color: 'var(--ops-fg-secondary)',
              cursor: 'pointer',
            }}
          >
            恢复默认
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            style={{
              padding: '6px 16px',
              borderRadius: 4,
              border: 'none',
              background: 'var(--ops-fg-primary)',
              fontFamily: 'var(--ops-font-ui)',
              fontSize: 12,
              color: 'var(--ops-bg-canvas)',
              cursor: saving ? 'not-allowed' : 'pointer',
              opacity: saving ? 0.6 : 1,
            }}
          >
            {saving ? '保存中...' : '保存'}
          </button>
        </div>
      </div>

      {/* Add/Edit Form Modal */}
      {showAddForm && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div style={{ background: 'var(--ops-bg-elevated)', border: '1px solid var(--ops-border-default)', borderRadius: 8, padding: 24, width: 400, maxHeight: '80vh', overflow: 'auto' }}>
            <h3 style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 14, fontWeight: 600, color: 'var(--ops-fg-primary)', margin: '0 0 16px 0' }}>
              {editIndex !== null ? '编辑命令' : '添加命令到白名单'}
            </h3>

            <div style={{ marginBottom: 12 }}>
              <label style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', display: 'block', marginBottom: 4 }}>命令名 *</label>
              <input
                value={formName}
                onChange={e => setFormName(e.target.value)}
                disabled={editIndex !== null}
                style={{ width: '100%', padding: '6px 10px', fontFamily: 'var(--ops-font-mono)', fontSize: 12, border: '1px solid var(--ops-border-default)', borderRadius: 4, background: 'var(--ops-bg-input)', color: 'var(--ops-fg-primary)', outline: 'none' }}
              />
            </div>

            <div style={{ marginBottom: 12 }}>
              <label style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', display: 'block', marginBottom: 4 }}>类型</label>
              <div style={{ display: 'flex', gap: 12 }}>
                {(['readonly', 'readwrite', 'write'] as const).map(t => (
                  <label key={t} style={{ display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer' }}>
                    <input type="radio" name="formType" checked={formType === t} onChange={() => setFormType(t)} />
                    <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-primary)' }}>
                      {t === 'readonly' ? '只读' : t === 'readwrite' ? '读写' : '写'}
                    </span>
                  </label>
                ))}
              </div>
            </div>

            <div style={{ marginBottom: 12 }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
                <input type="checkbox" checked={formPathCheck} onChange={e => setFormPathCheck(e.target.checked)} />
                <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-primary)' }}>需要路径检查</span>
              </label>
            </div>

            <div style={{ marginBottom: 12 }}>
              <label style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', display: 'block', marginBottom: 4 }}>允许的子命令（逗号分隔，留空=全部允许）</label>
              <input
                value={formSubcommands}
                onChange={e => setFormSubcommands(e.target.value)}
                placeholder="status, restart, start"
                style={{ width: '100%', padding: '6px 10px', fontFamily: 'var(--ops-font-mono)', fontSize: 12, border: '1px solid var(--ops-border-default)', borderRadius: 4, background: 'var(--ops-bg-input)', color: 'var(--ops-fg-primary)', outline: 'none' }}
              />
            </div>

            <div style={{ marginBottom: 16 }}>
              <label style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', display: 'block', marginBottom: 4 }}>禁止的参数（逗号分隔）</label>
              <input
                value={formForbiddenArgs}
                onChange={e => setFormForbiddenArgs(e.target.value)}
                placeholder="-9, -KILL"
                style={{ width: '100%', padding: '6px 10px', fontFamily: 'var(--ops-font-mono)', fontSize: 12, border: '1px solid var(--ops-border-default)', borderRadius: 4, background: 'var(--ops-bg-input)', color: 'var(--ops-fg-primary)', outline: 'none' }}
              />
            </div>

            {error && (
              <div style={{ marginBottom: 12, fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-status-danger)' }}>
                {error}
              </div>
            )}

            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              {editIndex !== null ? (
                <button
                  onClick={() => handleDelete(editIndex)}
                  style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-status-danger)', background: 'none', border: 'none', cursor: 'pointer' }}
                >
                  删除此命令
                </button>
              ) : <div />}
              <div style={{ display: 'flex', gap: 8 }}>
                <button
                  onClick={() => setShowAddForm(false)}
                  style={{ padding: '6px 16px', borderRadius: 4, border: '1px solid var(--ops-border-default)', background: 'transparent', fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-secondary)', cursor: 'pointer' }}
                >
                  取消
                </button>
                <button
                  onClick={handleSubmitForm}
                  style={{ padding: '6px 16px', borderRadius: 4, border: 'none', background: 'var(--ops-fg-primary)', fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-bg-canvas)', cursor: 'pointer' }}
                >
                  {editIndex !== null ? '保存' : '添加'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// Default commands matching rules.go
function getDefaultCommands(): CommandEntry[] {
  return [
    { name: 'df', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'du', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'ls', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'ps', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'ss', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'ip', type: 'readonly', requiresPathCheck: false, allowedSubcommands: ['addr', 'route', 'link', 'a', 'r', 'l'], forbiddenArgs: [] },
    { name: 'free', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'uptime', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'uname', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'top', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'cat', type: 'readonly', requiresPathCheck: true, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'head', type: 'readonly', requiresPathCheck: true, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'tail', type: 'readonly', requiresPathCheck: true, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'grep', type: 'readonly', requiresPathCheck: true, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'find', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: ['-delete', '-exec'] },
    { name: 'lsof', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'wc', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'sort', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'journalctl', type: 'readonly', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: ['--rotate', '--vacuum-files', '--vacuum-time'] },
    { name: 'systemctl', type: 'readwrite', requiresPathCheck: false, allowedSubcommands: ['status', 'is-active', 'list-units', 'restart', 'start', 'stop', 'reload'], forbiddenArgs: [] },
    { name: 'truncate', type: 'write', requiresPathCheck: true, allowedSubcommands: [], forbiddenArgs: [] },
    { name: 'kill', type: 'write', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: ['-9', '-KILL', '-SIGKILL'] },
    { name: 'logrotate', type: 'write', requiresPathCheck: false, allowedSubcommands: [], forbiddenArgs: [] },
  ]
}
