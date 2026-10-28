<!--
  SPDX-License-Identifier: AGPL-3.0-or-later
  Copyright (C) 2026  magiccode (魔法代码)
-->
<template>
  <div class="folders-view">
    <!-- 账号选择 -->
    <div class="account-row card">
      <label>邮箱账号：</label>
      <select v-model="selectedAccountId" @change="loadAll">
        <option v-for="a in accountStore.accounts" :key="a.id" :value="a.id">
          {{ a.name || a.email }}（{{ a.email }}）
        </option>
      </select>
      <p class="hint">目录按邮箱账号独立管理，仅归档本地收到的邮件，不影响服务器端邮件。</p>
    </div>

    <template v-if="selectedAccountId">
      <!-- 目录列表 -->
      <div class="section card">
        <div class="section-head">
          <h3>目录</h3>
          <button class="btn btn-primary" @click="openFolderForm(null)">＋ 新建目录</button>
        </div>
        <p v-if="folders.length === 0" class="empty">还没有自定义目录</p>
        <ul v-else class="folder-list">
          <li v-for="f in folders" :key="f.id" class="folder-row">
            <span class="folder-icon">📁</span>
            <span class="folder-name">{{ f.name }}</span>
            <span class="folder-slug">{{ f.slug }}</span>
            <span class="row-actions">
              <button class="btn-link" @click="openFolderForm(f)">改名</button>
              <button class="btn-link danger" @click="removeFolder(f)">删除</button>
            </span>
          </li>
        </ul>
      </div>

      <!-- 规则列表 -->
      <div class="section card">
        <div class="section-head">
          <h3>归档规则</h3>
          <button class="btn btn-primary" @click="openRuleForm(null)" :disabled="folders.length === 0">＋ 新建规则</button>
        </div>
        <p v-if="folders.length === 0" class="empty">请先创建目录，再配置规则</p>
        <p v-else-if="rules.length === 0" class="empty">还没有规则，新邮件将全部进入收件箱</p>
        <ul v-else class="rule-list">
          <li v-for="r in rules" :key="r.id" class="rule-row" :class="{ disabled: !r.enabled }">
            <span class="rule-priority">P{{ r.priority }}</span>
            <span class="rule-desc">{{ ruleText(r) }}</span>
            <span class="rule-target">→ 📁 {{ folderName(r.folder_slug) }}</span>
            <label class="switch" :title="r.enabled ? '已启用' : '已禁用'">
              <input type="checkbox" :checked="r.enabled" @change="toggleRule(r, $event.target.checked)" />
            </label>
            <span class="row-actions">
              <button class="btn-link" @click="openRuleForm(r)">编辑</button>
              <button class="btn-link danger" @click="removeRule(r)">删除</button>
            </span>
          </li>
        </ul>
      </div>
    </template>

    <!-- 目录表单弹窗 -->
    <div v-if="showFolderForm" class="modal-mask" @click.self="showFolderForm = false">
      <div class="modal card">
        <h3>{{ editingFolder ? '重命名目录' : '新建目录' }}</h3>
        <input v-model.trim="folderForm.name" placeholder="目录名称，如：账单 / 订阅" maxlength="64" @keyup.enter="saveFolder" />
        <div class="modal-actions">
          <button class="btn" @click="showFolderForm = false">取消</button>
          <button class="btn btn-primary" @click="saveFolder">保存</button>
        </div>
      </div>
    </div>

    <!-- 规则表单弹窗 -->
    <div v-if="showRuleForm" class="modal-mask" @click.self="showRuleForm = false">
      <div class="modal card">
        <h3>{{ editingRule ? '编辑规则' : '新建规则' }}</h3>
        <label class="field">
          <span>规则名称（可选）</span>
          <input v-model.trim="ruleForm.name" maxlength="64" placeholder="如：账单邮件归档" />
        </label>
        <label class="field">
          <span>当邮件的</span>
          <select v-model="ruleForm.field">
            <option value="from">发件人</option>
            <option value="subject">主题</option>
            <option value="to">收件人</option>
            <option value="body">正文</option>
            <option value="has_attachment">附件</option>
          </select>
        </label>
        <label class="field">
          <span>匹配方式</span>
          <select v-model="ruleForm.operator">
            <option value="contains">包含</option>
            <option value="not_contains">不包含</option>
            <option value="equals">等于</option>
            <option value="starts_with">开头为</option>
            <option value="is_set" v-if="ruleForm.field === 'has_attachment'">存在</option>
          </select>
        </label>
        <label class="field" v-if="ruleForm.operator !== 'is_set'">
          <span>匹配值</span>
          <input v-model.trim="ruleForm.value" :placeholder="fieldPlaceholder" />
        </label>
        <label class="field">
          <span>归入目录</span>
          <select v-model="ruleForm.folder_slug">
            <option v-for="f in folders" :key="f.slug" :value="f.slug">{{ f.name }}</option>
          </select>
        </label>
        <label class="field">
          <span>优先级（越大越先匹配）</span>
          <input type="number" v-model.number="ruleForm.priority" min="0" max="999" />
        </label>
        <label class="field switch-field">
          <span>启用</span>
          <input type="checkbox" v-model="ruleForm.enabled" />
        </label>
        <div class="modal-actions">
          <button class="btn" @click="showRuleForm = false">取消</button>
          <button class="btn btn-primary" @click="saveRule">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineOptions({ name: 'MailFolders' })
