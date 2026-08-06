package bot

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/adashlabs/adash-bot/internal/config"
	"github.com/adashlabs/adash-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	cfg            config.Config
	db             *database.DB
	dg             *discordgo.Session
	started        time.Time
	mu             sync.Mutex
	cooldowns      map[string]time.Time
	confirms       map[string]confirmation
	drafts         map[string]*embedDraft
	searches       map[string]*searchSession
	gameLocks      sync.Map
	giveawayTimers map[int64]*time.Timer
	ai             *aiClient
}
type confirmation struct {
	UserID, GuildID, Title, Target, Reason, Details string
	Expires                                         time.Time
	Action                                          func() error
}
type embedDraft struct {
	ChannelID, Content, Title, Description, Color, URL       string
	Image, Thumbnail, Author, AuthorIcon, Footer, FooterIcon string
	Fields                                                   []*discordgo.MessageEmbedField
	Timestamp                                                bool
	Updated                                                  time.Time
}
type searchSession struct {
	OwnerID, Query string
	Results        []searchResult
	Page           int
	Created        time.Time
}
type searchResult struct{ Title, Snippet, URL string }

func New(cfg config.Config, db *database.DB) (*Bot, error) {
	dg, e := discordgo.New("Bot " + cfg.Token)
	if e != nil {
		return nil, e
	}
	b := &Bot{cfg: cfg, db: db, dg: dg, started: time.Now(), cooldowns: map[string]time.Time{}, confirms: map[string]confirmation{}, drafts: map[string]*embedDraft{}, searches: map[string]*searchSession{}, giveawayTimers: map[int64]*time.Timer{}}
	b.ai = newAI(cfg, db)
	dg.State.MaxMessageCount = 0
	dg.State.TrackMembers = false
	dg.State.TrackThreads = false
	dg.State.TrackThreadMembers = false
	dg.State.TrackEmojis = false
	dg.State.TrackStickers = false
	dg.State.TrackVoice = false
	dg.State.TrackPresences = false
	go b.janitor()
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuildMembers
	dg.AddHandler(b.ready)
	dg.AddHandler(b.messageCreate)
	dg.AddHandler(b.interactionCreate)
	dg.AddHandler(b.guildCreate)
	dg.AddHandler(b.memberAdd)
	dg.AddHandler(b.memberRemove)
	return b, nil
}
func (b *Bot) janitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for now := range ticker.C {
		b.mu.Lock()
		for key, expiry := range b.cooldowns {
			if now.After(expiry) {
				delete(b.cooldowns, key)
			}
		}
		for key, item := range b.confirms {
			if now.After(item.Expires) {
				delete(b.confirms, key)
			}
		}
		for key, item := range b.drafts {
			if now.Sub(item.Updated) > 15*time.Minute {
				delete(b.drafts, key)
			}
		}
		for key, item := range b.searches {
			if now.Sub(item.Created) > 10*time.Minute {
				delete(b.searches, key)
			}
		}
		b.mu.Unlock()
		b.ai.cleanup(now)
	}
}

