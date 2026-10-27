<!--
  SPDX-License-Identifier: AGPL-3.0-or-later
  Copyright (C) 2026  magiccode (魔法代码)

  手机端底部导航栏：邮件 / 写信 / 草稿 / 设置（可收缩）
  仅 ≤768px 且已登录时由 App.vue 渲染。
  默认收起：只在左下角显示一个圆形把手按钮，点击展开完整导航条；
  点击导航项跳转后自动收起，避免遮挡写信等页面的底部操作按钮。
-->
<template>
  <!-- 收起态：左下角把手按钮 -->
  <button
    v-if="collapsed"
    class="tabbar-handle"
    :class="{ 'has-unread': unreadCount > 0 }"
    aria-label="展开导航菜单"
    @click="toggle(false)"
  >
    <span class="handle-icon" v-html="icons.mail"></span>
    <span v-if="unreadCount > 0" class="handle-badge">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
  </button>

  <!-- 展开态：完整导航条 -->
  <nav v-else class="mobile-tabbar">
    <button
      v-for="tab in tabs"
      :key="tab.path"
      class="tabbar-item"
      :class="{ active: isActive(tab) }"
      @click="go(tab)"
    >
      <span class="tabbar-icon" v-html="tab.icon"></span>
      <span class="tabbar-label">{{ tab.label }}</span>
      <span v-if="tab.badge && tab.badge > 0" class="tabbar-badge">{{ tab.badge > 99 ? '99+' : tab.badge }}</span>
    </button>
    <!-- 收起按钮（左下角圆形，回到把手状态） -->
    <button class="tabbar-collapse" aria-label="收起导航菜单" @click="toggle(true)">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>
    </button>
  </nav>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const props = defineProps({
  unreadCount: { type: Number, default: 0 }
})

const route = useRoute()
const router = useRouter()

const COLLAPSE_KEY = 'novamail-tabbar-collapsed'
const collapsed = ref(localStorage.getItem(COLLAPSE_KEY) !== 'false') // 默认收起

function toggle(v) {
  collapsed.value = v
  localStorage.setItem(COLLAPSE_KEY, String(v))
}

// 跳转后自动收起，避免遮挡页面底部按钮（写信发送等）
watch(() => route.path, () => {
  if (!collapsed.value) toggle(true)
})

const icons = {
  mail: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-10 6L2 7"/></svg>',
  compose: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>',
  draft: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.8 2.8 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>',
  settings: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z"/></svg>',
  folder: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"/><path d="M12 10v6M9 13h6"/></svg>'
}

const tabs = [
  { path: '/mails', label: '邮件', icon: icons.mail, badgeSource: 'unread' },
  { path: '/compose', label: '写信', icon: icons.compose },
  { path: '/drafts', label: '草稿', icon: icons.draft },
  { path: '/folders', label: '目录', icon: icons.folder },
  { path: '/settings', label: '设置', icon: icons.settings }
]

function isActive(tab) {
  if (tab.path === '/mails') {
    return route.path === '/mails' || route.path.startsWith('/mail/')
  }
  return route.path.startsWith(tab.path)
}

function go(tab) {
  if (isActive(tab)) {
    toggle(true)
    return
  }
  router.push(tab.path)
}
</script>

<style scoped>
.mobile-tabbar {
  display: none;
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  height: calc(56px + env(safe-area-inset-bottom, 0px));
  padding-bottom: env(safe-area-inset-bottom, 0px);
  background: var(--bg-secondary, #fff);
  border-top: 1px solid var(--border-color, #e5e7eb);
  z-index: var(--z-tabbar, 900);
}

/* 收起态把手按钮：左下角圆形悬浮 */
.tabbar-handle {
  display: none;
  position: fixed;
  left: 10px;
  bottom: calc(10px + env(safe-area-inset-bottom, 0px));
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 1px solid var(--border-color, #e5e7eb);
  background: var(--bg-secondary, #fff);
  color: var(--text-secondary, #6b7280);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  align-items: center;
  justify-content: center;
  z-index: var(--z-tabbar, 900);
  -webkit-tap-highlight-color: transparent;
  padding: 0;
}

.tabbar-handle .handle-icon {
  display: flex;
}

.tabbar-handle.has-unread {
  color: var(--primary-color, #3b82f6);
}

.handle-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: #ef4444;
  color: #fff;
  font-size: 10px;
  line-height: 16px;
  font-weight: 600;
}

/* 展开态右下角收起按钮 */
.tabbar-collapse {
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: none;
  background: var(--bg-tertiary, #f3f4f6);
  color: var(--text-secondary, #6b7280);
  display: flex;
  align-items: center;
  justify-content: center;
  -webkit-tap-highlight-color: transparent;
  padding: 0;
}

.tabbar-item {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border: none;
  background: none;
  color: var(--text-secondary, #6b7280);
  font-size: 11px;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}

.tabbar-item.active {
  color: var(--primary-color, #3b82f6);
}

.tabbar-label {
  line-height: 1;
}

.tabbar-badge {
  position: absolute;
  top: 4px;
  right: calc(50% - 20px);
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: #ef4444;
  color: #fff;
  font-size: 10px;
  line-height: 16px;
  font-weight: 600;
}

@media (max-width: 768px) {
  .mobile-tabbar,
  .tabbar-handle {
    display: flex;
  }
}
</style>
