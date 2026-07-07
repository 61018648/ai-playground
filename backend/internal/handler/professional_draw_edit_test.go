package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"image-ai/backend/internal/model"
)

func TestLatestDrawAssetURLReturnsMostRecentAssistantAsset(t *testing.T) {
	messages := []model.ConversationMessage{
		{Role: "assistant", Meta: json.RawMessage(`{"status":"succeeded","assetUrl":"data:image/png;base64,old"}`)},
		{Role: "user", Content: "改成亮色风格"},
		{Role: "assistant", Meta: json.RawMessage(`{"status":"running","assetUrl":"data:image/png;base64,ignore"}`)},
		{Role: "assistant", Meta: json.RawMessage(`{"status":"succeeded","assetUrl":"data:image/png;base64,new"}`)},
	}

	got := latestDrawAssetURL(messages)

	if got != "data:image/png;base64,new" {
		t.Fatalf("expected newest succeeded assistant asset, got %q", got)
	}
}

func TestRequestOpenAIImageWithSourceUsesEditsEndpoint(t *testing.T) {
	imageBytes := []byte("previous png bytes")
	source := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes)
	var requestPath string
	var modelValue string
	var promptValue string
	var sourceValue string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected JSON request, got %q", r.Header.Get("Content-Type"))
		}
		var body struct {
			Model  string `json:"model"`
			Prompt string `json:"prompt"`
			Images []struct {
				ImageURL string `json:"image_url"`
			} `json:"images"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		modelValue = body.Model
		promptValue = body.Prompt
		if len(body.Images) > 0 {
			sourceValue = body.Images[0].ImageURL
		}
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"` + base64.StdEncoding.EncodeToString([]byte("edited image")) + `"}]}`))
	}))
	defer server.Close()

	got, err := requestOpenAIImageWithSource(context.Background(), model.APIProvider{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-key",
		Model:   "gpt-image-2",
	}, "帮我把图片变成亮色风格", nil, source)

	if err != nil {
		t.Fatalf("requestOpenAIImageWithSource returned error: %v", err)
	}
	if requestPath != "/v1/images/edits" {
		t.Fatalf("expected edits endpoint, got %q", requestPath)
	}
	if modelValue != "gpt-image-2" {
		t.Fatalf("expected model from provider, got %q", modelValue)
	}
	if promptValue != "帮我把图片变成亮色风格" {
		t.Fatalf("expected prompt in form, got %q", promptValue)
	}
	if sourceValue != source {
		t.Fatalf("expected source image data URL, got %q", sourceValue)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("expected data URL response, got %q", got)
	}
}