func (b *Bot) Start() error { return b.dg.Open() }
func (b *Bot) Close() {
	b.mu.Lock()
	for _, t := range b.giveawayTimers {
		t.Stop()
	}
	b.mu.Unlock()
	_ = b.dg.Close()
}
func (b *Bot) ready(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("%s olarak bağlandı", r.User.Username)
	_ = s.UpdateGameStatus(0, "a!help • /yardim")
	for _, g := range r.Guilds {
		if full, e := s.Guild(g.ID); e == nil {
			_ = b.db.RegisterGuild(g.ID, full.Name)
		}
	}
	if _, e := s.ApplicationCommandBulkOverwrite(r.Application.ID, "", slashCommands()); e != nil {
		log.Printf("slash komut kaydı: %v", e)
	}
	b.restoreGiveaways()
}
func (b *Bot) guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	_ = b.db.RegisterGuild(g.ID, g.Name)
}
func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot || m.GuildID == "" {
		return
	}
	_ = b.db.RegisterUser(m.Author.ID, m.Author.Username, m.Author.Discriminator)
	if b.handleGame(m) {
		return
	}
	settings, e := b.db.Settings(m.GuildID)
	if e == nil {
		aiChannel := settings.AIChannelID.Valid && settings.AIChannelID.String == m.ChannelID
		mentioned := s.State.User != nil && strings.Contains(m.Content, "<@"+s.State.User.ID+">")
		mentioned = mentioned || (s.State.User != nil && strings.Contains(m.Content, "<@!"+s.State.User.ID+">"))
		if aiChannel || mentioned {
			if b.ai.handle(s, m) {
				return
			}
		}
	}
	prefix := b.db.Prefix(m.GuildID)
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}
	parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(m.Content, prefix)))
	if len(parts) == 0 {
		return
	}
	name := strings.ToLower(parts[0])
	args := parts[1:]
	aliases := map[string]string{"sil": "clear", "temizle": "clear", "kurulum": "setup", "ayarlar": "setup", "yardım": "help", "yardim": "help", "komutlar": "help", "çekiliş": "giveaway", "cekilis": "giveaway", "çekilişyönet": "giveawaymanage", "cekilisyonet": "giveawaymanage", "uyar": "warn", "uyarı": "warn", "uyari": "warn", "uyarılar": "warnings", "uyarilar": "warnings", "uyarıtemizle": "clearwarns", "uyaritemizle": "clearwarns", "yasakla": "ban", "at": "kick", "sustur": "mute", "timeout": "mute", "susturmaaç": "unmute", "yasakaç": "unban", "yasakac": "unban", "vakalar": "cases", "modlog": "cases", "modayar": "modconfig", "kilit": "lock", "yavaşmod": "slowmode", "yavasmod": "slowmode", "kullanıcı": "userinfo", "kullanicibilgi": "userinfo", "sunucu": "serverinfo", "sunucubilgi": "serverinfo", "oyunlar": "games", "oyundurumu": "games", "yazıtura": "coinflip", "yazitura": "coinflip", "sihirliküre": "8ball", "sihirlikure": "8ball", "sekiztop": "8ball", "zar": "roll", "pp": "avatar", "sözlük": "tdk", "sozluk": "tdk", "webara": "wsearch", "itiraz": "appeal", "talep": "ticket", "ticketkurulum": "ticketsetup", "destekkur": "ticketsetup", "embedbuilder": "embed"}
	if x := aliases[name]; x != "" {
		name = x
	}
	key := m.GuildID + ":" + m.Author.ID + ":" + name
	b.mu.Lock()
	until := b.cooldowns[key]
	if time.Now().Before(until) {
		b.mu.Unlock()
		return
	}
	b.cooldowns[key] = time.Now().Add(2 * time.Second)
	b.mu.Unlock()
	b.db.LogCommand(m.GuildID, m.Author.ID, name, strings.Join(args, " "))
	if e := b.runPrefix(s, m, name, args); e != nil {
		log.Printf("komut %s: %v", name, e)
		_, _ = s.ChannelMessageSendEmbed(m.ChannelID, errorEmbed("Komut çalıştırılırken hata oluştu. Yetkileri ve rol sırasını kontrol et."))
	}
}
func (b *Bot) pingEmbed(guild string) *discordgo.MessageEmbed {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	guilds, users, commands := b.db.Stats()
	startedUnix := b.started.Unix()
	return &discordgo.MessageEmbed{
		Title:       "🏓 Bot Durumu",
		Description: "Bot çalışıyor ve Discord bağlantısı aktif.",
		Color:       colorSuccess,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Gecikme", Value: fmt.Sprintf("WebSocket: **%d ms**", b.dg.HeartbeatLatency().Milliseconds()), Inline: true},
			{Name: "RAM", Value: fmt.Sprintf("Kullanım: **%.1f MB**\nAyrılan: **%.1f MB**", float64(mem.HeapAlloc)/1048576, float64(mem.HeapSys)/1048576), Inline: true},
			{Name: "Çalışma süresi", Value: fmt.Sprintf("<t:%d:R>\nBaşlangıç: <t:%d:F>", startedUnix, startedUnix), Inline: false},
			{Name: "Bot verileri", Value: fmt.Sprintf("Sunucu: **%d**\nKullanıcı: **%d**", guilds, users), Inline: true},
			{Name: "İşlem durumu", Value: fmt.Sprintf("Komut: **%d**\nGoroutine: **%d**", commands, runtime.NumGoroutine()), Inline: true},
			{Name: "Çalışma ortamı", Value: fmt.Sprintf("**%s**\nSQLite", runtime.Version()), Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "Prefix: " + b.db.Prefix(guild) + " • Son kontrol"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}
