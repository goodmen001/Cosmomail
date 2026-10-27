// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

import request from './request'

/** 联系人列表（发信收件人记录，供写信时选择） */
export function getContacts(params = {}) {
  return request.get('/contacts', { params })
}

/** 手动全量提取指定账号已同步邮件中的联系人 */
export function syncAccountContacts(accountId) {
  return request.post('/contacts/sync', null, { params: { account_id: accountId } })
}

/** 从指定账号的 CardDAV 服务器同步联系人 */
export function syncCardDAVContacts(accountId) {
  return request.post('/contacts/carddav/sync', null, { params: { account_id: accountId } })
}

/** 手动新增联系人 */
export function createContact(data) {
  return request.post('/contacts', data)
}

/** 修改联系人 */
export function updateContact(id, data) {
  return request.put(`/contacts/${id}`, data)
}

/** 删除联系人 */
export function deleteContact(id) {
  return request.delete(`/contacts/${id}`)
}
