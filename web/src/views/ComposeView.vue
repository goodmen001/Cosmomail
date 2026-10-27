<!-- 
  SPDX-License-Identifier: AGPL-3.0-or-later
  Copyright (C) 2026  magiccode (魔法代码)
-->
<template>
  <div class="compose-page">
    <div class="compose-header">
      <h2 class="page-title">写邮件</h2>
      <button class="btn btn-primary" @click="goBack" :disabled="sending">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
          <path d="M10 3L5 8L10 13" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        返回
      </button>
    </div>

    <!-- 发件人选择 -->
    <div class="form-section">
      <label class="form-label">发件账号</label>
      <select v-model="form.account_id" class="form-select">
        <option value="" disabled>选择发送邮箱</option>
        <option v-for="acc in accounts" :key="acc.id" :value="acc.id">
          {{ acc.name }} ({{ acc.email }})
        </option>
      </select>
    </div>

    <!-- 收件人 -->
    <div class="form-section">
      <label class="form-label">收件人
        <label class="contact-scope-toggle">
          <input type="checkbox" v-model="allAccountContacts" @change="loadContacts" />
          包含所有邮箱账号的联系人
        </label>
      </label>
      <div class="input-group contact-input-wrap">
        <input
          v-model="toInput"
          type="email"
          class="form-input"
          placeholder="多个收件人用逗号分隔，如: a@b.com, c@d.com"
          @keydown.enter.prevent="addToRecipient"
          @blur="onRecipientBlur"
          @input="onContactInput"
          @focus="onContactInput"
        />
        <div v-if="contactSuggestions.length" class="contact-dropdown">
          <div
            v-for="ct in contactSuggestions"
            :key="ct.email"
            class="contact-item"
            @mousedown.prevent="pickContact(ct.email)"
          >
            <span class="contact-email">{{ ct.email }}</span>
            <span class="contact-meta">已发 {{ ct.use_count }} 次</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 抄送（可折叠） -->
    <details class="cc-details" :open="showCc">
      <summary class="form-label cc-toggle" @click.prevent="showCc = !showCc">
        抄送 / 密送
        <span class="toggle-icon">{{ showCc ? '−' : '+' }}</span>
      </summary>
      <div class="cc-fields">
        <div class="input-group">
          <input
            v-model="ccInput"
            class="form-input"
            placeholder="抄送（可选）"
            @keydown.enter.prevent="addCc"
            @blur="addCc"
          />
        </div>
        <div class="input-group">
          <input
            v-model="bccInput"
            class="form-input"
            placeholder="密送（可选）"
            @keydown.enter.prevent="addBcc"
            @blur="addBcc"
          />
        </div>
      </div>
    </details>

    <!-- 已选收件人标签 -->
    <div v-if="form.to.length || form.cc.length || form.bcc.length" class="recipient-tags">
      <span
        v-for="(addr, idx) in form.to"
        :key="'to-'+idx"
        class="tag tag-to"
      >
        {{ addr }}
        <button class="tag-remove" @click="removeAddr('to', idx)">×</button>
      </span>
      <span
        v-for="(addr, idx) in form.cc"
        :key="'cc-'+idx"
        class="tag tag-cc"
      >
        C: {{ addr }}
        <button class="tag-remove" @click="removeAddr('cc', idx)">×</button>
      </span>
      <span
        v-for="(addr, idx) in form.bcc"
        :key="'bcc-'+idx"
        class="tag tag-bcc"
      >
        BCC: {{ addr }}
        <button class="tag-remove" @click="removeAddr('bcc', idx)">×</button>
      </span>
    </div>

    <!-- 主题 -->
    <div class="form-section">
      <label class="form-label">主题</label>
      <input
        v-model="form.subject"
        class="form-input"
        type="text"
        placeholder="请输入邮件主题"
      />
    </div>

    <!-- 正文编辑器 -->
    <div class="form-section editor-area">
      <div class="editor-toolbar">
        <button class="toolbar-btn" title="撤销" @click="execCmd('undo')">↶</button>
        <button class="toolbar-btn" title="重做" @click="execCmd('redo')">↷</button>
        <span class="toolbar-sep"></span>
        <button class="toolbar-btn" title="加粗" @mousedown.prevent @click="execCmd('bold')"><b>B</b></button>
        <button class="toolbar-btn" title="斜体" @mousedown.prevent @click="execCmd('italic')"><i>I</i></button>
        <button class="toolbar-btn" title="下划线" @mousedown.prevent @click="execCmd('underline')"><u>U</u></button>
        <button class="toolbar-btn" title="删除线" @mousedown.prevent @click="execCmd('strikeThrough')"><s>S</s></button>
        <span class="toolbar-sep"></span>
        <select class="toolbar-select" title="字号" @change="applyFontSize($event)">
          <option value="" disabled selected>字号</option>
          <option value="1">极小</option>
          <option value="2">很小</option>
          <option value="3">小</option>
          <option value="4">标准</option>
          <option value="5">大</option>
          <option value="6">很大</option>
          <option value="7">极大</option>
        </select>
        <label class="toolbar-btn color-btn" title="文字颜色">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 20h16"/><path d="m6 16 5-12 5 12M8 12h6"/></svg>
          <input type="color" hidden @input="execCmd('foreColor', $event.target.value)" />
        </label>
        <label class="toolbar-btn color-btn" title="背景色">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="m9 11 4-8 4 8Z"/><path d="M3 21h18M6.5 17 9 11l2.5 6"/></svg>
          <input type="color" hidden @input="execCmd('hiliteColor', $event.target.value)" />
        </label>
        <span class="toolbar-sep"></span>
        <button class="toolbar-btn" title="左对齐" @mousedown.prevent @click="execCmd('justifyLeft')"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18M3 12h12M3 18h15"/></svg></button>
        <button class="toolbar-btn" title="居中" @mousedown.prevent @click="execCmd('justifyCenter')"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18M6 12h12M4 18h16"/></svg></button>
        <button class="toolbar-btn" title="右对齐" @mousedown.prevent @click="execCmd('justifyRight')"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18M9 12h12M6 18h15"/></svg></button>
        <span class="toolbar-sep"></span>
        <button class="toolbar-btn" title="无序列表" @mousedown.prevent @click="execCmd('insertUnorderedList')"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M9 6h12M9 12h12M9 18h12"/><circle cx="4" cy="6" r="1.3" fill="currentColor" stroke="none"/><circle cx="4" cy="12" r="1.3" fill="currentColor" stroke="none"/><circle cx="4" cy="18" r="1.3" fill="currentColor" stroke="none"/></svg></button>
        <button class="toolbar-btn" title="有序列表" @mousedown.prevent @click="execCmd('insertOrderedList')"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M10 6h11M10 12h11M10 18h11"/><path d="M4 6h1v4M4 10h2" stroke-width="1.5"/></svg></button>
        <button class="toolbar-btn" title="引用" @mousedown.prevent @click="execCmd('formatBlock', 'blockquote')"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 11H6a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v6a4 4 0 0 1-4 4"/><path d="M20 11h-4a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v6a4 4 0 0 1-4 4"/></svg></button>
        <span class="toolbar-sep"></span>
        <button class="toolbar-btn" title="插入链接" @mousedown.prevent @click="insertLink">🔗</button>
        <button class="toolbar-btn" title="插入图片" @mousedown.prevent @click="insertImage">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/></svg>
        </button>
        <button class="toolbar-btn" title="插入分隔线" @mousedown.prevent @click="execCmd('insertHorizontalRule')">—</button>
        <span class="toolbar-sep"></span>
        <button class="toolbar-btn" title="清除格式" @mousedown.prevent @click="execCmd('removeFormat')">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M4 7V5h12v2M8 5v14M12 19H4M15 13l6 6M21 13l-6 6"/></svg>
        </button>
        <span class="toolbar-sep"></span>
        <label class="editor-mode-toggle" title="切换为 HTML 源码编辑">
          <input type="checkbox" :checked="editorMode === 'html'" @change="toggleHtmlMode" />
          HTML 源码
        </label>
      </div>
      <!-- 富文本可视化编辑 -->
      <div
        v-if="editorMode === 'rich'"
        ref="richEditor"
        class="form-textarea editor-rich"
        contenteditable="true"
        data-placeholder="请输入邮件正文..."
        @input="syncFromRich"
      ></div>
      <!-- HTML 源码编辑 -->
      <textarea
        v-else
        v-model="form.html_body"
        class="form-textarea editor-text code-font"
        rows="14"
        placeholder="<html>...</html>"
      ></textarea>
    </div>

    <!-- 附件 -->
    <div class="form-section">
      <div class="attachment-bar">
        <label class="attachment-add-btn">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"/>
          </svg>
          添加附件
          <input type="file" multiple hidden @change="handleFileSelect" />
        </label>
        <span v-if="attachments.length" class="attachment-total">{{ attachments.length }} 个附件 · 共 {{ formatSize(totalAttachSize) }}</span>
      </div>
      <ul v-if="attachments.length" class="attachment-list">
        <li v-for="(f, i) in attachments" :key="i" class="attachment-chip">
          <span class="attach-name" :title="f.name">{{ f.name }}</span>
          <span class="attach-size">{{ formatSize(f.size) }}</span>
          <button class="attach-remove" title="移除" @click="removeAttachment(i)">✕</button>
        </li>
      </ul>
    </div>

    <!-- 发送按钮 -->
    <div class="compose-actions">
      <button class="btn btn-secondary btn-lg draft-btn" :disabled="saving" @click="handleSaveDraft">
        <template v-if="!saving">
          <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
            <rect x="3.5" y="4.5" width="13" height="13" rx="2.5" stroke="currentColor" stroke-width="1.6"/>
            <path d="M7 2v3M13 2v3M5 8.5h10M9 12l1 1 2-3" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          暂存草稿
        </template>
        <template v-else>
          <span class="spinner-sm"></span>
          暂存中...
        </template>
      </button>
      <button class="btn btn-primary btn-lg send-btn" :disabled="!canSend || sending" @click="handleSend">
        <template v-if="!sending">
          <svg width="18" height="18" viewBox="0 0 20 20" fill="none">
            <path d="M3 10L17 4L11 18L9 12L3 10Z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
            <path d="M9 12L17 4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>
          </svg>
          发送邮件
        </template>
        <template v-else>
          <span class="spinner-sm"></span>
          发送中...
        </template>
      </button>
      <span v-if="sendResult" class="send-result" :class="{ error: sendError }">
        {{ sendError ? '✗ ' + sendResult : '✓ ' + sendResult }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onActivated, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAccountStore } from '@/stores/accountStore'
