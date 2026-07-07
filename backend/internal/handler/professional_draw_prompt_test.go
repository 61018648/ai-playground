package handler

import (
	"strings"
	"testing"

	"image-ai/backend/internal/model"
)

func TestBuildDrawContextPromptUsesReadableChinese(t *testing.T) {
	prompt := buildDrawContextPrompt([]model.ConversationMessage{
		{Role: "user", Content: "生成一张科技海报"},
	}, "改成亮色风格")

	if !strings.Contains(prompt, "这是同一个生图对话的连续上下文") {
		t.Fatalf("context prompt should contain readable Chinese guidance, got %q", prompt)
	}
	if !strings.Contains(prompt, "历史用户需求") || !strings.Contains(prompt, "最新用户需求") {
		t.Fatalf("context prompt should label history and latest request, got %q", prompt)
	}
	if strings.Contains(prompt, "鐠") || strings.Contains(prompt, "閸") || strings.Contains(prompt, "€") {
		t.Fatalf("context prompt should not contain mojibake, got %q", prompt)
	}
}

func TestProfessionalDrawRewritePromptUsesReadableChinese(t *testing.T) {
	prompt := professionalDrawRewritePrompt("帮我画一张海报")

	if !strings.Contains(prompt, "专业绘图提示词优化师") {
		t.Fatalf("rewrite prompt should contain readable Chinese guidance, got %q", prompt)
	}
	if strings.Contains(prompt, "鐠") || strings.Contains(prompt, "閸") || strings.Contains(prompt, "€") {
		t.Fatalf("rewrite prompt should not contain mojibake, got %q", prompt)
	}
}
