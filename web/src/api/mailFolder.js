// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

import request from './request'

// ---- 自定义邮件目录 ----

// 获取指定账号的目录列表
export function getFolders(accountId) {
  return request.get('/mail-folders', { params: { account_id: accountId } })
}

// 创建目录
export function createFolder(accountId, data) {
  return request.post('/mail-folders', data, { params: { account_id: accountId } })
}

// 重命名/排序目录
export function updateFolder(id, data) {
  return request.put(`/mail-folders/${id}`, data)
}

// 删除目录（目录下邮件退回收件箱，引用该目录的规则一并删除）
export function deleteFolder(id) {
  return request.delete(`/mail-folders/${id}`)
}

// ---- 归档规则 ----

// 获取指定账号的规则列表
export function getRules(accountId) {
  return request.get('/mail-rules', { params: { account_id: accountId } })
}

// 创建规则
export function createRule(accountId, data) {
  return request.post('/mail-rules', data, { params: { account_id: accountId } })
}

// 修改规则
export function updateRule(id, data) {
  return request.put(`/mail-rules/${id}`, data)
}

// 删除规则
export function deleteRule(id) {
  return request.delete(`/mail-rules/${id}`)
}
