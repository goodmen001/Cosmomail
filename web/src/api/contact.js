// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

import request from './request'

/** 联系人列表（发信收件人记录，供写信时选择） */
export function getContacts(params = {}) {
  return request.get('/contacts', { params })
}
