package ai

type GeneratePostRequest struct {
	Topic            string `json:"topic" binding:"required"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}

type SearchSource struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type GeneratedPostResponse struct {
	Title    string         `json:"title"`
	Category string         `json:"category"`
	Excerpt  string         `json:"excerpt"`
	Content  string         `json:"content"`
	Sources  []SearchSource `json:"sources"`
}