import { useToast } from '@/composables/useToast'

const toast = useToast()
import { sendEmail } from '@/api/mail'
import { saveDraft, getDraftById } from '@/api/draft'
import { getContacts } from '@/api/contact'

defineOptions({ name: 'Compose' })

const router = useRouter()
const route = useRoute()
const accountStore = useAccountStore()

// 账号列表
const accounts = ref([])

// 表单数据
const form = ref({
  account_id: '',
  to: [],
  cc: [],
  bcc: [],
  subject: '',
  body: '',
  html_body: ''
})

// 输入框临时值
const toInput = ref('')
const ccInput = ref('')
const bccInput = ref('')

// ---- 附件 ----
const attachments = ref([]) // { name, size, contentType, data(base64) }
const MAX_SINGLE_SIZE = 25 * 1024 * 1024   // 25MB
const MAX_TOTAL_SIZE = 50 * 1024 * 1024    // 50MB

const totalAttachSize = computed(() => attachments.value.reduce((s, f) => s + f.size, 0))

function formatSize(bytes) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function handleFileSelect(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = '' // 允许重复选择同一文件
  for (const file of files) {
    if (file.size > MAX_SINGLE_SIZE) {
      toast.error(`「${file.name}」超过单文件 25MB 限制`)
      continue
    }
    if (totalAttachSize.value + file.size > MAX_TOTAL_SIZE) {
      toast.error('附件总大小超过 50MB 限制')
      break
    }
    const reader = new FileReader()
    reader.onload = () => {
      const base64 = String(reader.result || '').split(',')[1] || ''
      attachments.value.push({
        name: file.name,
        size: file.size,
        contentType: file.type || 'application/octet-stream',
        data: base64,
      })
    }
    reader.readAsDataURL(file)
  }
}

