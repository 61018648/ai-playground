package handler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readProjectFile(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{repoRoot(t)}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestProfessionalDrawUsesConfigurableGeneralProviders(t *testing.T) {
	handler := readProjectFile(t, "internal", "handler", "handler.go")
	admin := readProjectFile(t, "internal", "handler", "admin.go")
	router := readProjectFile(t, "internal", "server", "router.go")

	if !strings.Contains(admin, `key != "professional_draw"`) {
		t.Fatal("admin setting whitelist should allow professional_draw")
	}
	if !strings.Contains(router, `POST /api/v1/generations/professional-draw/rewrite`) {
		t.Fatal("rewrite route should be registered")
	}
	if !strings.Contains(handler, "CreateProfessionalDrawRewrite") {
		t.Fatal("rewrite handler should exist")
	}
	if !strings.Contains(handler, `professionalDrawProvider(r.Context(), "drawProviderId"`) {
		t.Fatal("draw generation should resolve the configured draw provider")
	}
	if !strings.Contains(handler, `professionalDrawProvider(r.Context(), "rewriteProviderId"`) {
		t.Fatal("prompt rewrite should resolve the configured rewrite provider")
	}
	if !strings.Contains(handler, `sourceImageURL = latestDrawAssetURL(detail.Messages)`) {
		t.Fatal("follow-up draw should use the previous generated image from conversation messages")
	}
	if !strings.Contains(handler, `requestOpenAIImageWithSource(ctx, provider, prompt, params, sourceImageURL)`) {
		t.Fatal("professional draw job should route follow-up requests through the source-aware image requester")
	}
	if !strings.Contains(handler, `expectedCategory := "general"`) {
		t.Fatal("draw provider should come from general image provider configs")
	}
	if !strings.Contains(handler, `expectedCategory = "general_text"`) {
		t.Fatal("rewrite provider should come from general text provider configs")
	}
}
