// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)
//
// 新邮件角标：App 端用 @capawesome/capacitor-badge 设置桌面图标数字角标，
// Web 端回退 navigator.setAppBadge（部分浏览器支持）。

import { getServerBase, apiBase, isNativeApp } from '@/api/serverBase'

let Badge = null
// 动态引入：Web 端构建不打包原生插件模块也可正常加载（插件对 web 无副作用）
if (isNativeApp()) {
  import('@capawesome/capacitor-badge')
    .then((m) => { Badge = m.Badge; syncBadgeFromStorage() })
    .catch(() => { Badge = null })
}

const COUNT_KEY = 'cosmomail-badge-count'

function readLastCount() {
  const n = parseInt(localStorage.getItem(COUNT_KEY) || '0', 10)
  return Number.isFinite(n) && n > 0 ? n : 0
}

function storeCount(n) {
  if (n > 0) localStorage.setItem(COUNT_KEY, String(n))
  else localStorage.removeItem(COUNT_KEY)
}

async function applyBadge(count) {
  storeCount(count)
  try {
    if (Badge) {
      if (count > 0) await Badge.set({ count })
      else await Badge.clear()
      return
    }
  } catch (e) {
    console.warn('[badge] 原生角标设置失败:', e && e.message)
  }
  try {
    if (navigator.setAppBadge) {
      if (count > 0) await navigator.setAppBadge(count)
      else await navigator.clearAppBadge()
    }
  } catch (e) {
    /* 浏览器不支持时静默 */
  }
}

/** 启动时恢复（App 重启后角标还原） */
function syncBadgeFromStorage() {
  const n = readLastCount()
  if (n > 0 && Badge) {
    Badge.set({ count: n }).catch(() => {})
  }
}

/** 未读数变化入口：0 表示清除角标 */
export function updateBadge(unreadCount) {
  const n = Math.max(0, parseInt(unreadCount || 0, 10) || 0)
  applyBadge(n)
}

/** 页面可见时以服务端统计为准刷新角标 */
export async function refreshBadgeFromStats() {
  try {
    const token = localStorage.getItem('cosmomail-token')
    if (!token) return
    const res = await fetch(`${apiBase()}/mails/stats`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    if (!res.ok) return
    const data = await res.json()
    updateBadge(data && data.unread)
  } catch (e) {
    /* 离线时忽略 */
  }
}