function removeAttachment(index) {
  attachments.value.splice(index, 1)
}

// 联系人（发信收件人记录）
const contacts = ref([])
const contactSuggestions = ref([])

// 联系人范围：默认当前发送账号的通讯录，可切换为所有账号
const allAccountContacts = ref(false)

async function loadContacts() {
  try {
    const params = { limit: 200 }
    if (!allAccountContacts.value && form.value.account_id) {
      params.account_id = form.value.account_id
    }
    const res = await getContacts(params)
    contacts.value = res.contacts || []
  } catch (e) {
    // 联系人加载失败不影响写信
  }
}

function onContactInput() {
  const kw = (toInput.value || '').trim().toLowerCase().split(/[,;，；]/).pop().trim()
  if (!kw) {
    // 无输入时展示最近联系人（前 8 个）
    contactSuggestions.value = contacts.value.slice(0, 8).filter(ct => !form.value.to.includes(ct.email))
    return
  }
  contactSuggestions.value = contacts.value
    .filter(ct => ct.email.includes(kw) && !form.value.to.includes(ct.email))
    .slice(0, 8)
}

function pickContact(email) {
  if (!form.value.to.includes(email)) form.value.to.push(email)
  toInput.value = ''
  contactSuggestions.value = []
}

function onRecipientBlur() {
  addToRecipient()
  // 延迟收起下拉，让 mousedown 选择事件先触发
  setTimeout(() => { contactSuggestions.value = [] }, 150)
}

