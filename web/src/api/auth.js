// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

import request from './request'

/** 查询认证状态（是否需要注册） */
export function getAuthStatus() {
  return request.get('/auth/status')
}

/** 当前登录用户实时信息（含数据库中的管理员身份） */
export function getMe() {
  return request.get('/auth/me')
}

/** 登录 */
export function login(data) {
  return request.post('/auth/login', data)
}

/** 注册（仅首次可用） */
export function register(data) {
  return request.post('/auth/register', data, { timeout: 10000 })
}

/** 查询注册开关 */
export function getRegistrationEnabled() {
  return request.get('/auth/registration-enabled')
}

/** 修改自己的密码 */
export function changePassword(data) {
  return request.put('/auth/change-password', data)
}

/** 用户列表（管理员） */
export function listUsers() {
  return request.get('/admin/users')
}

/** 删除用户（管理员） */
export function deleteUser(id) {
  return request.delete(`/admin/users/${id}`)
}

/** 设置注册开关（管理员） */
export function setRegistrationEnabled(enabled) {
  return request.put('/admin/registration-enabled', { enabled })
}

/** 创建用户（管理员） */
export function createUser(data) {
  return request.post('/admin/users', data)
}

/** 禁用/启用用户（管理员） */
export function setUserDisabled(id, disabled) {
  return request.put(`/admin/users/${id}/disabled`, disabled)
}

/** 获取（或首次生成）当前用户 API Token */
export function getApiToken() {
  return request.get('/auth/api-token')
}

/** 吊销当前用户 API Token */
export function revokeApiToken() {
  return request.delete('/auth/api-token')
}

/** 获取 Gotify 通知配置 */
export function getGotifyConfig() {
  return request.get('/auth/gotify')
}

/** 保存 Gotify 通知配置 */
export function saveGotifyConfig(data) {
  return request.put('/auth/gotify', data)
}

/** 发送 Gotify 测试通知 */
export function testGotify(data) {
  return request.post('/auth/gotify/test', data)
}
