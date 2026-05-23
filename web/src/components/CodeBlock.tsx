import { type FC, useState, useCallback, useMemo } from 'react'

interface CodeBlockProps {
  language: string
  code: string
}

const TOKEN_COLORS = {
  keyword: '#569CD6',
  string: '#CE9178',
  comment: '#6A9955',
  number: '#B5CEA8',
} as const

const KEYWORDS: Record<string, Set<string>> = {
  js: new Set(['const','let','var','function','return','if','else','for','while','class','import','export','from','default','new','this','typeof','async','await','try','catch','throw','switch','case','break','continue','null','undefined','true','false']),
  ts: new Set(['const','let','var','function','return','if','else','for','while','class','import','export','from','default','new','this','typeof','async','await','try','catch','throw','switch','case','break','continue','null','undefined','true','false','interface','type','enum','extends','implements','readonly','private','public','protected','abstract','as','keyof','infer','never','unknown','any','void']),
  go: new Set(['func','return','if','else','for','range','switch','case','break','continue','package','import','var','const','type','struct','interface','map','chan','go','defer','select','nil','true','false','make','append','len','cap','error','string','int','bool','byte','float64']),
  python: new Set(['def','return','if','elif','else','for','while','class','import','from','as','try','except','raise','with','pass','break','continue','lambda','yield','None','True','False','and','or','not','in','is','global','nonlocal','async','await']),
  sh: new Set(['if','then','else','elif','fi','for','do','done','while','until','case','esac','function','return','local','export','echo','exit','cd','ls','grep','awk','sed','cat','rm','cp','mv','mkdir','chmod','chown','sudo','apt','yum','dnf','systemctl','journalctl']),
}

function getKeywords(lang: string): Set<string> {
  const normalized = lang.toLowerCase().replace(/^(typescript|javascript|tsx|jsx)$/, (m) => {
    if (m === 'javascript' || m === 'jsx') return 'js'
    return 'ts'
  }).replace(/^(bash|shell|zsh)$/, 'sh').replace(/^(golang)$/, 'go').replace(/^(py|python3)$/, 'python')
  return KEYWORDS[normalized] || KEYWORDS.js
}

function tokenize(code: string, lang: string): Array<{ text: string; color?: string }> {
  const keywords = getKeywords(lang)
  const tokens: Array<{ text: string; color?: string }> = []
  let i = 0

  while (i < code.length) {
    // Single-line comment
    if ((code[i] === '/' && code[i + 1] === '/') || (code[i] === '#' && lang !== 'python' ? false : code[i] === '#')) {
      if (code[i] === '/' && code[i + 1] === '/') {
        const end = code.indexOf('\n', i)
        const slice = end === -1 ? code.slice(i) : code.slice(i, end)
        tokens.push({ text: slice, color: TOKEN_COLORS.comment })
        i += slice.length
        continue
      }
      if (code[i] === '#') {
        const end = code.indexOf('\n', i)
        const slice = end === -1 ? code.slice(i) : code.slice(i, end)
        tokens.push({ text: slice, color: TOKEN_COLORS.comment })
        i += slice.length
        continue
      }
    }

    // Multi-line comment
    if (code[i] === '/' && code[i + 1] === '*') {
      const end = code.indexOf('*/', i + 2)
      const slice = end === -1 ? code.slice(i) : code.slice(i, end + 2)
      tokens.push({ text: slice, color: TOKEN_COLORS.comment })
      i += slice.length
      continue
    }

    // Strings
    if (code[i] === '"' || code[i] === "'" || code[i] === '`') {
      const quote = code[i]
      let j = i + 1
      while (j < code.length && code[j] !== quote) {
        if (code[j] === '\\') j++
        j++
      }
      const slice = code.slice(i, j + 1)
      tokens.push({ text: slice, color: TOKEN_COLORS.string })
      i = j + 1
      continue
    }

    // Numbers
    if (/\d/.test(code[i]) && (i === 0 || /[\s([\]{},;:=+\-*/<>!&|^~%]/.test(code[i - 1]))) {
      let j = i
      while (j < code.length && /[\d.xXa-fA-F_eE+-]/.test(code[j])) j++
      tokens.push({ text: code.slice(i, j), color: TOKEN_COLORS.number })
      i = j
      continue
    }

    // Words (potential keywords)
    if (/[a-zA-Z_$]/.test(code[i])) {
      let j = i
      while (j < code.length && /[a-zA-Z0-9_$]/.test(code[j])) j++
      const word = code.slice(i, j)
      tokens.push({ text: word, color: keywords.has(word) ? TOKEN_COLORS.keyword : undefined })
      i = j
      continue
    }

    // Other characters
    tokens.push({ text: code[i] })
    i++
  }

  return tokens
}

export const CodeBlock: FC<CodeBlockProps> = ({ language, code }) => {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }, [code])

  const highlighted = useMemo(() => tokenize(code, language), [code, language])

  return (
    <div style={{ position: 'relative', borderRadius: 6, background: 'var(--ops-bg-input)', margin: '8px 0', overflow: 'hidden' }}>
      {/* Header bar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '4px 10px', background: 'var(--ops-bg-elevated)' }}>
        <span style={{ fontSize: 10, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)' }}>
          {language}
        </span>
        <div onClick={handleCopy} style={{ display: 'flex', alignItems: 'center', gap: 3, cursor: 'pointer' }}>
          <span className="material-symbols-outlined" style={{ fontSize: 12, color: copied ? 'var(--ops-status-ok)' : 'var(--ops-fg-muted)' }}>
            {copied ? 'check' : 'content_copy'}
          </span>
          <span style={{ fontSize: 10, color: 'var(--ops-fg-muted)' }}>
            {copied ? '已复制' : '复制'}
          </span>
        </div>
      </div>
      {/* Code content */}
      <pre style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 12, padding: '10px 12px', margin: 0, overflow: 'auto', lineHeight: 1.5 }}>
        <code>
          {highlighted.map((token, i) => (
            token.color
              ? <span key={i} style={{ color: token.color }}>{token.text}</span>
              : <span key={i} style={{ color: 'var(--ops-fg-secondary)' }}>{token.text}</span>
          ))}
        </code>
      </pre>
    </div>
  )
}
