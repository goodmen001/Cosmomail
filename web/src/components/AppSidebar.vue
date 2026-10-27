<!-- 
  SPDX-License-Identifier: AGPL-3.0-or-later
  Copyright (C) 2026  magiccode (魔法代码)
-->
<template>
  <aside class="app-sidebar" :class="{ collapsed, open: mobileOpen }">
    <!-- Logo 区域 -->
    <div class="sidebar-logo">
      <button class="logo-btn" @click="$emit('navigate', '/mails')">
        <img class="sidebar-brand-icon" src="/icons/icon-64x64.png" alt="" />
        <span v-show="!collapsed" class="logo-text">Cosmo Mail</span>
      </button>
      <button v-if="isMobile && !collapsed" class="btn-icon btn-ghost close-btn" @click="mobileOpen = false">
        <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
          <path d="M4 4L14 14M14 4L4 14" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
        </svg>
      </button>
    </div>

    <!-- 导航菜单 -->
    <nav class="sidebar-nav">
      <!-- 邮件列表 (主页) -->
      <router-link
        to="/mails"
        class="nav-item"
        active-class="active"
        @click="handleNavClick('/mails')"
      >
        <span class="nav-icon">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <rect x="2.5" y="5" width="15" height="11" rx="2" stroke="currentColor" stroke-width="1.6"/>
            <path d="M2.5 6.5L10 11L17.5 6.5" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
          </svg>
        </span>
        <transition name="fade-text">
          <span v-show="!collapsed" class="nav-label">收件箱</span>
        </transition>
        <span v-show="!collapsed" class="nav-badge" v-if="unreadCount > 0">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
      </router-link>

      <!-- 已发送 -->
      <router-link
        to="/sent"
        class="nav-item"
        active-class="active"
        @click="handleNavClick('/sent')"
      >
        <span class="nav-icon">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M3 10L17 4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M3 10L17 16" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M15 5.5V3M15 14.5V17" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>
          </svg>
        </span>
        <transition name="fade-text">
          <span v-show="!collapsed" class="nav-label">已发送</span>
        </transition>
      </router-link>

      <!-- 草稿 -->
      <router-link
        to="/drafts"
        class="nav-item"
        active-class="active"
        @click="handleNavClick('/drafts')"
      >
        <span class="nav-icon">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <rect x="3.5" y="4.5" width="13" height="13" rx="2.5" stroke="currentColor" stroke-width="1.6"/>
            <path d="M7 4V3a1 1 0 011-1h4a1 1 0 011 1v1M7 10l1.5 1.5L13 7" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </span>
        <transition name="fade-text">
          <span v-show="!collapsed" class="nav-label">草稿</span>
        </transition>
      </router-link>

      <!-- 写邮件 (次要操作) -->
      <router-link
        to="/compose"
        class="nav-item compose-item"
        active-class="active"
        @click="handleNavClick('/compose')"
      >
        <span class="nav-icon">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M3 10L17 4L11 18L9 12L3 10Z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
            <path d="M9 12L17 4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>
          </svg>
        </span>
        <transition name="fade-text">
          <span v-show="!collapsed" class="nav-label">写邮件</span>
        </transition>
      </router-link>

      <!-- 目录与规则 -->
      <router-link
        to="/folders"
        class="nav-item"
        active-class="active"
        @click="handleNavClick('/folders')"
      >
        <span class="nav-icon">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M2.5 5.5A2 2 0 0 1 4.5 3.5h3.2l1.6 2h6.2a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2h-11a2 2 0 0 1-2-2v-9Z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
            <path d="M7 10.5h6M10 7.5v6" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
          </svg>
        </span>
        <transition name="fade-text">
          <span v-show="!collapsed" class="nav-label">目录与规则</span>
        </transition>
      </router-link>

      <!-- 邮箱管理 -->
      <router-link
        to="/accounts"
        class="nav-item"
        active-class="active"
        @click="handleNavClick('/accounts')"
      >
        <span class="nav-icon">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <rect x="2.5" y="3.5" width="15" height="13" rx="2" stroke="currentColor" stroke-width="1.6"/>
            <circle cx="7" cy="9" r="2" stroke="currentColor" stroke-width="1.4"/>
            <path d="M11.5 8H16M11.5 11H14" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
          </svg>
        </span>
        <transition name="fade-text">
          <span v-show="!collapsed" class="nav-label">邮箱管理</span>
        </transition>
      </router-link>

      <!-- 通讯录（按邮箱账号） -->
      <div class="addr-section" v-show="!collapsed">
        <button class="nav-item addr-toggle" @click="showAddrBook = !showAddrBook">
          <span class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <rect x="3" y="2.5" width="11" height="15" rx="2" stroke="currentColor" stroke-width="1.6"/>
              <path d="M14 6h2a1 1 0 011 1v8a1 1 0 01-1 1h-2" stroke="currentColor" stroke-width="1.6"/>
              <circle cx="8.5" cy="8" r="1.8" stroke="currentColor" stroke-width="1.4"/>
              <path d="M5.8 13.2c.5-1.3 2.9-1.3 3.4 0" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
            </svg>
          </span>
          <span class="nav-label">通讯录</span>
          <span class="toggle-icon">{{ showAddrBook ? '▾' : '▸' }}</span>
        </button>
        <div v-if="showAddrBook" class="addr-book">
          <div v-for="acc in accountStore.accounts" :key="acc.id" class="addr-group">
            <div class="addr-group-title" @click="toggleGroup(acc.id)">
              <span class="addr-group-name">{{ acc.name || acc.email }}</span>
              <span class="addr-group-actions">
                <button class="addr-action-btn" title="新增联系人" @click.stop="openEdit(acc, null)">＋</button>
                <span class="toggle-icon">{{ expandedGroup === acc.id ? '▾' : '▸' }}</span>
              </span>
            </div>
            <div v-if="expandedGroup === acc.id" class="addr-list">
              <div v-if="addrLoading" class="addr-empty">加载中…</div>
              <div v-else-if="!accountContacts.length" class="addr-empty">暂无联系人</div>
              <div v-for="ct in accountContacts" :key="ct.id" class="addr-row">
                <button
                  class="addr-item"
                  :title="'点击复制 ' + ct.email"
                  @click="copyEmail(ct.email)"
                >{{ ct.name ? ct.name + ' · ' : '' }}{{ ct.email }}</button>
                <span class="addr-row-actions">
                  <button class="addr-action-btn" title="编辑" @click.stop="openEdit(acc, ct)">✎</button>
                  <button class="addr-action-btn addr-del" title="删除" @click.stop="removeContact(ct)">×</button>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 分割线 -->
      <div class="nav-divider"></div>

      <!-- 设置 -->
      <router-link
        to="/settings"
        class="nav-item"
        active-class="active"
        @click="handleNavClick('/settings')"
      >
        <span class="nav-icon">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <circle cx="10" cy="10" r="6" stroke="currentColor" stroke-width="1.6"/>
            <path d="M10 2V3.5M10 16.5V18M18 10H16.5M3.5 10H2M15.66 15.66l-1.06-1.06M5.4 5.4L4.34 4.34M15.66 4.34l-1.06 1.06M5.4 14.6l-1.06 1.06" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
          </svg>
        </span>
        <transition name="fade-text">
          <span v-show="!collapsed" class="nav-label">设置</span>
        </transition>
      </router-link>
    </nav>

    <!-- 底部信息 -->
    <div class="sidebar-footer">
      <div v-show="!collapsed" class="footer-status">
        <span class="status-dot online"></span>
        <span class="status-text">服务在线</span>
      </div>
      <button class="btn-icon btn-ghost toggle-btn" @click="$emit('toggle')" :title="collapsed ? '展开侧边栏' : '收起侧边栏'">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
          <path v-if="collapsed" d="M6 3L11 8L6 13" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
          <path v-else d="M10 3L5 8L10 13" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
    </div>

    <!-- 联系人编辑弹窗 -->
    <div v-if="editDialog.open" class="addr-dialog-mask" @click.self="editDialog.open = false">
      <div class="addr-dialog">
        <div class="addr-dialog-title">{{ editDialog.id ? '编辑联系人' : '新增联系人' }}</div>
        <input v-model="editDialog.email" type="email" class="form-input" placeholder="邮箱地址" />
        <input v-model="editDialog.name" type="text" class="form-input" placeholder="名称（可选）" />
        <div v-if="editDialog.error" class="addr-dialog-error">{{ editDialog.error }}</div>
        <div class="addr-dialog-btns">
          <button class="btn btn-ghost" @click="editDialog.open = false">取消</button>
          <button class="btn btn-primary" :disabled="editDialog.saving" @click="saveContact">保存</button>
        </div>
      </div>
    </div>

    <!-- 移动端遮罩层 -->
    <div
      v-if="isMobile && (open || mobileOpen)"
      class="sidebar-overlay"
      @click="mobileOpen = false"
    ></div>
  </aside>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAccountStore } from '@/stores/accountStore'
