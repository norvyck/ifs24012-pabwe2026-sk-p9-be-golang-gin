package ai

import (
	"fmt"
	"strings"
)

type Service struct {
	searchClient SearchClient
	aiClient     OpenAIClient
}

func NewService(searchClient SearchClient, aiClient OpenAIClient) *Service {
	return &Service{
		searchClient: searchClient,
		aiClient:     aiClient,
	}
}

func (s *Service) GeneratePost(req GeneratePostRequest) (*GeneratedPostResponse, error) {
	topic := strings.TrimSpace(req.Topic)
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Umum"
	}

	sources := []SearchSource{}
	if req.IncludeWebSearch && s.searchClient != nil {
		if found, err := s.searchClient.Search(topic); err == nil && len(found) > 0 {
			sources = found
		}
	}

	prompt := buildPrompt(topic, category, sources)
	result, err := s.aiClient.GenerateArticle(prompt)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(result.Category) == "" {
		result.Category = category
	}
	result.Sources = sources
	return result, nil
}

func buildPrompt(topic, category string, sources []SearchSource) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Tuliskan artikel blog mendalam tentang topik: %q dalam kategori %q.\n", topic, category)
	b.WriteString("Sertakan judul menarik, ringkasan (excerpt) 1-2 kalimat, dan isi lengkap berformat Markdown terstruktur.\n")

	if len(sources) > 0 {
		b.WriteString("\nGunakan referensi hasil penelusuran berikut sebagai fakta pendukung dan cantumkan sitasi:\n")
		for i, src := range sources {
			fmt.Fprintf(&b, "%d. %s (%s): %s\n", i+1, src.Title, src.URL, src.Snippet)
		}
	}
	return b.String()
}
