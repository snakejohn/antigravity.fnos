import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import json from 'highlight.js/lib/languages/json'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import python from 'highlight.js/lib/languages/python'
import go from 'highlight.js/lib/languages/go'
import yaml from 'highlight.js/lib/languages/yaml'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'

hljs.registerLanguage('bash', bash)
hljs.registerLanguage('sh', bash)
hljs.registerLanguage('shell', bash)
hljs.registerLanguage('json', json)
hljs.registerLanguage('js', javascript)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('ts', typescript)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('py', python)
hljs.registerLanguage('python', python)
hljs.registerLanguage('go', go)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('yml', yaml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('css', css)

function escapeHtml(raw: string): string {
  return raw
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function formatInline(text: string): string {
  let out = escapeHtml(text)
  out = out.replace(/`([^`]+)`/g, '<code class="agy-inline-code">$1</code>')
  out = out.replace(/\*\*([^*]+)\*\*/g, '<strong class="font-semibold text-slate-900 dark:text-white">$1</strong>')
  out = out.replace(/\*([^*]+)\*/g, '<em>$1</em>')
  return out
}

export function renderMarkdown(source: string): string {
  if (!source) return ''
  const lines = source.replace(/\r\n/g, '\n').split('\n')
  const html: string[] = []
  let inCode = false
  let codeLang = ''
  let codeBuf: string[] = []
  let inList = false

  const closeList = () => {
    if (inList) {
      html.push('</ul>')
      inList = false
    }
  }

  for (const line of lines) {
    if (line.trim().startsWith('```')) {
      if (!inCode) {
        closeList()
        inCode = true
        codeLang = line.trim().slice(3).trim().toLowerCase()
        codeBuf = []
      } else {
        const rawCode = codeBuf.join('\n')
        let highlighted = escapeHtml(rawCode)
        if (codeLang && hljs.getLanguage(codeLang)) {
          try {
            highlighted = hljs.highlight(rawCode, { language: codeLang }).value
          } catch {
            // fallback
          }
        }
        const langBadge = codeLang ? `<div class="agy-code-lang">${escapeHtml(codeLang)}</div>` : ''
        html.push(`<div class="agy-code-wrap">${langBadge}<pre class="agy-code-pre"><code>${highlighted}</code></pre></div>`)
        inCode = false
        codeLang = ''
        codeBuf = []
      }
      continue
    }

    if (inCode) {
      codeBuf.push(line)
      continue
    }

    const trimmed = line.trim()
    if (!trimmed) {
      closeList()
      continue
    }

    if (trimmed.startsWith('#### ')) {
      closeList()
      html.push(`<h4 class="text-xs font-bold text-slate-800 dark:text-slate-100 mt-3 mb-1">${formatInline(trimmed.slice(5))}</h4>`)
    } else if (trimmed.startsWith('### ')) {
      closeList()
      html.push(`<h3 class="text-sm font-bold text-slate-800 dark:text-slate-100 mt-3 mb-1.5">${formatInline(trimmed.slice(4))}</h3>`)
    } else if (trimmed.startsWith('## ')) {
      closeList()
      html.push(`<h2 class="text-base font-bold text-slate-800 dark:text-slate-100 mt-3 mb-1.5">${formatInline(trimmed.slice(3))}</h2>`)
    } else if (trimmed.startsWith('# ')) {
      closeList()
      html.push(`<h1 class="text-lg font-bold text-slate-800 dark:text-slate-100 mt-3 mb-2">${formatInline(trimmed.slice(2))}</h1>`)
    } else if (trimmed.startsWith('> ')) {
      closeList()
      html.push(`<blockquote class="border-l-2 border-blue-500/70 pl-3 py-1 my-2 text-xs text-slate-600 dark:text-slate-300 bg-blue-500/5 rounded-r-lg">${formatInline(trimmed.slice(2))}</blockquote>`)
    } else if (/^[-*+]\s+/.test(trimmed)) {
      if (!inList) {
        html.push('<ul class="list-disc pl-5 my-1.5 space-y-1">')
        inList = true
      }
      html.push(`<li>${formatInline(trimmed.replace(/^[-*+]\s+/, ''))}</li>`)
    } else {
      closeList()
      html.push(`<p class="my-1.5">${formatInline(line)}</p>`)
    }
  }

  closeList()
  if (inCode && codeBuf.length) {
    html.push(`<pre class="agy-code-pre"><code>${escapeHtml(codeBuf.join('\n'))}</code></pre>`)
  }
  return html.join('\n')
}
