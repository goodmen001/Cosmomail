// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)
//
// 运行时服务器地址：手机 App（Capacitor）用户在登录页填写，
// Web 端保持同源相对路径。统一从这里读取，避免散落各处。

const SERVER_KEY = 'novamail-server'

/** 规范化服务器地址：去尾部斜杠；无协议时默认 http:// */
function normalizeServer(input) {
  let s = String(input || '').trim().replace(/\/+$/, '')
  if (s && !/^https?:\/\//i.test(s)) s = 'http://' + s
  return s
}

/** 保存服务器地址（空值清除，回退同源模式） */
export function setServerBase(input) {
  const s = normalizeServer(input)
  if (s) localStorage.setItem(SERVER_KEY, s)
  else localStorage.removeItem(SERVER_KEY)
}

/** 获取服务器地址；Web 端（未配置）返回 ''，即同源相对路径 */
export function getServerBase() {
  const isNative = typeof window !== 'undefined' && window.Capacitor && window.Capacitor.isNativePlatform && window.Capacitor.isNativePlatform()
  let s = localStorage.getItem(SERVER_KEY) || ''
  if (isNative && !s) {
    // App 端从未配置过服务器，走登录页
    return ''
  }
  return normalizeServer(s)
}

/** 是否为原生 App 环境 */
export function isNativeApp() {
  return typeof window !== 'undefined' && !!(window.Capacitor && window.Capacitor.isNativePlatform && window.Capacitor.isNativePlatform())
}

/** API 根地址（axios baseURL 用） */
export function apiBase() {
  return getServerBase() + '/api/v1'
}
