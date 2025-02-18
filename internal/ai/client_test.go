package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockHTTPClient is a mock HTTP client for testing
type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// createMockResponse creates a mock response with the given status code and body
func createMockResponse(statusCode int, body interface{}) *http.Response {
	jsonBody, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBuffer(jsonBody)),
	}
}

// mockConfig is a mock config for testing
type mockConfig struct {
	values map[string]string
}

func (m *mockConfig) Get(key string, required bool) string {
	if val, ok := m.values[key]; ok {
		return val
	}
	return ""
}

// setupTest sets up the test environment
func setupTest(t *testing.T) func() {
	originalClient := http.DefaultClient
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")

	// Reset client and environment after test
	return func() {
		http.DefaultClient = originalClient
		if originalAPIKey != "" {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		}
	}
}

func TestNewClient(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name          string
		baseURL       string
		envAPIKey     string
		configValues  map[string]string
		expectedURL   string
		expectedKey   string
		expectedModel string
	}{
		{
			name:          "default values",
			baseURL:       "",
			envAPIKey:     "",
			configValues:  map[string]string{},
			expectedURL:   "https://api.openai.com/v1",
			expectedKey:   "",
			expectedModel: "gpt-4o",
		},
		{
			name:          "custom base URL",
			baseURL:       "https://custom.api.com",
			envAPIKey:     "test-key",
			configValues:  map[string]string{},
			expectedURL:   "https://custom.api.com",
			expectedKey:   "test-key",
			expectedModel: "gpt-4o",
		},
		{
			name:    "config values",
			baseURL: "",
			configValues: map[string]string{
				"ai.base_url": "https://config.api.com",
				"ai.api_key":  "config-key",
				"ai.model":    "custom-model",
			},
			expectedURL:   "https://config.api.com",
			expectedKey:   "config-key",
			expectedModel: "custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock config
			mockCfg := &mockConfig{values: tt.configValues}

			if tt.envAPIKey != "" {
				os.Setenv("OPENAI_API_KEY", tt.envAPIKey)
			}

			client := NewClient(tt.baseURL, mockCfg)

			assert.Equal(t, tt.expectedURL, client.BaseURL)
			assert.Equal(t, tt.expectedKey, client.APIKey)
			assert.Equal(t, tt.expectedModel, client.Model)

			if tt.envAPIKey != "" {
				os.Unsetenv("OPENAI_API_KEY")
			}
		})
	}
}

func TestGenerateCommitMessage(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		diff           string
		mockResponse   *GenerateResponse
		mockStatusCode int
		mockErr        error
		expectedMsg    string
		expectError    bool
	}{
		{
			name: "successful generation",
			diff: "test diff",
			mockResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "feat: add user authentication",
						},
						FinishReason: "stop",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedMsg:    "feat: add user authentication",
			expectError:    false,
		},
		{
			name:           "missing API key",
			diff:           "test diff",
			mockStatusCode: http.StatusOK,
			expectError:    true,
		},
		{
			name:           "API error",
			diff:           "test diff",
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				BaseURL:    "https://api.test.com",
				APIKey:     "test-key",
				Model:      "gpt-4",
				config:     &mockConfig{},
				httpClient: &http.Client{},
			}

			if tt.name == "missing API key" {
				client.APIKey = ""
			}

			if !tt.expectError {
				// Create a mock response
				resp := createMockResponse(tt.mockStatusCode, tt.mockResponse)
				// Create a custom transport that returns our mock response
				mockClient := &http.Client{
					Transport: &mockTransport{
						response: resp,
						err:      tt.mockErr,
					},
				}
				client.SetHTTPClient(mockClient)
			}

			msg, err := client.GenerateCommitMessage(tt.diff)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMsg, msg)
		})
	}
}

func TestGeneratePRDescription(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		commits        string
		diff           string
		mockResponse   *GenerateResponse
		mockStatusCode int
		mockErr        error
		expectedDesc   string
		expectError    bool
	}{
		{
			name:    "successful generation",
			commits: "test commits",
			diff:    "test diff",
			mockResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "## Summary\nTest PR description",
						},
						FinishReason: "stop",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedDesc:   "## Summary\nTest PR description",
			expectError:    false,
		},
		{
			name:        "missing API key",
			commits:     "test commits",
			diff:        "test diff",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				BaseURL:    "https://api.test.com",
				APIKey:     "test-key",
				Model:      "gpt-4",
				config:     &mockConfig{},
				httpClient: &http.Client{},
			}

			if tt.name == "missing API key" {
				client.APIKey = ""
			}

			if !tt.expectError {
				// Create a mock response
				resp := createMockResponse(tt.mockStatusCode, tt.mockResponse)
				// Create a custom transport that returns our mock response
				mockClient := &http.Client{
					Transport: &mockTransport{
						response: resp,
						err:      tt.mockErr,
					},
				}
				client.SetHTTPClient(mockClient)
			}

			desc, err := client.GeneratePRDescription(tt.commits, tt.diff)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedDesc, desc)
		})
	}
}

func TestGeneratePRLabels(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		commits        string
		diff           string
		mockResponse   *GenerateResponse
		mockStatusCode int
		mockErr        error
		expectedLabels []string
		expectError    bool
	}{
		{
			name:    "successful generation",
			commits: "test commits",
			diff:    "test diff",
			mockResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "feature, bug, documentation",
						},
						FinishReason: "stop",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedLabels: []string{"feature", "bug", "documentation"},
			expectError:    false,
		},
		{
			name:    "invalid labels filtered",
			commits: "test commits",
			diff:    "test diff",
			mockResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "feature, invalid-label, documentation",
						},
						FinishReason: "stop",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedLabels: []string{"feature", "documentation"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				BaseURL:    "https://api.test.com",
				APIKey:     "test-key",
				Model:      "gpt-4",
				config:     &mockConfig{},
				httpClient: &http.Client{},
			}

			// Create a mock response
			resp := createMockResponse(tt.mockStatusCode, tt.mockResponse)
			// Create a custom transport that returns our mock response
			mockClient := &http.Client{
				Transport: &mockTransport{
					response: resp,
					err:      tt.mockErr,
				},
			}
			client.SetHTTPClient(mockClient)

			labels, err := client.GeneratePRLabels(tt.commits, tt.diff)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLabels, labels)
		})
	}
}

func TestGeneratePRTitle(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		commits        string
		diff           string
		mockResponse   *GenerateResponse
		mockStatusCode int
		mockErr        error
		expectedTitle  string
		expectError    bool
	}{
		{
			name:    "successful generation",
			commits: "test commits",
			diff:    "test diff",
			mockResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "feat: add user authentication system",
						},
						FinishReason: "stop",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedTitle:  "feat: add user authentication system",
			expectError:    false,
		},
		{
			name:    "breaking change",
			commits: "test commits",
			diff:    "test diff",
			mockResponse: &GenerateResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "feat!: breaking change in API",
						},
						FinishReason: "stop",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedTitle:  "feat!: breaking change in API",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				BaseURL:    "https://api.test.com",
				APIKey:     "test-key",
				Model:      "gpt-4",
				config:     &mockConfig{},
				httpClient: &http.Client{},
			}

			// Create a mock response
			resp := createMockResponse(tt.mockStatusCode, tt.mockResponse)
			// Create a custom transport that returns our mock response
			mockClient := &http.Client{
				Transport: &mockTransport{
					response: resp,
					err:      tt.mockErr,
				},
			}
			client.SetHTTPClient(mockClient)

			title, err := client.GeneratePRTitle(tt.commits, tt.diff)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTitle, title)
		})
	}
}

// mockTransport implements http.RoundTripper
type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}
