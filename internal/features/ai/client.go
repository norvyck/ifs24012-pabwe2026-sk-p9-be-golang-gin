package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const systemPromptText = `Anda adalah penulis artikel blog profesional berbahasa Indonesia. Balas HANYA dengan objek JSON valid berformat: {"title":"...","category":"...","excerpt":"...","content":"...markdown..."}.`

type OpenAIClient interface {
	GenerateArticle(prompt string) (*GeneratedPostResponse, error)
}

type HTTPOpenAIClient struct {
	Endpoint   string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func NewHTTPOpenAIClient(endpoint, apiKey, model string) *HTTPOpenAIClient {
	return &HTTPOpenAIClient{
		Endpoint:   strings.TrimRight(endpoint, "/"),
		APIKey:     apiKey,
		Model:      model,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenConfig struct {
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  geminiGenConfig `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (c *HTTPOpenAIClient) GenerateArticle(prompt string) (*GeneratedPostResponse, error) {
	if strings.Contains(c.Endpoint, "generativelanguage.googleapis.com") {
		return c.generateWithGemini(prompt)
	}
	return c.generateWithOpenAI(prompt)
}

func (c *HTTPOpenAIClient) generateWithOpenAI(prompt string) (*GeneratedPostResponse, error) {
	payload := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPromptText},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
	}

	rawBody, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, c.Endpoint+"/chat/completions", bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, formatAPIError("openai api", resp)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, err
	}
	if len(chatResp.Choices) == 0 {
		return nil, errors.New("openai api returned empty choices")
	}

	return parseArticleJSON(chatResp.Choices[0].Message.Content)
}

func (c *HTTPOpenAIClient) generateWithGemini(prompt string) (*GeneratedPostResponse, error) {
	base := strings.Split(c.Endpoint, "/v1beta")[0]
	targetURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent", strings.TrimRight(base, "/"), c.Model)

	payload := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPromptText}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenConfig{
			Temperature:      0.7,
			ResponseMimeType: "application/json",
		},
	}

	rawBody, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("x-goog-api-key", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, formatAPIError("gemini api", resp)
	}

	var gemResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gemResp); err != nil {
		return nil, err
	}
	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini api returned empty candidates")
	}

	return parseArticleJSON(gemResp.Candidates[0].Content.Parts[0].Text)
}

func formatAPIError(provider string, resp *http.Response) error {
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
	msg := strings.TrimSpace(string(errBody))
	if msg != "" {
		return fmt.Errorf("%s returned status %d: %s", provider, resp.StatusCode, msg)
	}
	return fmt.Errorf("%s returned status %d", provider, resp.StatusCode)
}

func parseArticleJSON(raw string) (*GeneratedPostResponse, error) {
	cleaned := cleanJSONFence(raw)
	var generated GeneratedPostResponse
	if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
		return nil, fmt.Errorf("failed to parse generated article json: %w", err)
	}
	return &generated, nil
}

func cleanJSONFence(s string) string {
	trimmed := strings.TrimSpace(s)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}