import { ref, computed, onMounted } from 'vue'
import { useAccountStore } from '@/stores/accountStore'
import { useToast } from '@/composables/useToast'
import {
  getFolders, createFolder, updateFolder, deleteFolder,
  getRules, createRule, updateRule, deleteRule,
} from '@/api/mailFolder'

const toast = useToast()
const accountStore = useAccountStore()

const selectedAccountId = ref('')
const folders = ref([])
const rules = ref([])

const showFolderForm = ref(false)
const editingFolder = ref(null)
const folderForm = ref({ name: '' })

const showRuleForm = ref(false)
const editingRule = ref(null)
const emptyRuleForm = () => ({ name: '', field: 'from', operator: 'contains', value: '', folder_slug: '', priority: 0, enabled: true })
const ruleForm = ref(emptyRuleForm())

const fieldPlaceholder = computed(() => {
  switch (ruleForm.value.field) {
    case 'from': return '如：noreply@example.com'
    case 'subject': return '如：账单'
    case 'to': return '如：team@corp.com'
    case 'body': return '正文包含的关键词'
    default: return ''
  }
})

function ruleText(r) {
  const fieldNames = { from: '发件人', subject: '主题', to: '收件人', body: '正文', has_attachment: '附件' }
  const opNames = { contains: '包含', not_contains: '不包含', equals: '等于', starts_with: '开头为', is_set: '存在' }
  const cond = r.operator === 'is_set'
    ? `${fieldNames[r.field] || r.field} ${opNames[r.operator] || r.operator}`
    : `${fieldNames[r.field] || r.field} ${opNames[r.operator] || r.operator}「${r.value}」`
  return r.name ? `${r.name}（${cond}）` : cond
}

function folderName(slug) {
  const f = folders.value.find(x => x.slug === slug)
  return f ? f.name : slug
}

async function loadAll() {
  if (!selectedAccountId.value) return
  try {
    const [fr, rr] = await Promise.all([getFolders(selectedAccountId.value), getRules(selectedAccountId.value)])
    folders.value = fr.folders || []
    rules.value = rr.rules || []
  } catch (e) {
    toast.error('加载目录/规则失败: ' + e.message)
  }
}

// --- 目录 ---
function openFolderForm(f) {
  editingFolder.value = f
  folderForm.value = { name: f ? f.name : '' }
  showFolderForm.value = true
}

async function saveFolder() {
  if (!folderForm.value.name) { toast.warning('请输入目录名称'); return }
  try {
    if (editingFolder.value) {
      await updateFolder(editingFolder.value.id, { name: folderForm.value.name, sort_order: editingFolder.value.sort_order })
      toast.success('目录已更新')
    } else {
      await createFolder(selectedAccountId.value, { name: folderForm.value.name, sort_order: folders.value.length })
      toast.success('目录已创建')
    }
    showFolderForm.value = false
    loadAll()
  } catch (e) {
    toast.error('保存失败: ' + e.message)
  }
}

async function removeFolder(f) {
  if (!confirm(`确定删除目录「${f.name}」吗？\n该目录下的邮件将退回收件箱，引用此目录的规则也会一并删除。`)) return
  try {
    await deleteFolder(f.id)
    toast.success('目录已删除')
    loadAll()
  } catch (e) {
    toast.error('删除失败: ' + e.message)
  }
}

// --- 规则 ---
function openRuleForm(r) {
  editingRule.value = r
  ruleForm.value = r ? {
    name: r.name, field: r.field, operator: r.operator, value: r.value,
    folder_slug: r.folder_slug, priority: r.priority, enabled: r.enabled,
  } : emptyRuleForm()
  if (!ruleForm.value.folder_slug && folders.value.length) {
    ruleForm.value.folder_slug = folders.value[0].slug
  }
  showRuleForm.value = true
}

