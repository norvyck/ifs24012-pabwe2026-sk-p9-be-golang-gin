package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func makeHTTPResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestHTTPSearchClient(t *testing.T) {
	t.Run("empty endpoint returns empty slice", func(t *testing.T) {
		c := NewHTTPSearchClient("", "")
		res, err := c.Search("golang")
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("invalid endpoint url", func(t *testing.T) {
		c := NewHTTPSearchClient("://bad-url", "key")
		_, err := c.Search("golang")
		assert.Error(t, err)
	})

	t.Run("http transport error", func(t *testing.T) {
		c := NewHTTPSearchClient("https://search.local/api", "key")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("network unreachable")
		})}
		_, err := c.Search("golang")
		assert.Error(t, err)
	})

	t.Run("non-200 status", func(t *testing.T) {
		c := NewHTTPSearchClient("https://search.local/api", "my-key")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer my-key", r.Header.Get("Authorization"))
			return makeHTTPResp(http.StatusInternalServerError, ""), nil
		})}
		_, err := c.Search("golang")
		assert.Error(t, err)
	})

	t.Run("invalid json body", func(t *testing.T) {
		c := NewHTTPSearchClient("https://search.local/api", "")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusOK, "not-json"), nil
		})}
		_, err := c.Search("golang")
		assert.Error(t, err)
	})

	t.Run("results field", func(t *testing.T) {
		c := NewHTTPSearchClient("https://search.local/api", "")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			raw, _ := json.Marshal(map[string]any{
				"results": []SearchSource{{Title: "Go Docs", URL: "https://go.dev", Snippet: "Official Go"}},
			})
			return makeHTTPResp(http.StatusOK, string(raw)), nil
		})}
		res, err := c.Search("golang")
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "Go Docs", res[0].Title)
	})

	t.Run("data field", func(t *testing.T) {
		c := NewHTTPSearchClient("https://search.local/api", "")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			raw, _ := json.Marshal(map[string]any{
				"data": []SearchSource{{Title: "Delcom", URL: "https://delcom.org", Snippet: "Info"}},
			})
			return makeHTTPResp(http.StatusOK, string(raw)), nil
		})}
		res, err := c.Search("delcom")
		require.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("empty results", func(t *testing.T) {
		c := NewHTTPSearchClient("https://search.local/api", "")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusOK, `{}`), nil
		})}
		res, err := c.Search("empty")
		require.NoError(t, err)
		assert.Empty(t, res)
	})
}