import { useMailStore } from '@/stores/mailStore'
import { getContacts, createContact, updateContact, deleteContact } from '@/api/contact'

const props = defineProps({
  collapsed: Boolean,
})

const emit = defineEmits(['toggle', 'navigate'])

const route = useRoute()
const accountStore = useAccountStore()
const mailStore = useMailStore()

const mobileOpen = ref(false)
const isMobile = ref(window.innerWidth <= 1024)

// 统一使用 mailStore.stats 作为未读数据源（与邮件列表页同步）
const unreadCount = computed(() => mailStore.stats?.unread || 0)
const open = computed(() => {
  return isMobile.value && mobileOpen.value
})

// 响应式检测
function handleResize() {
  isMobile.value = window.innerWidth <= 1024
  if (!isMobile.value) {
    mobileOpen.value = false
  }
}

function handleNavClick(path) {
  emit('navigate', path)
  if (isMobile.value) {
    mobileOpen.value = false
  }
}

// ---- 通讯录（按账号展开，点击复制） ----
const showAddrBook = ref(false)
const expandedGroup = ref(null)
const accountContacts = ref([])
const addrLoading = ref(false)

function toggleGroup(accountId) {
  if (expandedGroup.value === accountId) {
    expandedGroup.value = null
    return
  }
  expandedGroup.value = accountId
  loadAccountContacts(accountId)
}

