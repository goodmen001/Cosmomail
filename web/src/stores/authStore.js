// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('cosmomail-token') || '')
  const username = ref('')
  const setupRequired = ref(false)
  const registrationEnabled = ref(false)
  const isAdmin = ref(false)
  const initialized = ref(false)

  const isLoggedIn = computed(() => !!token.value)

  async function init() {
    if (token.value) {
      // 有 token 时优先用服务端实时身份（避免浏览器旧 JWT 里 is_admin 过期）
      try {
        const me = await authApi.getMe()
        username.value = me.username || ''
        isAdmin.value = !!me.is_admin
      } catch (_) {
        // token 无效（401 / 后端已重置等），清除并标记需要重新登录
        logout()
      }
    } else {
      try {
        const res = await authApi.getAuthStatus()
        setupRequired.value = res.setup_required
        registrationEnabled.value = !!res.registration_enabled
      } catch (_) {
        setupRequired.value = true
      }
    }
    initialized.value = true
  }

  async function doLogin(loginData) {
    const res = await authApi.login(loginData)
    setToken(res.token, res.username)
    isAdmin.value = !!res.is_admin
    return res
  }

  async function doRegister(regData) {
    const res = await authApi.register(regData)
    if (res.token) {
      setToken(res.token, res.username)
      isAdmin.value = !!res.is_admin
    }
    return res
  }

  function setToken(newToken, name) {
    token.value = newToken
    username.value = name || ''
    localStorage.setItem('cosmomail-token', newToken || '')
  }

  // 从 JWT payload 解析完整信息（纯前端解析）
  function parsePayload(jwtStr) {
    try {
      return JSON.parse(atob(jwtStr.split('.')[1]))
    } catch { return {} }
  }

  function logout() {
    token.value = ''
    username.value = ''
    isAdmin.value = false
    localStorage.removeItem('cosmomail-token')
  }

  // 从 JWT payload 解析用户名（纯前端解析，无需后端请求）
  function parseUsername(jwtStr) {
    try {
      const payload = JSON.parse(atob(jwtStr.split('.')[1]))
      return payload.username || ''
    } catch { return '' }
  }

  return {
    token, username, isLoggedIn, setupRequired, registrationEnabled, isAdmin, initialized,
    init, doLogin, doRegister, logout,
  }
})