func TestHTTPOpenAIClient(t *testing.T) {
	t.Run("invalid endpoint url", func(t *testing.T) {
		c := NewHTTPOpenAIClient("://bad", "key", "gpt")
		_, err := c.GenerateArticle("topic")
		assert.Error(t, err)
	})

	t.Run("transport error", func(t *testing.T) {
		c := NewHTTPOpenAIClient("https://ai.local/v1", "key", "gpt")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("connection failed")
		})}
		_, err := c.GenerateArticle("topic")
		assert.Error(t, err)
	})

	t.Run("non-200 status with and without body", func(t *testing.T) {
		c := NewHTTPOpenAIClient("https://ai.local/v1", "sk-1", "gpt")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusUnauthorized, ""), nil
		})}
		_, err := c.GenerateArticle("topic")
		assert.EqualError(t, err, "openai api returned status 401")

		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusBadRequest, `{"error":"bad key"}`), nil
		})}
		_, err = c.GenerateArticle("topic")
		assert.Contains(t, err.Error(), "bad key")
	})

	t.Run("googleapis uses native generateContent with x-goog-api-key", func(t *testing.T) {
		c := NewHTTPOpenAIClient("https://generativelanguage.googleapis.com/v1beta/openai", "AQ.test-key", "gemini-2.5-flash")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent", r.URL.String())
			assert.Equal(t, "AQ.test-key", r.Header.Get("x-goog-api-key"))
			assert.Empty(t, r.Header.Get("Authorization"))
			raw, _ := json.Marshal(geminiResponse{
				Candidates: []struct {
					Content geminiContent `json:"content"`
				}{{Content: geminiContent{Parts: []geminiPart{{Text: `{"title":"Gemini OK","category":"Teknologi","excerpt":"Ex","content":"Body"}`}}}}},
			})
			return makeHTTPResp(http.StatusOK, string(raw)), nil
		})}
		res, err := c.GenerateArticle("topic")
		require.NoError(t, err)
		assert.Equal(t, "Gemini OK", res.Title)

		// Invalid URL
		cBadURL := NewHTTPOpenAIClient("://generativelanguage.googleapis.com/v1beta", "AQ.k", "gemini-2.5-flash")
		_, err = cBadURL.GenerateArticle("topic")
		assert.Error(t, err)

		// Transport error
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("gem net err")
		})}
		_, err = c.GenerateArticle("topic")
		assert.Error(t, err)

		// Non-200 status
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusForbidden, "forbidden"), nil
		})}
		_, err = c.GenerateArticle("topic")
		assert.Contains(t, err.Error(), "gemini api returned status 403")

		// Invalid JSON
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusOK, "bad-json"), nil
		})}
		_, err = c.GenerateArticle("topic")
		assert.Error(t, err)

		// Empty candidates
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusOK, `{"candidates":[]}`), nil
		})}
		_, err = c.GenerateArticle("topic")
		assert.EqualError(t, err, "gemini api returned empty candidates")
	})

	t.Run("invalid response json", func(t *testing.T) {
		c := NewHTTPOpenAIClient("https://ai.local/v1", "sk-1", "gpt")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusOK, "invalid"), nil
		})}
		_, err := c.GenerateArticle("topic")
		assert.Error(t, err)
	})

	t.Run("empty choices", func(t *testing.T) {
		c := NewHTTPOpenAIClient("https://ai.local/v1", "sk-1", "gpt")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return makeHTTPResp(http.StatusOK, `{"choices":[]}`), nil
		})}
		_, err := c.GenerateArticle("topic")
		assert.Error(t, err)
	})

	t.Run("invalid inner article json", func(t *testing.T) {
		c := NewHTTPOpenAIClient("https://ai.local/v1", "sk-1", "gpt")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			raw, _ := json.Marshal(chatResponse{
				Choices: []struct {
					Message chatMessage `json:"message"`
				}{{Message: chatMessage{Content: "not-json"}}},
			})
			return makeHTTPResp(http.StatusOK, string(raw)), nil
		})}
		_, err := c.GenerateArticle("topic")
		assert.Error(t, err)
	})

	t.Run("valid fenced json article", func(t *testing.T) {
		c := NewHTTPOpenAIClient("https://ai.local/v1", "sk-1", "gpt")
		c.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer sk-1", r.Header.Get("Authorization"))
			content := "```json\n{\"title\":\"Judul AI\",\"category\":\"Teknologi\",\"excerpt\":\"Ringkas\",\"content\":\"# Halo\"}\n```"
			raw, _ := json.Marshal(chatResponse{
				Choices: []struct {
					Message chatMessage `json:"message"`
				}{{Message: chatMessage{Content: content}}},
			})
			return makeHTTPResp(http.StatusOK, string(raw)), nil
		})}
		res, err := c.GenerateArticle("topic")
		require.NoError(t, err)
		assert.Equal(t, "Judul AI", res.Title)
		assert.Equal(t, "# Halo", res.Content)
	})
}

type mockSearch struct {
	sources []SearchSource
	err     error
}

func (m *mockSearch) Search(string) ([]SearchSource, error) {
	return m.sources, m.err
}

type mockAI struct {
	res *GeneratedPostResponse
	err error
}

func (m *mockAI) GenerateArticle(string) (*GeneratedPostResponse, error) {
	return m.res, m.err
}

func TestAIServiceAndHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("handler invalid payload", func(t *testing.T) {
		svc := NewService(&mockSearch{}, &mockAI{})
		h := NewHandler(svc)
		r := gin.New()
		r.POST("/api/posts/ai-generate", h.GeneratePost)

		req := httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("handler success with web search and fallback category", func(t *testing.T) {
		svc := NewService(
			&mockSearch{sources: []SearchSource{{Title: "Ref", URL: "https://example.com", Snippet: "Info"}}},
			&mockAI{res: &GeneratedPostResponse{Title: "AI Title", Excerpt: "Ex", Content: "Body"}},
		)
		h := NewHandler(svc)
		r := gin.New()
		r.POST("/api/posts/ai-generate", h.GeneratePost)

		body, _ := json.Marshal(GeneratePostRequest{
			Topic:            "Cloud Computing",
			IncludeWebSearch: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var parsed struct {
			Data GeneratedPostResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &parsed))
		assert.Equal(t, "Umum", parsed.Data.Category)
		assert.Len(t, parsed.Data.Sources, 1)
	})

	t.Run("handler ai error", func(t *testing.T) {
		svc := NewService(
			&mockSearch{err: errors.New("search down")},
			&mockAI{err: errors.New("ai down")},
		)
		h := NewHandler(svc)
		r := gin.New()
		r.POST("/api/posts/ai-generate", h.GeneratePost)

		body, _ := json.Marshal(GeneratePostRequest{
			Topic:            "AI",
			Category:         "Teknologi",
			IncludeWebSearch: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadGateway, w.Code)
	})
}
