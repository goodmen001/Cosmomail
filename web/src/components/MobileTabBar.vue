<!--
  SPDX-License-Identifier: AGPL-3.0-or-later
  Copyright (C) 2026  magiccode (魔法代码)

  手机端底部导航栏：邮件 / 写信 / 草稿 / 设置
  仅 ≤768px 且已登录时由 App.vue 渲染
-->
<template>
  <nav class="mobile-tabbar">
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
  </nav>
</template>

<script setup>
import { useRoute, useRouter } from 'vue-router'

defineProps({
  unreadCount: { type: Number, default: 0 }
})

const route = useRoute()
const router = useRouter()

const icons = {
  mail: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-10 6L2 7"/></svg>',
  compose: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>',
  draft: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.8 2.8 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>',
  settings: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z"/></svg>'
}

const tabs = [
  { path: '/mails', label: '邮件', icon: icons.mail, badgeSource: 'unread' },
  { path: '/compose', label: '写信', icon: icons.compose },
  { path: '/drafts', label: '草稿', icon: icons.draft },
  { path: '/settings', label: '设置', icon: icons.settings }
]

function isActive(tab) {
  if (tab.path === '/mails') {
    return route.path === '/mails' || route.path.startsWith('/mail/')
  }
  return route.path.startsWith(tab.path)
}

function go(tab) {
  if (isActive(tab)) return
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
  .mobile-tabbar {
    display: flex;
  }
}
</style>
