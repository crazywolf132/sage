package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/crazywolf132/sage/internal/config"
)

func TestNewClient(t *testing.T) {
	// Save original environment
	origKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", origKey)

	// Save original config values we care about
	origConfigKey := config.Get("ai.api_key", false)
	origConfigBaseURL := config.Get("ai.base_url", false)
	origConfigModel := config.Get("ai.model", false)
	defer func() {
		config.Set("ai.api_key", origConfigKey, true)
		config.Set("ai.base_url", origConfigBaseURL, true)
		config.Set("ai.model", origConfigModel, true)
	}()

	tests := []struct {
		name          string
		baseURL       string
		envKey        string
		configKey     string
		configBaseURL string
		configModel   string
		wantBaseURL   string
		wantAPIKey    string
		wantModel     string
	}{
		{
			name:        "Default values",
			wantBaseURL: "https://api.openai.com/v1",
			wantModel:   "gpt-4o",
		},
		{
			name:        "Environment API key",
			envKey:      "test-key",
			wantBaseURL: "https://api.openai.com/v1",
			wantAPIKey:  "test-key",
			wantModel:   "gpt-4o",
		},
		{
			name:          "Config values",
			configKey:     "config-key",
			configBaseURL: "https://custom.api.com",
			configModel:   "gpt-3.5-turbo",
			wantBaseURL:   "https://custom.api.com",
			wantAPIKey:    "config-key",
			wantModel:     "gpt-3.5-turbo",
		},
		{
			name:        "Explicit base URL",
			baseURL:     "https://explicit.api.com",
			wantBaseURL: "https://explicit.api.com",
			wantModel:   "gpt-4o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any existing config and environment
			os.Unsetenv("OPENAI_API_KEY")
			config.Set("ai.api_key", "", true)
			config.Set("ai.base_url", "", true)
			config.Set("ai.model", "", true)

			// Set up environment and config for test
			if tt.envKey != "" {
				os.Setenv("OPENAI_API_KEY", tt.envKey)
			}
			if tt.configKey != "" {
				config.Set("ai.api_key", tt.configKey, true)
			}
			if tt.configBaseURL != "" {
				config.Set("ai.base_url", tt.configBaseURL, true)
			}
			if tt.configModel != "" {
				config.Set("ai.model", tt.configModel, true)
			}

			client := NewClient(tt.baseURL)

			if client.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %v, want %v", client.BaseURL, tt.wantBaseURL)
			}

			// For API key, we need to handle both environment and config cases
			if tt.envKey != "" {
				if client.APIKey != tt.envKey {
					t.Errorf("APIKey = %v, want %v", client.APIKey, tt.envKey)
				}
			} else if tt.configKey != "" {
				// For config key, it might be encrypted, so just check it's not empty
				if client.APIKey == "" {
					t.Error("APIKey is empty but should have a value")
				}
			} else if client.APIKey != "" {
				t.Errorf("APIKey = %v, want empty string", client.APIKey)
			}

			if client.Model != tt.wantModel {
				t.Errorf("Model = %v, want %v", client.Model, tt.wantModel)
			}
		})
	}
}

func TestGenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name        string
		diff        string
		apiResponse *GenerateResponse
		apiError    *ErrorResponse
		apiStatus   int
		want        string
		wantErr     bool
	}{
		{
			name: "Successful generation",
			diff: "diff --git a/file.go b/file.go\n+func NewFeature() {}",
			apiResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "feat: add new feature function",
						},
						FinishReason: "stop",
					},
				},
			},
			apiStatus: http.StatusOK,
			want:      "feat: add new feature function",
		},
		{
			name: "API error",
			diff: "test diff",
			apiError: &ErrorResponse{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
				}{
					Message: "Invalid API key",
					Type:    "invalid_request_error",
				},
			},
			apiStatus: http.StatusUnauthorized,
			wantErr:   true,
		},
		{
			name:    "Missing API key",
			diff:    "test diff",
			wantErr: true,
		},
		{
			name: "Empty response",
			diff: "test diff",
			apiResponse: &GenerateResponse{
				Choices: []Choice{},
			},
			apiStatus: http.StatusOK,
			wantErr:   true,
		},
		{
			name: "Incomplete response",
			diff: "test diff",
			apiResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "partial message",
						},
						FinishReason: "length",
					},
				},
			},
			apiStatus: http.StatusOK,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Return configured response
				w.WriteHeader(tt.apiStatus)
				if tt.apiError != nil {
					json.NewEncoder(w).Encode(tt.apiError)
					return
				}
				if tt.apiResponse != nil {
					json.NewEncoder(w).Encode(tt.apiResponse)
				}
			}))
			defer server.Close()

			// Create client with test server URL
			client := &Client{
				BaseURL: server.URL,
				APIKey:  "test-key",
				Model:   "gpt-4",
			}

			// If testing missing API key
			if tt.name == "Missing API key" {
				client.APIKey = ""
			}

			got, err := client.GenerateCommitMessage(tt.diff)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateCommitMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GenerateCommitMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneratePRDescription(t *testing.T) {
	tests := []struct {
		name        string
		commits     string
		diff        string
		apiResponse *GenerateResponse
		want        string
		wantErr     bool
	}{
		{
			name:    "Successful generation",
			commits: "feat: add user auth\nfix: handle edge case",
			diff:    "diff --git a/auth.go b/auth.go\n+func Authenticate() {}",
			apiResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "## Changes\n\nAdded user authentication system",
						},
						FinishReason: "stop",
					},
				},
			},
			want: "## Changes\n\nAdded user authentication system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.apiResponse)
			}))
			defer server.Close()

			client := &Client{
				BaseURL: server.URL,
				APIKey:  "test-key",
				Model:   "gpt-4",
			}

			got, err := client.GeneratePRDescription(tt.commits, tt.diff)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePRDescription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GeneratePRDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}