async function loadAccountContacts(accountId) {
  addrLoading.value = true
  try {
    const res = await getContacts({ account_id: accountId, limit: 100 })
    accountContacts.value = res.contacts || []
  } catch (e) {
    accountContacts.value = []
  } finally {
    addrLoading.value = false
  }
}

async function copyEmail(email) {
  try {
    await navigator.clipboard.writeText(email)
    copiedTip.value = email
    setTimeout(() => { copiedTip.value = '' }, 1500)
  } catch (e) {
    // 剪贴板不可用时静默失败
  }
}
const copiedTip = ref('')

// ---- 联系人手动维护 ----
const editDialog = ref({ open: false, id: null, accountId: null, email: '', name: '', saving: false, error: '' })

function openEdit(account, contact) {
  editDialog.value = {
    open: true,
    id: contact ? contact.id : null,
    accountId: account ? account.id : null,
    email: contact ? contact.email : '',
    name: contact ? (contact.name || '') : '',
    saving: false,
    error: ''
  }
  // 新增时若该组尚未展开，先展开以便保存后可见
  if (account && expandedGroup.value !== account.id) toggleGroup(account.id)
}

async function saveContact() {
  const d = editDialog.value
  if (!d.email.trim() || !d.email.includes('@')) {
    d.error = '请输入有效的邮箱地址'
    return
  }
  d.saving = true
  d.error = ''
  try {
    if (d.id) {
      await updateContact(d.id, { email: d.email, name: d.name })
    } else {
      await createContact({ email: d.email, name: d.name, account_id: d.accountId })
    }
    d.open = false
    // 刷新当前展开的账号联系人
    if (expandedGroup.value != null) loadAccountContacts(expandedGroup.value)
  } catch (e) {
    d.error = e.message || '保存失败'
  } finally {
    d.saving = false
  }
}

async function removeContact(ct) {
  if (!confirm('确定删除联系人 ' + ct.email + ' 吗？')) return
  try {
    await deleteContact(ct.id)
    if (expandedGroup.value != null) loadAccountContacts(expandedGroup.value)
  } catch (e) {
    // 删除失败静默
  }
}

onMounted(() => {
  // 仅当无缓存数据时加载账号
  if (!accountStore.accounts.length) {
    accountStore.fetchAccounts()
  }
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.app-sidebar {
  position: fixed;
  top: var(--space-md);
  left: var(--space-md);
  bottom: var(--space-md);
  width: var(--sidebar-width);
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-light);
  box-shadow: var(--float-shadow);
  transition: width var(--transition-slow), transform var(--transition-slow), box-shadow var(--transition-slow);
  z-index: var(--z-sidebar);
  overflow: hidden;
}

.app-sidebar.collapsed { width: var(--sidebar-collapsed-width); }

/* ---- Logo 区域 ---- */
.sidebar-logo {
  display: flex;
  align-items: center;
  padding: 16px 16px 12px;
  gap: 8px;
  min-height: var(--header-height);
}

.logo-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-primary);
  font-family: inherit;
  flex: 1;
  border-radius: var(--radius-md);
  padding: 6px;
}
.logo-btn:hover { background: var(--bg-hover); }

.sidebar-brand-icon {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  object-fit: contain;
  filter: drop-shadow(0 5px 7px rgba(83, 40, 99, 0.18));
}

.logo-text {
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-bold);
  background: linear-gradient(135deg, var(--primary-500), var(--primary-400));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  letter-spacing: -0.3px;
}

.close-btn { flex-shrink: 0; }

/* ---- 底部状态 / 折叠按钮 ---- */
.sidebar-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-top: 1px solid var(--border-light);
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
}

.toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  margin-left: auto;
  border-radius: var(--radius-md);
}

.collapsed .sidebar-footer {
  padding: 12px 8px;
  justify-content: center;
}

.collapsed .toggle-btn {
  margin-left: 0;
}