async function saveRule() {
  if (ruleForm.value.operator !== 'is_set' && !ruleForm.value.value) {
    toast.warning('请输入匹配值'); return
  }
  if (!ruleForm.value.folder_slug) { toast.warning('请选择目标目录'); return }
  try {
    if (editingRule.value) {
      await updateRule(editingRule.value.id, ruleForm.value)
      toast.success('规则已更新')
    } else {
      await createRule(selectedAccountId.value, ruleForm.value)
      toast.success('规则已创建，之后同步的新邮件将按规则归档')
    }
    showRuleForm.value = false
    loadAll()
  } catch (e) {
    toast.error('保存失败: ' + e.message)
  }
}

async function toggleRule(r, enabled) {
  try {
    await updateRule(r.id, { ...r, enabled })
    r.enabled = enabled
  } catch (e) {
    toast.error('操作失败: ' + e.message)
    loadAll()
  }
}

async function removeRule(r) {
  if (!confirm(`确定删除规则「${r.name || ruleText(r)}」吗？`)) return
  try {
    await deleteRule(r.id)
    toast.success('规则已删除')
    loadAll()
  } catch (e) {
    toast.error('删除失败: ' + e.message)
  }
}

onMounted(async () => {
  if (accountStore.accounts.length === 0) {
    await accountStore.fetchAccounts()
  }
  if (accountStore.accounts.length > 0) {
    selectedAccountId.value = accountStore.accounts[0].id
    loadAll()
  }
})
</script>

<style scoped>
.folders-view { display: flex; flex-direction: column; gap: 16px; max-width: 900px; }
.card { background: var(--bg-primary); border: 1px solid var(--border-light); border-radius: var(--radius-lg, 12px); padding: 16px 20px; }
.account-row select { padding: 6px 10px; border: 1px solid var(--border-color); border-radius: 8px; background: var(--bg-primary); color: var(--text-primary); }
.hint { margin: 8px 0 0; font-size: var(--font-size-sm); color: var(--text-secondary); }
.section-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.section-head h3 { margin: 0; font-size: var(--font-size-md); font-weight: var(--font-weight-semibold); }
.empty { color: var(--text-secondary); font-size: var(--font-size-base); text-align: center; padding: 16px 0; }
.btn { padding: 6px 14px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-primary); color: var(--text-primary); cursor: pointer; font-size: var(--font-size-base); }
.btn-primary { background: var(--primary-color, #4F6EF7); border-color: var(--primary-color, #4F6EF7); color: #fff; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-link { background: none; border: none; color: var(--primary-color, #4F6EF7); cursor: pointer; font-size: var(--font-size-sm); padding: 2px 6px; }
.btn-link.danger { color: #ef4444; }
.folder-list, .rule-list { list-style: none; margin: 0; padding: 0; }
.folder-row, .rule-row { display: flex; align-items: center; gap: 10px; padding: 10px 4px; border-bottom: 1px solid var(--border-light); font-size: var(--font-size-base); flex-wrap: wrap; }
.folder-row:last-child, .rule-row:last-child { border-bottom: none; }
.rule-row.disabled { opacity: 0.55; }
.folder-slug { color: var(--text-secondary); font-size: var(--font-size-sm); font-family: monospace; }
.row-actions { margin-left: auto; display: flex; gap: 4px; }
.rule-priority { background: var(--primary-50, rgba(79,110,247,0.1)); color: var(--primary-color, #4F6EF7); border-radius: 6px; padding: 2px 6px; font-size: var(--font-size-xs); font-weight: var(--font-weight-semibold); }
.rule-desc { min-width: 0; word-break: break-all; }
.rule-target { color: var(--text-secondary); white-space: nowrap; }
.switch { display: inline-flex; align-items: center; }
.modal-mask { position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { width: min(420px, calc(100vw - 32px)); max-height: 85vh; overflow-y: auto; }
.modal h3 { margin: 0 0 14px; font-size: var(--font-size-md); }
.modal input, .modal select { width: 100%; box-sizing: border-box; padding: 8px 10px; border: 1px solid var(--border-color); border-radius: 8px; background: var(--bg-primary); color: var(--text-primary); font-size: var(--font-size-base); margin-top: 4px; }
.field { display: block; margin-bottom: 12px; font-size: var(--font-size-base); color: var(--text-secondary); }
.switch-field input { width: auto; }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 6px; }
@media (max-width: 768px) {
  .folders-view { gap: 10px; }
  .card { padding: 12px 14px; }
}
</style>