const showCc = ref(false)
// 编辑器模式：rich = 可视化富文本（默认）, html = HTML 源码
const editorMode = ref('rich')
const richEditor = ref(null)
const sending = ref(false)
const saving = ref(false)
const sendResult = ref('')
const sendError = ref(false)
const editingDraftId = ref(null) // 正在编辑的草稿ID（如果有）

const canSend = computed(() => {
  return form.value.account_id && form.value.to.length > 0 && form.value.subject.trim() !== ''
})

function goBack() {
  if (route.query.from) {
    router.push(route.query.from)
  } else {
    router.push('/mails')
  }
}

function parseEmails(str) {
  if (!str) return []
  return str.split(/[,;，；]/).map(s => s.trim().toLowerCase()).filter(s => s.includes('@'))
}

function addToRecipient() {
  const emails = parseEmails(toInput.value)
  for (const e of emails) {
    if (!form.value.to.includes(e)) form.value.to.push(e)
  }
  toInput.value = ''
}

function addCc() {
  const emails = parseEmails(ccInput.value)
  for (const e of emails) {
    if (!form.value.cc.includes(e) && !form.value.to.includes(e)) form.value.cc.push(e)
  }
  ccInput.value = ''
}

function addBcc() {
  const emails = parseEmails(bccInput.value)
  for (const e of emails) {
    if (!form.value.bcc.includes(e) && !form.value.to.includes(e) && !form.value.cc.includes(e)) form.value.bcc.push(e)
  }
  bccInput.value = ''
}

function removeAddr(type, index) {
  form.value[type].splice(index, 1)
}

async function handleSend() {
  if (!canSend.value) return

  sending.value = true
  sendResult.value = ''
  sendError.value = false

  try {
    const payload = {
      account_id: parseInt(form.value.account_id),
      to: form.value.to,
      subject: form.value.subject,
      body: form.value.body || ''
    }

    if (form.value.cc.length > 0) payload.cc = form.value.cc
    if (form.value.bcc.length > 0) payload.bcc = form.value.bcc
    // 富文本/HTML 模式都发送 html_body；纯文本回退 body
    if (editorMode.value === 'rich') {
      syncFromRich()
    }
    if (form.value.html_body) payload.html_body = form.value.html_body
    if (attachments.value.length > 0) {
      payload.attachments = attachments.value.map(f => ({
        filename: f.name,
        content_type: f.contentType,
        data: f.data,
      }))
    }

    await sendEmail(payload)
    sendResult.value = '邮件已成功发送'
    sendError.value = false

    // 发送成功后删除对应的草稿（如果有）
    if (editingDraftId.value) {
      try {
        const { deleteDraft } = await import('@/api/draft')
        await deleteDraft(editingDraftId.value)
        editingDraftId.value = null
      } catch (_) { /* 忽略草稿删除错误 */ }
    }

    // 发送成功后跳转到已发送页面
    setTimeout(() => router.push('/sent'), 1500)
  } catch (e) {
    const msg = e.response?.data?.detail || e.message || '未知错误'
    sendResult.value = `发送失败: ${msg}`
    sendError.value = true
  } finally {
    sending.value = false
  }
}

