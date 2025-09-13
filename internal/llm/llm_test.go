package llm

import (
	"os"
	"strings"
	"testing"

	"github.com/edhuardotierrez/gommit/internal/env"
	"github.com/edhuardotierrez/gommit/internal/git"
	"github.com/edhuardotierrez/gommit/internal/types"
)

// TestGetAvailableModels_Sanity ensures each provider reports at least one model we can try.
func TestGetAvailableModels_Sanity(t *testing.T) {
	providers := []types.ProviderName{
		types.ProviderOpenAI,
		types.ProviderAnthropic,
		types.ProviderOllama,
		types.ProviderGoogle,
	}

	for _, p := range providers {
		models := GetAvailableModels(p)
		if len(models) == 0 {
			t.Fatalf("provider %s returned no models", p)
		}
	}
}

// TestGenerateCommitMessage_Minimal runs a minimal integration for each provider if env is present.
// It uses a tiny diff and a short custom prompt file (>=101 chars to be picked up by code logic).
func TestGenerateCommitMessage_Minimal(t *testing.T) {
	// Create minimal custom prompt used by GenerateCommitMessage when >100 chars.
	// Keep it compact but above the threshold to exercise the custom prompt path.
	custom := strings.Repeat("Keep commits concise. Avoid secrets. Use imperative mood. ", 3)
	if len(custom) <= 100 {
		t.Fatalf("custom prompt should exceed 100 chars, got %d", len(custom))
	}
	if err := os.WriteFile(".gommitrules", []byte(custom), 0o644); err != nil {
		t.Fatalf("failed to write .gommitrules: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(".gommitrules") })

	// Common minimal config and changes
	cfg := &types.Config{
		CommitStyle:   "simple",
		TruncateLines: 3,
		MaxLineWidth:  60,
	}
	changes := []git.StagedChange{{
		Path:   "file.txt",
		Status: "M",
		Diff:   "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n+hello world\n",
	}}

	// Table of providers with env requirements and optional URI key.
	type caseDef struct {
		name     string
		provider types.ProviderName
		apiEnv   string
		uriEnv   string
	}
	cases := []caseDef{
		{name: "openai", provider: types.ProviderOpenAI, apiEnv: "OPENAI_API_KEY"},
		{name: "anthropic", provider: types.ProviderAnthropic, apiEnv: "ANTHROPIC_API_KEY"},
		{name: "ollama", provider: types.ProviderOllama, apiEnv: "OLLAMA_API_KEY", uriEnv: "OLLAMA_URI"},
		{name: "google", provider: types.ProviderGoogle, apiEnv: "GOOGLE_API_KEY"},
	}

	// Load env file if present
	env.LoadFile()

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			// Skip if required env vars not present
			if tc.apiEnv != "" && env.GetString(tc.apiEnv) == "" {
				t.Skipf("skipping %s: missing %s", tc.name, tc.apiEnv)
			}
			if tc.uriEnv != "" && env.GetString(tc.uriEnv) == "" {
				t.Skipf("skipping %s: missing %s", tc.name, tc.uriEnv)
			}

			models := GetAvailableModels(tc.provider)
			if len(models) == 0 {
				t.Fatalf("no models for provider %s", tc.provider)
			}
			selectedModel := models[0]

			// Prepare provider-specific config
			sel := types.ProviderConfig{
				APIKey:      env.GetString(tc.apiEnv),
				URI:         env.GetString(tc.uriEnv),
				Model:       selectedModel,
				Temperature: 0.0,
			}

			msg, err := GenerateCommitMessage(cfg, changes, string(tc.provider), sel)
			if err != nil {
				t.Fatalf("GenerateCommitMessage failed for %s: %v", tc.name, err)
			}
			if strings.TrimSpace(msg) == "" {
				t.Fatalf("empty commit message for %s", tc.name)
			}
			t.Logf("[model=%s] generated message: %s", tc.name, msg)
			t.Log("-----------------------------")
		})
	}
}
