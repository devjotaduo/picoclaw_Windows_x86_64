package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"picoclaw/pkg/llm"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ---- web_fetch ----

type webFetch struct{}

func (t *webFetch) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name:        "web_fetch",
		Description: "Fetch a URL and return its text content (HTML tags stripped).",
		Parameters:  objectSchema(map[string]any{"url": strProp("Absolute http(s) URL to fetch.")}, "url"),
	}
}

func (t *webFetch) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	if !strings.HasPrefix(a.URL, "http://") && !strings.HasPrefix(a.URL, "https://") {
		return "", fmt.Errorf("url must be http(s)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "PicoClaw/0.1")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFileBytes))
	if err != nil {
		return "", err
	}
	text := stripHTML(string(data))
	return fmt.Sprintf("[%d] %s\n\n%s", resp.StatusCode, a.URL, text), nil
}

// ---- web_search (DuckDuckGo HTML) ----

type webSearch struct{}

func (t *webSearch) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name:        "web_search",
		Description: "Search the web (DuckDuckGo) and return the top result titles, URLs, and snippets.",
		Parameters:  objectSchema(map[string]any{"query": strProp("Search query.")}, "query"),
	}
}

var ddgResult = regexp.MustCompile(`(?s)<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>(.*?)</a>.*?<a[^>]*class="result__snippet"[^>]*>(.*?)</a>`)

func (t *webSearch) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	if strings.TrimSpace(a.Query) == "" {
		return "", fmt.Errorf("empty query")
	}
	endpoint := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(a.Query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (PicoClaw)")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	matches := ddgResult.FindAllStringSubmatch(string(data), 8)
	if len(matches) == 0 {
		return "no results", nil
	}
	var b strings.Builder
	for i, m := range matches {
		title := stripHTML(m[2])
		link := html_unescape(m[1])
		snippet := stripHTML(m[3])
		fmt.Fprintf(&b, "%d. %s\n   %s\n   %s\n", i+1, title, link, snippet)
	}
	return b.String(), nil
}

// ---- helpers ----

var tagRe = regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>`)
var anyTag = regexp.MustCompile(`<[^>]+>`)
var wsRe = regexp.MustCompile(`[ \t]*\n[ \t\n]+`)

func stripHTML(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = anyTag.ReplaceAllString(s, "")
	s = html_unescape(s)
	s = wsRe.ReplaceAllString(s, "\n")
	s = strings.TrimSpace(s)
	if len(s) > maxFileBytes {
		s = s[:maxFileBytes] + "\n... [truncated]"
	}
	return s
}

var entities = strings.NewReplacer(
	"&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'", "&#x27;", "'", "&nbsp;", " ",
)

func html_unescape(s string) string { return entities.Replace(s) }