async function handleSaveDraft() {
  saving.value = true

  try {
    const payload = {
      id: editingDraftId.value,
      account_id: form.value.account_id ? parseInt(form.value.account_id) : 0,
      to: form.value.to,
      cc: form.value.cc,
      bcc: form.value.bcc,
      subject: form.value.subject,
      body: form.value.body || '',
      html_body: editorMode.value === 'html' ? form.value.html_body : (form.value.html_body || ''),
    }

    // 更新已有草稿时需要传 id 字段让后端走更新逻辑
    if (editingDraftId.value) {
      payload.id = editingDraftId.value
    }

    const res = await saveDraft(payload)

    if (!editingDraftId.value && res.data?.id) {
      editingDraftId.value = res.data.id
    }

    sendResult.value = '草稿已保存'
    sendError.value = false

    setTimeout(() => { sendResult.value = '' }, 2000)
  } catch (e) {
    const msg = e.response?.data?.detail || e.message || '未知错误'
    sendResult.value = `暂存失败: ${msg}`
    sendError.value = true
  } finally {
    saving.value = false
  }
}

// ---- 富文本编辑器 ----
function applyFontSize(e) {
  const v = e.target.value
  e.target.value = '' // 复位，便于重复选择同一字号
  if (v) execCmd('fontSize', v)
}

function execCmd(command, value = null) {
  if (editorMode.value !== 'rich') {
    toast.warning('请切换回富文本模式使用格式工具')
    return
  }
  richEditor.value?.focus()
  document.execCommand(command, false, value)
  syncFromRich()
}

// contenteditable 内容 → form.html_body（同时生成纯文本摘要到 body）
function syncFromRich() {
  if (editorMode.value !== 'rich' || !richEditor.value) return
  form.value.html_body = richEditor.value.innerHTML
  // 纯文本版本（去掉标签与引言样式）供不支持 HTML 的客户端回显
  const div = document.createElement('div')
  div.innerHTML = richEditor.value.innerHTML
  form.value.body = div.textContent || ''
}

// HTML 源码 ↔ 富文本切换
function toggleHtmlMode(e) {
  if (e.target.checked) {
    // rich → html：先把当前内容同步到 html_body
    if (editorMode.value === 'rich') syncFromRich()
    editorMode.value = 'html'
  } else {
    editorMode.value = 'rich'
    nextTick(() => {
      if (richEditor.value) {
        richEditor.value.innerHTML = form.value.html_body || ''
      }
    })
  }
}

function insertLink() {
  const url = prompt('请输入链接地址:', 'https://')
  if (url) execCmd('createLink', url)
}

function insertImage() {
  const url = prompt('请输入图片地址（URL）:', 'https://')
  if (url) execCmd('insertImage', url)
}

// HTML 转义（用于回复引言中的纯文本原文）
function escapeHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

// 切换发送账号 → 默认加载该账号通讯录
watch(() => form.value.account_id, () => {
  if (!allAccountContacts.value) loadContacts()
})

