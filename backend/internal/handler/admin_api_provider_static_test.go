package handler

import (
	"strings"
	"testing"
)

func TestAdminAPIProviderConfigOnlyAllowsGeneralOpenAIProviders(t *testing.T) {
	admin := readProjectFile(t, "internal", "handler", "admin.go")

	if !strings.Contains(admin, `req.Category != "general" && req.Category != "general_text"`) {
		t.Fatal("admin API provider save should only allow general image and general text categories")
	}
	if !strings.Contains(admin, `normalizeAPIProviderCategory(req.Category)`) {
		t.Fatal("admin API provider save should normalize category aliases before validation")
	}
	if !strings.Contains(admin, `case "通用文本":`) {
		t.Fatal("admin API provider save should accept the general text label as an alias")
	}
	if strings.Contains(admin, `req.Category != "general" && req.Category != "professional_drawing"`) {
		t.Fatal("admin API provider save should not allow professional_drawing category")
	}
	if strings.Contains(admin, `req.Category = "assistant_chat"`) {
		t.Fatal("admin API provider save should not map assistant category to assistant_chat")
	}
	if !strings.Contains(admin, `req.Provider = "openai"`) {
		t.Fatal("admin API provider save should force OpenAI provider type")
	}
	if strings.Contains(admin, `req.Provider = "custom"`) {
		t.Fatal("admin API provider save should not default provider type to custom")
	}
}
