package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/adashlabs/adash-bot/internal/config"
	"github.com/adashlabs/adash-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

const defaultSystemPrompt = "Sen Adash adlı sıcak, samimi ve doğal konuşan bir Discord sohbet arkadaşısın. Temel amacın bilgi dökmek değil; kullanıcıyla gerçek bir sohbet kurmak, onun üslubuna uyum sağlamak ve gerektiğinde sohbeti nazikçe ilerletmektir. Türkçe konuş; kısa, canlı ve insani yanıtlar ver. Bilmediğin bir konuda uydurma. Toplu etiket, rol etiketi veya zararlı içerik üretme."

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type chatHistory struct {
	Messages []chatMessage
	Updated  time.Time
}
type aiClient struct {
	cfg       config.Config
	db        *database.DB
	http      *http.Client
	mu        sync.Mutex
	histories map[string]chatHistory
	cooldowns map[string]time.Time
}

func newAI(cfg config.Config, db *database.DB) *aiClient {
	return &aiClient{cfg: cfg, db: db, http: &http.Client{Timeout: cfg.OpenAITimeout}, histories: map[string]chatHistory{}, cooldowns: map[string]time.Time{}}
}
func (a *aiClient) configured() bool { return a.cfg.OpenAIBaseURL != "" && a.cfg.OpenAIModel != "" }
func (a *aiClient) cleanup(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for key, expiry := range a.cooldowns {
		if now.After(expiry) {
			delete(a.cooldowns, key)
		}
	}
	for key, history := range a.histories {
		if now.Sub(history.Updated) > 30*time.Minute {
			delete(a.histories, key)
		}
	}
}

func (a *aiClient) handle(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if !a.db.ConfigBool(m.GuildID, "ai_enabled", true) {
		return false
	}
	if !a.configured() {
		_, _ = s.ChannelMessageSendReply(m.ChannelID, "Yapay zekâ yapılandırılmamış. OPENAI_BASE_URL, OPENAI_API_KEY ve OPENAI_MODEL gerekli.", m.Reference())
		return true
	}
	prompt := m.Content
	if s.State.User != nil {
		prompt = strings.ReplaceAll(prompt, "<@"+s.State.User.ID+">", "")
		prompt = strings.ReplaceAll(prompt, "<@!"+s.State.User.ID+">", "")
	}
	prompt = regexp.MustCompile(`<@&\d+>`).ReplaceAllString(prompt, "[rol]")
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		_, _ = s.ChannelMessageSendReply(m.ChannelID, "Bana bir soru da yazmalısın.", m.Reference())
		return true
	}
	key := m.GuildID + ":" + m.Author.ID
	a.mu.Lock()
	until := a.cooldowns[key]
	if time.Now().Before(until) {
		a.mu.Unlock()
		_, _ = s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Yeni soru için %d saniye bekle.", int(time.Until(until).Seconds())+1), m.Reference())
		return true
	}
	a.cooldowns[key] = time.Now().Add(10 * time.Second)
	historyKey := m.GuildID + ":" + m.ChannelID
	h := a.histories[historyKey]
	if time.Since(h.Updated) > 30*time.Minute {
		h.Messages = nil
	}
	a.mu.Unlock()
	system := a.db.ConfigString(m.GuildID, "ai_system_prompt", valueOr(a.cfg.OpenAISystemPrompt, defaultSystemPrompt))
	messages := append([]chatMessage{{Role: "system", Content: system}}, h.Messages...)
	userText := m.Author.Username + ": " + trunc(prompt, 4000)
	messages = append(messages, chatMessage{Role: "user", Content: userText})
	go func() {
		_ = s.ChannelTyping(m.ChannelID)
		answer, e := a.complete(messages)
		if e != nil {
			_, _ = s.ChannelMessageSendReply(m.ChannelID, "Yapay zekâ servisine şu an ulaşılamıyor. URL, model ve API anahtarını kontrol et.", m.Reference())
			return
		}
		answer = safeAIText(answer)
		parts := chunks(answer, 1900)
		if len(parts) == 0 {
			return
		}
		_, _ = s.ChannelMessageSendReply(m.ChannelID, parts[0], m.Reference())
		for _, x := range parts[1:] {
			_, _ = s.ChannelMessageSend(m.ChannelID, x)
		}
		a.mu.Lock()
		updated := append(h.Messages, chatMessage{Role: "user", Content: userText}, chatMessage{Role: "assistant", Content: answer})
		if len(updated) > 12 {
			updated = updated[len(updated)-12:]
		}
		a.histories[historyKey] = chatHistory{Messages: updated, Updated: time.Now()}
		if len(a.histories) > 500 {
			for k := range a.histories {
				delete(a.histories, k)
				break
			}
		}
		a.mu.Unlock()
	}()
	return true
}
func (a *aiClient) complete(messages []chatMessage) (string, error) {
	endpoint := strings.TrimRight(a.cfg.OpenAIBaseURL, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	payload := map[string]any{"model": a.cfg.OpenAIModel, "messages": messages, "temperature": a.cfg.OpenAITemperature, "max_tokens": a.cfg.OpenAIMaxTokens}
	raw, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.OpenAITimeout)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if e != nil {
		return "", e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.OpenAIKey)
	res, e := a.http.Do(req)
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 400))
		return "", fmt.Errorf("AI HTTP %d: %s", res.StatusCode, msg)
	}
	var out struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
	}
	if e = json.NewDecoder(res.Body).Decode(&out); e != nil {
		return "", e
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("boş AI yanıtı")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
func safeAIText(value string) string {
	value = safeText(value)
	return regexp.MustCompile(`<@!?\\d+>`).ReplaceAllStringFunc(value, func(mention string) string { return strings.Replace(mention, "@", "@\\u200b", 1) })
}

func chunks(s string, n int) []string {
	var out []string
	s = strings.TrimSpace(s)
	for len([]rune(s)) > n {
		r := []rune(s)
		cut := n
		for cut > n/2 && r[cut] != '\n' && r[cut] != ' ' {
			cut--
		}
		if cut <= n/2 {
			cut = n
		}
		out = append(out, string(r[:cut]))
		s = strings.TrimSpace(string(r[cut:]))
	}
	if s != "" {
		out = append(out, s)
	}
	return out
}