onMounted(async () => {
  await accountStore.fetchAccounts()
  accounts.value = accountStore.accounts || []
  loadContacts() // 加载发信收件人记录（联系人建议）

  // 从路由参数预填信息（回复）
  if (route.query.replyTo) form.value.to = [route.query.replyTo]
  if (route.query.replySubject) form.value.subject = route.query.replySubject
  if (route.query.accountId) form.value.account_id = route.query.accountId
  // 回复引用原文（由 MailReaderView 在跳转前写入 sessionStorage）
  try {
    const rawQuote = sessionStorage.getItem('novamail-reply-quote')
    if (rawQuote && route.query.replyTo) {
      const quote = JSON.parse(rawQuote)
      sessionStorage.removeItem('novamail-reply-quote')
      const quoteHtml = quote.htmlBody
        ? `<br><br><div class="novamail-quote" style="margin:0;padding-left:12px;border-left:3px solid #ddd;color:#666;"><div>${quote.header}</div><blockquote style="margin:8px 0 0;">${quote.htmlBody}</blockquote></div>`
        : `<br><br><div class="novamail-quote" style="margin:0;padding-left:12px;border-left:3px solid #ddd;color:#666;"><div>${quote.header}</div><blockquote style="margin:8px 0 0;white-space:pre-wrap;">${escapeHtml(quote.textBody || '')}</blockquote></div>`
      form.value.html_body = quoteHtml
      form.value.body = `\n\n${quote.header}\n${(quote.textBody || '').split('\n').map(l => '> ' + l).join('\n')}`
      editorMode.value = 'rich'
      nextTick(() => {
        if (richEditor.value) richEditor.value.innerHTML = form.value.html_body
      })
    }
  } catch (_) { /* 引用失败不影响回复 */ }

  // 从草稿加载
  if (route.query.draftId) {
    editingDraftId.value = parseInt(route.query.draftId)
    try {
      const draft = await getDraftById(editingDraftId.value)
      if (draft) {
        form.value.account_id = String(draft.account_id)
        try { form.value.to = JSON.parse(draft.to) } catch { form.value.to = [] }
        try { form.value.cc = JSON.parse(draft.cc || '[]') } catch { form.value.cc = [] }
        form.value.subject = draft.subject || ''
        form.value.body = draft.body || ''
        if (draft.html_body) {
          form.value.html_body = draft.html_body || ''
          editorMode.value = 'rich'
          nextTick(() => {
            if (richEditor.value) richEditor.value.innerHTML = form.value.html_body
          })
        }
      }
    } catch (e) {
      console.error('加载草稿失败:', e)
      editingDraftId.value = null
    }
  }
})

onActivated(() => {
  accountStore.fetchAccounts()
})
</script>

