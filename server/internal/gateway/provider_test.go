package gateway

import (
	"testing"
)

func TestProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	if len(registry.ListProviders()) != 0 {
		t.Error("Expected empty registry")
	}

	// Register providers
	registry.Register(NewOpenAIProvider("", ""))
	registry.Register(NewClaudeProvider(""))
	registry.Register(NewDeepSeekProvider(""))
	registry.Register(NewOllamaProvider(""))

	providers := registry.ListProviders()
	if len(providers) != 4 {
		t.Errorf("Expected 4 providers, got %d", len(providers))
	}
}

func TestProviderMatching(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(NewClaudeProvider(""))
	registry.Register(NewOllamaProvider(""))
	registry.Register(NewDeepSeekProvider(""))
	registry.Register(NewOpenAIProvider("", ""))

	testCases := []struct {
		model    string
		expected string
	}{
		{"gpt-4", "openai"},
		{"gpt-3.5-turbo", "openai"},
		{"claude-3-opus", "anthropic"},
		{"claude-2", "anthropic"},
		{"llama-2-70b", "ollama"},
		{"mistral-7b", "ollama"},
		{"qwen-14b", "ollama"},
		{"deepseek-chat", "deepseek"},
		{"deepseek-coder", "deepseek"},
		{"o1-preview", "openai"},
	}

	for _, tc := range testCases {
		provider := registry.GetProvider(tc.model)
		if provider == nil {
			t.Errorf("No provider found for model %s", tc.model)
			continue
		}
		if provider.Name() != tc.expected {
			t.Errorf("Model %s: expected provider %s, got %s", tc.model, tc.expected, provider.Name())
		}
	}
}

func TestOpenAIProvider(t *testing.T) {
	provider := NewOpenAIProvider("https://api.example.com/v1", "test-key")

	if provider.Name() != "openai" {
		t.Errorf("Expected name 'openai', got '%s'", provider.Name())
	}

	if !provider.SupportsStreaming() {
		t.Error("OpenAI should support streaming")
	}

	// Test matching
	if !provider.Match("gpt-4") {
		t.Error("Should match gpt-4")
	}
	if !provider.Match("gpt-3.5-turbo") {
		t.Error("Should match gpt-3.5-turbo")
	}
	if !provider.Match("o1-mini") {
		t.Error("Should match o1-mini")
	}
	if provider.Match("claude-3") {
		t.Error("Should not match claude-3")
	}
}

func TestClaudeProvider(t *testing.T) {
	provider := NewClaudeProvider("")

	if provider.Name() != "anthropic" {
		t.Errorf("Expected name 'anthropic', got '%s'", provider.Name())
	}

	// Test matching
	if !provider.Match("claude-3-opus") {
		t.Error("Should match claude-3-opus")
	}
	if !provider.Match("claude-2.1") {
		t.Error("Should match claude-2.1")
	}
	if provider.Match("gpt-4") {
		t.Error("Should not match gpt-4")
	}
}

func TestDeepSeekProvider(t *testing.T) {
	provider := NewDeepSeekProvider("")

	if provider.Name() != "deepseek" {
		t.Errorf("Expected name 'deepseek', got '%s'", provider.Name())
	}

	if !provider.Match("deepseek-chat") {
		t.Error("Should match deepseek-chat")
	}
	if !provider.Match("deepseek-coder-33b") {
		t.Error("Should match deepseek-coder-33b")
	}
}

func TestOllamaProvider(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434")

	if provider.Name() != "ollama" {
		t.Errorf("Expected name 'ollama', got '%s'", provider.Name())
	}

	// Test matching
	ollamaModels := []string{
		"llama-2-70b",
		"llama2",
		"mistral-7b",
		"mistral",
		"qwen-14b",
		"codellama",
		"phi-2",
		"gemma-7b",
		"mixtral-8x7b",
	}

	for _, model := range ollamaModels {
		if !provider.Match(model) {
			t.Errorf("Should match %s", model)
		}
	}
}

func TestProviderPriority(t *testing.T) {
	registry := NewProviderRegistry()

	// Register in wrong order
	registry.Register(NewOpenAIProvider("", "")) // Priority 100
	registry.Register(NewClaudeProvider(""))     // Priority 10
	registry.Register(NewOllamaProvider(""))     // Priority 15

	// Claude should be first (lowest priority number)
	providers := registry.ListProviders()
	if providers[0] != "anthropic" {
		t.Errorf("Expected first provider to be anthropic, got %s", providers[0])
	}
}

func TestConvertToClaudeRequest(t *testing.T) {
	req := &ChatCompletionRequest{
		Model: "claude-3-opus",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	claudeReq := convertToClaudeRequest(req)

	if claudeReq["model"] != "claude-3-opus" {
		t.Errorf("Expected model claude-3-opus, got %v", claudeReq["model"])
	}

	if claudeReq["system"] != "You are a helpful assistant." {
		t.Errorf("System prompt not extracted correctly")
	}

	messages := claudeReq["messages"].([]map[string]string)
	if len(messages) != 1 { // Only user message, system is separate
		t.Errorf("Expected 1 message (user only), got %d", len(messages))
	}

	if messages[0]["role"] != "user" || messages[0]["content"] != "Hello!" {
		t.Error("User message not converted correctly")
	}
}

func TestConvertToOllamaRequest(t *testing.T) {
	req := &ChatCompletionRequest{
		Model: "llama2",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello!"},
		},
		Temperature: 0.5,
		TopP:        0.9,
	}

	ollamaReq := convertToOllamaRequest(req)

	if ollamaReq["model"] != "llama2" {
		t.Errorf("Expected model llama2, got %v", ollamaReq["model"])
	}

	if ollamaReq["stream"] != false {
		t.Error("Stream should be false")
	}

	options := ollamaReq["options"].(map[string]interface{})
	if options["temperature"] != 0.5 {
		t.Errorf("Expected temperature 0.5, got %v", options["temperature"])
	}
}
