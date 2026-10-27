// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package smtp

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildMessageWithAttachments(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("hello attachment"))
	msg := string(buildMessage(
		"sender@example.com",
		[]string{"to@example.com"},
		nil, nil,
		"附件测试",
		"正文内容",
		"",
		[]SendAttachment{
			{Filename: "报告.txt", ContentType: "text/plain", Data: data},
		},
	))

	if !strings.Contains(msg, "multipart/mixed") {
		t.Fatalf("期望 multipart/mixed，实际:\n%s", msg)
	}
	if !strings.Contains(msg, `filename*=UTF-8''%E6%8A%A5%E5%91%8A.txt`) {
		t.Fatalf("期望 RFC5987 编码文件名，实际:\n%s", msg)
	}
	if !strings.Contains(msg, base64.StdEncoding.EncodeToString([]byte("hello attachment"))) {
		t.Fatalf("期望附件内容存在，实际:\n%s", msg)
	}
	if !strings.Contains(msg, base64.StdEncoding.EncodeToString([]byte("正文内容"))) {
		t.Fatalf("期望正文存在，实际:\n%s", msg)
	}

	// multipart 内部嵌套 alternative 也应保留
	if strings.Contains(msg, "multipart/alternative") {
		t.Logf("含 HTML 时正文应为 alternative 嵌套")
	}
}

func TestBuildMessageWithHTMLAndAttachments(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("binary\x01\x02"))
	msg := string(buildMessage(
		"sender@example.com",
		[]string{"to@example.com"},
		nil, nil,
		"主题",
		"text",
		"<p>html</p>",
		[]SendAttachment{
			{Filename: "a.bin", ContentType: "", Data: data},
		},
	))

	if !strings.Contains(msg, "multipart/alternative") {
		t.Fatalf("带 HTML 时期望嵌套 multipart/alternative，实际:\n%s", msg)
	}
	if !strings.Contains(msg, "application/octet-stream") {
		t.Fatalf("空 ContentType 应回退 octet-stream，实际:\n%s", msg)
	}
}

func TestBuildMessageInvalidAttachmentsSkipped(t *testing.T) {
	msg := string(buildMessage(
		"sender@example.com",
		[]string{"to@example.com"},
		nil, nil,
		"主题",
		"body",
		"",
		[]SendAttachment{
			{Filename: "", Data: "AAAA"},        // 无文件名，跳过
			{Filename: "b.txt", Data: ""},       // 无数据，跳过
			{Filename: "ok.txt", Data: base64.StdEncoding.EncodeToString([]byte("x"))},
		},
	))

	if !strings.Contains(msg, "multipart/mixed") {
		t.Fatalf("存在有效附件时应有 mixed 包裹，实际:\n%s", msg)
	}
	if !strings.Contains(msg, "ok.txt") {
		t.Fatalf("有效附件应保留，实际:\n%s", msg)
	}
	if strings.Contains(msg, "b.txt") {
		t.Fatalf("无效附件应被跳过，实际:\n%s", msg)
	}
}

func TestRfc5987Encode(t *testing.T) {
	cases := map[string]string{
		"plain.txt": "plain.txt",
		"报告.pdf":    "%E6%8A%A5%E5%91%8A.pdf",
		"a b.txt":   "a%20b.txt",
	}
	for in, want := range cases {
		if got := rfc5987Encode(in); got != want {
			t.Errorf("rfc5987Encode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRewrapBase64(t *testing.T) {
	in := "ab== c!d" + strings.Repeat("x", 100)
	got := rewrapBase64(in)
	if strings.ContainsAny(got, "! ") {
		t.Fatalf("期望清洗非法字符，实际: %q", got)
	}
	for _, line := range strings.Split(got, "\r\n") {
		if len(line) > 76 {
			t.Fatalf("行宽超 76: %d", len(line))
		}
	}
}