<style scoped>
/* ---- 附件 ---- */
.attachment-bar {
  display: flex;
  align-items: center;
  gap: 12px;
}
.attachment-add-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border: 1px dashed var(--border-color, #d1d5db);
  border-radius: 8px;
  color: var(--text-secondary, #6b7280);
  font-size: 13px;
  cursor: pointer;
  transition: all var(--transition-fast, 0.15s);
  -webkit-tap-highlight-color: transparent;
}
.attachment-add-btn:hover {
  border-color: var(--primary-color, #4F6EF7);
  color: var(--primary-color, #4F6EF7);
  background: rgba(79, 110, 247, 0.04);
}
.attachment-total { font-size: 12px; color: var(--text-tertiary, #9ca3af); }
.attachment-list {
  list-style: none;
  margin: 10px 0 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.attachment-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 260px;
  padding: 5px 10px;
  border-radius: 16px;
  background: var(--bg-tertiary, #f3f4f6);
  font-size: 12px;
}
.attach-name {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary, #111827);
}
.attach-size { color: var(--text-tertiary, #9ca3af); flex-shrink: 0; }
.attach-remove {
  border: none;
  background: none;
  color: var(--text-tertiary, #9ca3af);
  cursor: pointer;
  padding: 0 2px;
  font-size: 13px;
  line-height: 1;
}
.attach-remove:hover { color: #ef4444; }

.compose-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px 32px 48px;
}

.compose-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.page-title {
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-bold);
  color: var(--text-primary);
  margin: 0;
}

.form-section {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.form-select,
.form-input,
.form-textarea {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  color: var(--text-primary);
  font-family: inherit;
  font-size: var(--font-size-base);
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
  outline: none;
}
.form-select:focus,
.form-input:focus,
.form-textarea:focus {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px var(--mail-unread-bg);
}

/* ---- 富文本编辑器 ---- */
.editor-rich {
  min-height: 280px;
  max-height: 520px;
  overflow-y: auto;
  cursor: text;
  line-height: 1.6;
}
.editor-rich:empty::before {
  content: attr(data-placeholder);
  color: var(--text-tertiary, #9ca3af);
  pointer-events: none;
}
.editor-rich blockquote {
  margin: 8px 0;
  padding: 4px 12px;
  border-left: 3px solid var(--border-color, #d1d5db);
  color: var(--text-secondary, #6b7280);
}
.editor-rich img { max-width: 100%; }
.toolbar-select {
  padding: 4px 6px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
}
.color-btn { position: relative; }
.color-btn input[type="color"] { position: absolute; inset: 0; opacity: 0; cursor: pointer; }

.form-textarea { resize: vertical; line-height: 1.7; }

/* 抄送/密送折叠区 */
.cc-details {
  margin-bottom: 16px;
  border: none;
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  overflow: hidden;
}
.cc-details summary {
  list-style: none;
  cursor: pointer;
  user-select: none;
}
.cc-details summary::-webkit-details-marker { display: none; }

.cc-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: var(--radius-md);
  transition: background var(--transition-fast);
}
.cc-toggle:hover { background: var(--bg-hover); }

.toggle-icon {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--primary-100);
  border-radius: var(--radius-full);
  font-size: 14px;
  color: var(--primary-500);
}

.cc-fields { padding: 0 14px 10px; display: flex; flex-direction: column; gap: 8px; }

/* 收件人标签 */
.recipient-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  min-height: 40px;
}

.tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: var(--font-size-sm);
}
.tag-to { background: var(--mail-unread-bg); color: var(--primary-600); }
.tag-cc { background: rgba(245, 158, 11, 0.12); color: #b45309; }
.tag-bcc { background: rgba(107, 114, 128, 0.12); color: #374151; }

.tag-remove {
  background: none;
  border: none;
  color: inherit;
  opacity: 0.55;
  cursor: pointer;
  font-size: 15px;
  padding: 0 2px;
  line-height: 1;
  transition: opacity var(--transition-fast);
}
.tag-remove:hover { opacity: 1; }

/* 编辑器 */
.editor-area {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}
.editor-area:focus-within {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px var(--mail-unread-bg);
}

.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-light);
}

.toolbar-btn {
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: none;
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.toolbar-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  border-color: var(--border-color);
}

.toolbar-sep { width: 1px; height: 22px; background: var(--border-light); margin: 0 6px; }

.editor-mode-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  cursor: pointer;
  margin-left: auto;
}
.editor-mode-toggle input[type="checkbox"] {
  accent-color: var(--primary-500);
  width: 14px;
  height: 14px;
}

.editor-text { border: none !important; border-radius: 0 !important; padding: 14px 16px !important; min-height: 300px; }
.editor-text:focus { box-shadow: none !important; border-color: transparent !important; }
.code-font { font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace; font-size: 13px; line-height: 1.6; }

/* 底部操作栏 */
.compose-actions {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--border-light);
}

.btn-lg { padding: 12px 28px; font-size: var(--font-size-base); font-weight: var(--font-weight-semibold); }

.send-btn {
  gap: 8px;
  white-space: nowrap;
}

.draft-btn {
  gap: 8px;
  white-space: nowrap;
}

.spinner-sm {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin .6s linear infinite;
}

.send-result {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}
.send-result.error { color: var(--error); }
.send-result:not(.error) { color: var(--success); }

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 响应式 */
@media (max-width: 640px) {
  .compose-page { padding: 16px; }
  .compose-header { flex-direction: rows; gap: 12px; align-items: flex-start; }
  .btn-lg { width: 100%; justify-content: center; }
}

.contact-input-wrap { position: relative; }
.contact-scope-toggle { display: inline-flex; align-items: center; gap: 4px; margin-left: 10px; font-size: 12px; font-weight: normal; color: var(--text-secondary, #888); cursor: pointer; }

.contact-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  z-index: 50;
  margin-top: 4px;
  background: var(--bg-primary, #fff);
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  max-height: 240px;
  overflow-y: auto;
}

.contact-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  font-size: 13px;
}

.contact-item:hover { background: var(--bg-hover, #f5f5f5); }

.contact-email { color: var(--text-primary, #333); }

.contact-meta { color: var(--text-tertiary, #999); font-size: 12px; }
</style>