/* ---- 导航 ---- */
.sidebar-nav {
  flex: 1;
  padding: 8px 10px;
  overflow-y: auto;
  overflow-x: hidden;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  text-decoration: none;
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
  white-space: nowrap;
  margin-bottom: 2px;
}
.nav-item:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.nav-item.active {
  background: var(--mail-unread-bg);
  color: var(--primary-500);
  font-weight: var(--font-weight-medium);
}
.nav-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 60%;
  background: linear-gradient(180deg, var(--primary-500), var(--primary-400));
  border-radius: 0 3px 3px 0;
}

.collapsed .nav-item { justify-content: center; padding: 10px; }

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 22px;
  height: 22px;
}

.nav-label {
  flex: 1;
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-normal);
}

.nav-badge {
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  font-size: 11px;
  font-weight: var(--font-weight-semibold);
  background: var(--error);
  color: #fff;
  border-radius: var(--radius-full);
  line-height: 20px;
  text-align: center;
}

.nav-divider {
  height: 1px;
  background: var(--border-light);
  margin: 8px 8px 12px;
}

/* ---- 通讯录 ---- */
.addr-toggle .toggle-icon { font-size: 11px; color: var(--text-tertiary); }
.addr-book { padding: 0 4px 4px 22px; }
.addr-group-title {
  display: flex; align-items: center; justify-content: space-between;
  font-size: 12px; color: var(--text-tertiary);
  padding: 6px 8px; border-radius: var(--radius-sm, 6px); cursor: pointer;
}
.addr-group-title:hover { background: var(--bg-hover); color: var(--text-primary); }
.addr-group-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.addr-group-actions { display: inline-flex; align-items: center; gap: 2px; }
.addr-row { display: flex; align-items: center; }
.addr-row .addr-item { flex: 1; min-width: 0; }
.addr-row-actions { display: none; align-items: center; gap: 2px; flex-shrink: 0; }
.addr-row:hover .addr-row-actions { display: inline-flex; }
.addr-action-btn {
  background: none; border: none; cursor: pointer; color: var(--text-tertiary);
  font-size: 13px; line-height: 1; padding: 3px 5px; border-radius: 4px;
}
.addr-action-btn:hover { background: var(--bg-hover); color: var(--primary-500); }
.addr-action-btn.addr-del:hover { color: var(--error, #ef4444); }

/* 联系人编辑弹窗 */
.addr-dialog-mask {
  position: fixed; inset: 0; background: var(--bg-overlay, rgba(0,0,0,.4));
  display: flex; align-items: center; justify-content: center; z-index: calc(var(--z-sidebar) + 10);
}
.addr-dialog {
  background: var(--bg-primary); border: 1px solid var(--border-light);
  border-radius: var(--radius-lg); padding: 20px; width: 320px;
  display: flex; flex-direction: column; gap: 10px;
  box-shadow: var(--float-shadow, 0 10px 30px rgba(0,0,0,.15));
}
.addr-dialog-title { font-weight: var(--font-weight-bold); color: var(--text-primary); }
.addr-dialog-error { font-size: 12px; color: var(--error, #ef4444); }
.addr-dialog-btns { display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px; }
.addr-list { display: flex; flex-direction: column; padding-left: 6px; }
.addr-item {
  text-align: left; background: none; border: none; cursor: copy;
  font-size: 12px; color: var(--text-secondary); font-family: inherit;
  padding: 4px 8px; border-radius: var(--radius-sm, 6px);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.addr-item:hover { background: var(--bg-hover); color: var(--primary-500); }
.addr-empty { font-size: 12px; color: var(--text-tertiary); padding: 4px 8px; }

/* ---- 底部状态 ---- */
.footer-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--font-size-sm);
  color: var(--text-tertiary);
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  animation: pulse-dot 2s ease-in-out infinite;
}
.status-dot.online { background: var(--success); }
@keyframes pulse-dot {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.35); }
  50% { opacity: 0.8; box-shadow: 0 0 0 4px rgba(16, 185, 129, 0); }
}

/* ---- 文本展开/折叠过渡 ---- */
.fade-text-enter-active, .fade-text-leave-active {
  transition: opacity 0.15s ease;
}
.fade-text-enter-from, .fade-text-leave-to { opacity: 0; }

/* ---- 移动端遮罩 ---- */
.sidebar-overlay {
  display: none;
}

@media (max-width: 1024px) {
  .app-sidebar {
    top: 0;
    left: 0;
    bottom: 0;
    border-radius: 0;
    box-shadow: none;
  }

  .sidebar-overlay {
    display: block;
    position: fixed;
    inset: 0;
    background: var(--bg-overlay);
    z-index: calc(var(--z-sidebar) - 1);
  }

  .app-sidebar.open + .sidebar-overlay,
  .app-sidebar.open ~ .app-overlay {
    display: block;
  }
}
</style>
