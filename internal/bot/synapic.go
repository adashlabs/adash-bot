package bot

import (
	"encoding/json"
	"net/http"
	"net/url"
)

func (b *Bot) synapicSearch(client *http.Client, query string) []searchResult {
	if b.cfg.SynapicKey == "" {
		return nil
	}
	endpoint := "https://api.synapicsearch.com/api/search?q=" + url.QueryEscape(query) + "&apikey=" + url.QueryEscape(b.cfg.SynapicKey)
	res, e := client.Get(endpoint)
	if e != nil {
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil
	}
	var raw struct {
		Results []map[string]any `json:"results"`
	}
	if json.NewDecoder(res.Body).Decode(&raw) != nil {
		return nil
	}
	out := make([]searchResult, 0, len(raw.Results))
	for _, item := range raw.Results {
		pick := func(keys ...string) string {
			for _, key := range keys {
				if value, ok := item[key].(string); ok && value != "" {
					return value
				}
			}
			return ""
		}
		target := pick("url", "link", "source_url")
		if target == "" {
			continue
		}
		out = append(out, searchResult{Title: pick("title", "name"), Snippet: pick("description", "text", "snippet"), URL: target})
		if len(out) >= 50 {
			break
		}
	}
	return out
}
