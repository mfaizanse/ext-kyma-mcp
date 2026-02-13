package saphelp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/chromedp/chromedp"
)

const (
	sapHelpBaseURL        = "https://help.sap.com"
	sapHelpSemanticSearch = "https://help.sap.com/http.svc/semanticsearch"
	contentDivSelector    = `#page`
)

type SAPHelpSemanticSearchRequest struct {
	To                int      `json:"to"`
	IsExactMatch      bool     `json:"isExactMatch"`
	Query             string   `json:"query"`
	SearchType        string   `json:"searchType"`
	KeywordHighlight  bool     `json:"keywordHighlight"`
	SemanticHighlight bool     `json:"semanticHighlight"`
	TransTypes        []string `json:"transTypes"`
	States            []string `json:"states"`
}

type SAPHelpResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

func SAPHelpSemanticSearch(ctx context.Context, query string, maxResults int, locale string, isExactMatch bool) (string, error) {
	searchType := "SEMANTIC"
	keywordHighlight := false
	semanticHighlight := false
	transTypes := []string{"standard", "html"}
	states := []string{"PRODUCTION"}

	requestPayload := SAPHelpSemanticSearchRequest{
		To:                maxResults,
		IsExactMatch:      isExactMatch,
		Query:             query,
		SearchType:        searchType,
		KeywordHighlight:  keywordHighlight,
		SemanticHighlight: semanticHighlight,
		TransTypes:        transTypes,
		States:            states,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to build search request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sapHelpSemanticSearch, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if locale != "" {
		req.Header.Set("Accept-Language", locale)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call SAP Help search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read SAP Help response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			trimmed = "(empty response body)"
		}
		return "", fmt.Errorf("SAP Help search failed with status %d: %s", resp.StatusCode, trimmed)
	}

	content := formatSapHelpSearchResults(ctx, maxResults, locale, body)
	return content, nil
}

func formatSapHelpSearchResults(ctx context.Context, maxResults int, locale string, body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return strings.TrimSpace(string(body))
	}

	results := extractSapHelpResults(payload)
	if len(results) == 0 {
		return "No results found in response."
	}

	if maxResults > len(results) {
		maxResults = len(results)
	}

	output := make([]SAPHelpResult, 0, maxResults)
	for i := 0; i < maxResults; i++ {
		result := results[i]
		title := firstString(result, "title", "Title")
		if title == "" {
			title = fmt.Sprintf("Result %d", i+1)
		}
		url := firstString(result, "url", "Url", "link", "href")
		resolvedURL := normalizeSapHelpURL(url)
		content := ""
		if resolvedURL != "" {
			fetched, err := fetchSapHelpPageContent(ctx, resolvedURL, title, locale)
			if err != nil {
				content = fmt.Sprintf("failed to fetch content: %s", err.Error())
			} else {
				content = fetched
			}
		}
		output = append(output, SAPHelpResult{
			Title:   title,
			URL:     resolvedURL,
			Content: content,
		})
	}

	marshalled, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return strings.TrimSpace(string(body))
	}
	return strings.TrimSpace(string(marshalled))
}

func extractSapHelpResults(payload map[string]any) []map[string]any {
	if results, ok := sliceOfMaps(payload["results"]); ok {
		return results
	}
	if results, ok := sliceOfMaps(payload["items"]); ok {
		return results
	}
	if data, ok := payload["data"].(map[string]any); ok {
		if results, ok := sliceOfMaps(data["results"]); ok {
			return results
		}
		if results, ok := sliceOfMaps(data["items"]); ok {
			return results
		}
	}
	if response, ok := payload["response"].(map[string]any); ok {
		if results, ok := sliceOfMaps(response["results"]); ok {
			return results
		}
		if results, ok := sliceOfMaps(response["items"]); ok {
			return results
		}
	}
	return nil
}

func sliceOfMaps(value any) ([]map[string]any, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	results := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		results = append(results, entry)
	}
	return results, len(results) > 0
}

func firstString(entry map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := entry[key]
		if !ok || value == nil {
			continue
		}
		str, ok := value.(string)
		if ok && strings.TrimSpace(str) != "" {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

func normalizeSapHelpURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "/") {
		return sapHelpBaseURL + trimmed
	}
	return sapHelpBaseURL + "/" + trimmed
}

func fetchSapHelpPageContent(ctx context.Context, pageURL, title, locale string) (string, error) {
	if locale != "" {
		pageURL = appendLocaleParam(pageURL, locale)
	}

	chromeCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	chromeCtx, cancel = context.WithTimeout(chromeCtx, 45*time.Second)
	defer cancel()

	var html string
	err := chromedp.Run(
		chromeCtx,
		chromedp.Navigate(pageURL),
		chromedp.WaitVisible(contentDivSelector, chromedp.ByID),
		chromedp.WaitReady(contentDivSelector, chromedp.ByID),
		chromedp.Poll(fmt.Sprintf(`
			(function() {
				var el = document.querySelector("%s");
				if (!el) return false; // Element doesn't exist yet
				var text = el.innerText.trim();
				return text.length > 0 && text.indexOf("%s") !== -1;
			})()
		`, contentDivSelector, title), nil),
		chromedp.OuterHTML(contentDivSelector, &html, chromedp.ByID),
	)
	if err != nil {
		return "", fmt.Errorf("render page: %w", err)
	}

	if strings.TrimSpace(html) == "" {
		return "", nil
	}

	return convertToMarkdown(html)
}

func appendLocaleParam(pageURL, locale string) string {
	trimmed := strings.TrimSpace(locale)
	if trimmed == "" {
		return pageURL
	}
	separator := "?"
	if strings.Contains(pageURL, "?") {
		separator = "&"
	}
	return pageURL + separator + "locale=" + trimmed
}

func convertToMarkdown(raw string) (string, error) {
	markdown, err := htmltomarkdown.ConvertString(raw)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(markdown), nil
}
