package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) tdk(c *commandContext, word string) error {
	word = normalizeWord(word)
	client := &http.Client{Timeout: 8 * time.Second}
	res, e := client.Get("https://sozluk.gov.tr/gts?ara=" + url.QueryEscape(word))
	if e != nil {
		return fmt.Errorf("TDK servisine ulaşılamadı")
	}
	defer res.Body.Close()
	var rows []struct {
		Madde    string `json:"madde"`
		Anlamlar []struct {
			Anlam      string `json:"anlam"`
			Ozellikler []struct {
				TamAdi string `json:"tam_adi"`
			} `json:"ozelliklerListe"`
		} `json:"anlamlarListe"`
	}
	if e = json.NewDecoder(res.Body).Decode(&rows); e != nil || len(rows) == 0 {
		return fmt.Errorf("kelime TDK sözlüğünde bulunamadı")
	}
	var lines []string
	for _, a := range rows[0].Anlamlar {
		prefix := ""
		if len(a.Ozellikler) > 0 {
			prefix = "*" + a.Ozellikler[0].TamAdi + "* "
		}
		lines = append(lines, fmt.Sprintf("**%d.** %s%s", len(lines)+1, prefix, a.Anlam))
		if len(lines) >= 8 {
			break
		}
	}
	return c.embed(embed("📖 TDK · "+rows[0].Madde, trunc(strings.Join(lines, "\n"), 4000), colorPrimary), row(discordgo.Button{Style: discordgo.LinkButton, Label: "TDK'de Aç", URL: "https://sozluk.gov.tr/?ara=" + url.QueryEscape(word)}))
}
func (b *Bot) webSearch(c *commandContext, q string) error {
	results := []searchResult{}
	client := &http.Client{Timeout: 7 * time.Second}
	ddg := "https://api.duckduckgo.com/?q=" + url.QueryEscape(q) + "&format=json&no_html=1&skip_disambig=1"
	if res, e := client.Get(ddg); e == nil {
		var x struct {
			AbstractText, AbstractURL, Heading string
			RelatedTopics                      []struct{ Text, FirstURL string }
		}
		if json.NewDecoder(res.Body).Decode(&x) == nil {
			if x.AbstractText != "" {
				results = append(results, searchResult{Title: valueOr(x.Heading, q), Snippet: x.AbstractText, URL: x.AbstractURL})
			}
			for _, r := range x.RelatedTopics {
				if r.Text != "" && len(results) < 5 {
					results = append(results, searchResult{Title: trunc(r.Text, 80), Snippet: r.Text, URL: r.FirstURL})
				}
			}
		}
		res.Body.Close()
	}
	wiki := "https://tr.wikipedia.org/w/api.php?action=query&list=search&srsearch=" + url.QueryEscape(q) + "&format=json&origin=*"
	if res, e := client.Get(wiki); e == nil {
		var x struct {
			Query struct {
				Search []struct{ Title, Snippet string }
			}
		}
		if json.NewDecoder(res.Body).Decode(&x) == nil {
			for _, r := range x.Query.Search {
				if len(results) >= 10 {
					break
				}
				results = append(results, searchResult{Title: r.Title, Snippet: stripHTML(r.Snippet), URL: "https://tr.wikipedia.org/wiki/" + url.PathEscape(strings.ReplaceAll(r.Title, " ", "_"))})
			}
		}
		res.Body.Close()
	}
	if len(results) == 0 {
		return fmt.Errorf("sonuç bulunamadı")
	}
	id := token()
	b.mu.Lock()
	b.searches[id] = &searchSession{OwnerID: c.user.ID, Query: q, Results: results, Page: 0, Created: time.Now()}
	b.mu.Unlock()
	em, components := searchPage(q, results, 0, id)
	return c.embed(em, components...)
}
func stripHTML(s string) string {
	for strings.Contains(s, "<") {
		a := strings.Index(s, "<")
		z := strings.Index(s[a:], ">")
		if z < 0 {
			break
		}
		s = s[:a] + s[a+z+1:]
	}
	return s
}
func searchPage(q string, results []searchResult, page int, id string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	r := results[page]
	em := embed(fmt.Sprintf("🔎 %s · %d/%d", trunc(q, 150), page+1, len(results)), "**["+trunc(r.Title, 256)+"]("+r.URL+")**\n"+trunc(r.Snippet, 3500), colorPrimary)
	prev := button("wsearch:prev:"+id, "Geri", discordgo.SecondaryButton, "◀️")
	next := button("wsearch:next:"+id, "İleri", discordgo.SecondaryButton, "▶️")
	prev.Disabled = page == 0
	next.Disabled = page >= len(results)-1
	return em, []discordgo.MessageComponent{row(prev, next)}
}
func (b *Bot) searchComponent(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 3 {
		return nil
	}
	b.mu.Lock()
	x := b.searches[parts[2]]
	if x == nil || time.Since(x.Created) > 10*time.Minute {
		b.mu.Unlock()
		return fmt.Errorf("arama oturumunun süresi doldu")
	}
	if x.OwnerID != userOf(i).ID {
		b.mu.Unlock()
		return fmt.Errorf("bu arama sana ait değil")
	}
	if parts[1] == "prev" && x.Page > 0 {
		x.Page--
	}
	if parts[1] == "next" && x.Page < len(x.Results)-1 {
		x.Page++
	}
	em, components := searchPage(x.Query, x.Results, x.Page, parts[2])
	b.mu.Unlock()
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{em}, Components: components}})
}
