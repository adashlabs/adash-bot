package bot

import (
	"bufio"
	_ "embed"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var wordPattern = regexp.MustCompile(`^[aâbcçdefgğhıiîjklmnoöprsştuüûvyz]{2,30}$`)
var dictionaryOnce sync.Once
var dictionaryWords []string
var dictionaryInitials map[rune]bool

//go:embed assets/turkish_dictionary.txt
var dictionaryData string

func normalizeWord(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "İ", "i")
	s = strings.ReplaceAll(s, "I", "ı")
	return strings.ToLower(s)
}
func loadDictionary() {
	dictionaryWords = nil
	dictionaryInitials = make(map[rune]bool)
	scan := bufio.NewScanner(strings.NewReader(dictionaryData))
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		word := strings.Fields(line)[0]
		word = normalizeWord(strings.TrimSuffix(word, "'"))
		if wordPattern.MatchString(word) {
			dictionaryWords = append(dictionaryWords, word)
			if runes := []rune(word); len(runes) > 0 {
				dictionaryInitials[runes[0]] = true
			}
		}
	}
	sort.Strings(dictionaryWords)
}
func isDictionaryWord(word string) bool {
	dictionaryOnce.Do(loadDictionary)
	i := sort.SearchStrings(dictionaryWords, word)
	return i < len(dictionaryWords) && dictionaryWords[i] == word
}
func requiredWordInitial(lastWord string) rune {
	dictionaryOnce.Do(loadDictionary)
	runes := []rune(normalizeWord(lastWord))
	for index := len(runes) - 1; index >= 0; index-- {
		if dictionaryInitials[runes[index]] {
			return runes[index]
		}
	}
	if len(runes) > 0 {
		return runes[len(runes)-1]
	}
	return 0
}

func (b *Bot) temporaryGameEmbed(channelID string, em *discordgo.MessageEmbed, lifetime time.Duration) {
	message, _ := b.dg.ChannelMessageSendEmbed(channelID, em)
	if message != nil {
		time.AfterFunc(lifetime, func() { _ = b.dg.ChannelMessageDelete(message.ChannelID, message.ID) })
	}
}
func (b *Bot) handleGame(m *discordgo.MessageCreate) bool {
	s, e := b.db.Settings(m.GuildID)
	if e != nil {
		return false
	}
	counting := s.CountingEnabled && s.CountingChannelID.Valid && s.CountingChannelID.String == m.ChannelID
	word := s.WordChainEnabled && s.WordChainChannelID.Valid && s.WordChainChannelID.String == m.ChannelID
	if !counting && !word {
		return false
	}
	lockAny, _ := b.gameLocks.LoadOrStore(m.GuildID, &sync.Mutex{})
	lock := lockAny.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	if counting {
		b.counting(m)
	} else {
		b.wordChain(m)
	}
	return true
}
func (b *Bot) rejectGame(m *discordgo.MessageCreate, reason string) {
	if b.db.ConfigBool(m.GuildID, "game_delete_invalid", true) {
		_ = b.dg.ChannelMessageDelete(m.ChannelID, m.ID)
	}
	em := &discordgo.MessageEmbed{
		Title:       "❌ Geçersiz Hamle",
		Description: "<@" + m.Author.ID + "> " + reason,
		Color:       colorDanger,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Oyun kaldığı yerden devam ediyor."},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	b.temporaryGameEmbed(m.ChannelID, em, 5500*time.Millisecond)
}
func (b *Bot) counting(m *discordgo.MessageCreate) {
	g, e := b.db.Game(m.GuildID)
	if e != nil {
		return
	}
	expected := g.CountingValue + 1
	content := strings.TrimSpace(m.Content)
	value, ok := parsePositive(content)
	same := g.CountingUserID.Valid && g.CountingUserID.String == m.Author.ID
	if !ok || value != expected || same {
		reset := b.db.ConfigBool(m.GuildID, "counting_reset_on_error", true)
		if reset {
			_ = b.db.ResetGame(m.GuildID, "counting")
		}
		if same {
			b.rejectGame(m, "aynı kişi art arda sayamaz."+str(reset, " Oyun **1**'den yeniden başladı.", ""))
		} else {
			b.rejectGame(m, "sıradaki sayı **"+itoa64(expected)+"** olmalıydı."+str(reset, " Oyun **1**'den yeniden başladı.", ""))
		}
		return
	}
	_ = b.db.SetCount(m.GuildID, expected, m.Author.ID)
	emoji := "✅"
	if expected%100 == 0 {
		emoji = "💯"
	}
	_ = b.dg.MessageReactionAdd(m.ChannelID, m.ID, emoji)
}
func parsePositive(s string) (int64, bool) {
	if !regexp.MustCompile(`^\d+$`).MatchString(s) {
		return 0, false
	}
	var n int64
	for _, r := range s {
		n = n*10 + int64(r-'0')
	}
	return n, true
}
func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte(v%10) + '0'
		v /= 10
	}
	return string(b[i:])
}
func (b *Bot) wordChain(m *discordgo.MessageCreate) {
	word := normalizeWord(m.Content)
	minLen := b.db.ConfigInt(m.GuildID, "word_min_length", 2)
	if len([]rune(word)) < minLen || !wordPattern.MatchString(word) {
		b.rejectGame(m, "yalnızca Türkçe tek bir kelime yazmalısın.")
		return
	}
	g, e := b.db.Game(m.GuildID)
	if e != nil {
		return
	}
	if g.WordUserID.Valid && g.WordUserID.String == m.Author.ID {
		b.rejectGame(m, "aynı kişi art arda kelime yazamaz.")
		return
	}
	if g.LastWord.Valid {
		required := requiredWordInitial(g.LastWord.String)
		cur := []rune(word)
		if required != 0 && cur[0] != required {
			lastRunes := []rune(g.LastWord.String)
			reason := "kelime **" + strings.ToUpper(string(required)) + "** harfiyle başlamalı."
			if len(lastRunes) > 0 && required != lastRunes[len(lastRunes)-1] {
				reason = "**" + strings.ToUpper(string(lastRunes[len(lastRunes)-1])) + "** ile başlayan Türkçe kelime olmadığı için kelime **" + strings.ToUpper(string(required)) + "** harfiyle başlamalı."
			}
			b.rejectGame(m, reason)
			return
		}
	}
	if b.db.UsedWord(m.GuildID, word) {
		b.rejectGame(m, "**"+word+"** daha önce kullanıldı.")
		return
	}
	if !isDictionaryWord(word) {
		b.rejectGame(m, "**"+word+"** yerel Türkçe oyun sözlüğünde bulunamadı.")
		return
	}
	if b.db.SetWord(m.GuildID, word, m.Author.ID) == nil {
		_ = b.dg.MessageReactionAdd(m.ChannelID, m.ID, "✅")
		runes := []rune(word)
		if len(runes) > 1 {
			next := requiredWordInitial(word)
			last := runes[len(runes)-1]
			if next != 0 && next != last {
				_ = b.dg.MessageReactionAdd(m.ChannelID, m.ID, "🔄")
				em := embed(
					"🔄 Harf Kuralı",
					"**"+strings.ToUpper(string(last))+"** ile başlayan Türkçe kelime bulunmadığı için sıradaki kelime **"+strings.ToUpper(string(next))+"** harfiyle başlamalı.",
					colorPrimary,
				)
				b.temporaryGameEmbed(m.ChannelID, em, 12*time.Second)
			}
		}
	}
}
