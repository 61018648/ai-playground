package handler

import (
	"strings"
	"testing"
)

func TestAppCenterUsesTypedProviderConfigs(t *testing.T) {
	modelFile := readProjectFile(t, "internal", "model", "model.go")
	admin := readProjectFile(t, "internal", "handler", "admin.go")
	handler := readProjectFile(t, "internal", "handler", "handler.go")
	repository := readProjectFile(t, "internal", "repository", "repository.go")
	migration := readProjectFile(t, "migrations", "015_app_type.sql")

	if !strings.Contains(modelFile, "AppType") {
		t.Fatal("app model should expose appType")
	}
	if !strings.Contains(migration, "ADD COLUMN IF NOT EXISTS app_type") {
		t.Fatal("migration should add apps.app_type")
	}
	if !strings.Contains(admin, "AppType") || !strings.Contains(admin, "expectedAppProviderCategory") {
		t.Fatal("admin app save should accept appType and map it to a provider category")
	}
	if !strings.Contains(admin, "DecodeJSONLoose") {
		t.Fatal("admin app save should tolerate harmless extra fields from the editor")
	}
	if !strings.Contains(admin, "stringFromJSONSelectValue") {
		t.Fatal("admin app save should normalize select values posted as objects")
	}
	if !strings.Contains(admin, `provider.Category != expectedCategory`) {
		t.Fatal("admin app save should reject providers from the wrong category")
	}
	if !strings.Contains(repository, "app_type") {
		t.Fatal("repository app queries should persist and return app_type")
	}
	if !strings.Contains(handler, "resolveGenerationAppConfig") {
		t.Fatal("generation should resolve provider config from the selected app")
	}
	if !strings.Contains(handler, "renderAppPromptTemplate") {
		t.Fatal("generation should apply the app prompt template")
	}
	if !strings.Contains(handler, "provider.Model") {
		t.Fatal("generation should use the model from the provider config")
	}
}
