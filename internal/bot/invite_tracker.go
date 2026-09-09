package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) cacheGuildInvites(s *discordgo.Session, guildID string) {
	invites, err := s.GuildInvites(guildID)
	if err != nil {
		return
	}
	cache := make(map[string]int, len(invites))
	for _, inv := range invites {
		cache[inv.Code] = inv.Uses
	}
	b.invitesCache.Store(guildID, cache)
}

func (b *Bot) inviteCreate(s *discordgo.Session, e *discordgo.InviteCreate) {
	if e == nil || e.GuildID == "" {
		return
	}
	v, ok := b.invitesCache.Load(e.GuildID)
	var cache map[string]int
	if ok && v != nil {
		cache = v.(map[string]int)
	} else {
		cache = make(map[string]int)
	}
	cache[e.Code] = e.Uses
	b.invitesCache.Store(e.GuildID, cache)
}

func (b *Bot) inviteDelete(s *discordgo.Session, e *discordgo.InviteDelete) {
	if e == nil || e.GuildID == "" {
		return
	}
	v, ok := b.invitesCache.Load(e.GuildID)
	if ok && v != nil {
		cache := v.(map[string]int)
		delete(cache, e.Code)
		b.invitesCache.Store(e.GuildID, cache)
	}
}

func (b *Bot) trackMemberJoin(s *discordgo.Session, guildID string, member *discordgo.User) (inviterID string, code string) {
	if member == nil || member.Bot {
		return "", ""
	}

	currentInvites, err := s.GuildInvites(guildID)
	if err != nil {
		return "", ""
	}

	var cached map[string]int
	if v, ok := b.invitesCache.Load(guildID); ok && v != nil {
		cached = v.(map[string]int)
	} else {
		cached = make(map[string]int)
	}

	newCache := make(map[string]int, len(currentInvites))
	for _, inv := range currentInvites {
		newCache[inv.Code] = inv.Uses
		oldUses := cached[inv.Code]
		if inv.Uses > oldUses && inviterID == "" {
			if inv.Inviter != nil {
				inviterID = inv.Inviter.ID
			}
			code = inv.Code
		}
	}
	b.invitesCache.Store(guildID, newCache)

	if inviterID != "" {
		accountAge := time.Since(snowflakeTime(member.ID))
		isFake := accountAge < 72*time.Hour // 3 günden yeni hesaplar sahte/şüpheli kabul edilir
		_ = b.db.RecordMemberJoin(guildID, member.ID, inviterID, code, isFake)
	}

	return inviterID, code
}

func (b *Bot) inviteCommand(c *commandContext, args []string) error {
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "ekle", "add":
			return b.inviteAddBonusCommand(c, args[1:], false)
		case "sil", "remove", "cikar", "çıkar":
			return b.inviteAddBonusCommand(c, args[1:], true)
		case "top", "lider", "siralamasi", "sıralaması", "siralama", "sıralama", "leaderboard":
			return b.inviteLeaderboardCommand(c)
		}
	}

	targetID := c.user.ID
	if len(args) > 0 {
		if id := mentionID(args[0]); id != "" {
			targetID = id
		}
	}

	targetUser, err := c.s.User(targetID)
	if err != nil {
		targetUser = c.user
		targetID = c.user.ID
	}

	stats, err := b.db.GetInviteStats(c.guildID, targetID)
	if err != nil {
		return err
	}

	netInvites := stats.Net()

	em := &discordgo.MessageEmbed{
		Title:       "📨 Davet İstatistikleri",
		Description: fmt.Sprintf("<@%s> (`%s`) sunucu davet verileri:\n\n**%s** toplamda **%d** davet yaptı, **%d** kişi sunucudan ayrıldı. Toplam geçerli net davet: **%d**", targetUser.ID, targetUser.Username, targetUser.Username, stats.Regular, stats.Left, netInvites),
		Color:       colorPrimary,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: targetUser.AvatarURL("256")},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "✨ Net Davet",
				Value:  fmt.Sprintf("**%d** geçerli davet", netInvites),
				Inline: false,
			},
			{
				Name:   "📥 Katılan",
				Value:  fmt.Sprintf("**%d** üye", stats.Regular),
				Inline: true,
			},
			{
				Name:   "📤 Ayrılan",
				Value:  fmt.Sprintf("**%d** üye", stats.Left),
				Inline: true,
			},
			{
				Name:   "🤖 Şüpheli / Sahte",
				Value:  fmt.Sprintf("**%d** hesap", stats.Fake),
				Inline: true,
			},
			{
				Name:   "🎁 Bonus Davet",
				Value:  fmt.Sprintf("**%d** davet", stats.Bonus),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Net Davet = Katılan - Ayrılan - Şüpheli + Bonus • Adash Bot",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return c.embed(em)
}

func (b *Bot) inviteAddBonusCommand(c *commandContext, args []string, remove bool) error {
	if err := c.require(discordgo.PermissionManageServer); err != nil {
		return fmt.Errorf("bu komutu kullanmak için Sunucuyu Yönet (Manage Server) yetkisine sahip olmalısın")
	}

	if len(args) < 2 {
		if remove {
			return fmt.Errorf("kullanım: `%sdavetsil @üye <sayı>`", b.db.Prefix(c.guildID))
		}
		return fmt.Errorf("kullanım: `%sdavetekle @üye <sayı>`", b.db.Prefix(c.guildID))
	}

	targetID := mentionID(args[0])
	if targetID == "" {
		return fmt.Errorf("geçerli bir üye etiketle veya ID belirt")
	}

	amount, err := strconv.Atoi(args[1])
	if err != nil || amount <= 0 || amount > 10000 {
		return fmt.Errorf("geçerli bir sayı belirt (1–10000)")
	}

	change := amount
	if remove {
		change = -amount
	}

	newNet, err := b.db.AddBonusInvites(c.guildID, targetID, change)
	if err != nil {
		return err
	}

	actionText := "eklendi"
	if remove {
		actionText = "çıkarıldı"
	}

	em := successEmbed("✅ Bonus Davet Güncellendi",
		fmt.Sprintf("<@%s> kullanıcısına **%d** bonus davet %s.\nGüncel net daveti: **%d**",
			targetID, amount, actionText, newNet))

	return c.embed(em)
}

func (b *Bot) inviteLeaderboardCommand(c *commandContext) error {
	top, err := b.db.TopInviters(c.guildID, 10)
	if err != nil {
		return err
	}

	if len(top) == 0 {
		em := embed("🏆 Davet Lider Tablosu", "Bu sunucuda henüz kaydedilmiş bir davet verisi bulunmuyor.", colorPrimary)
		return c.embed(em)
	}

	medals := []string{"🥇", "🥈", "🥉"}
	var lines []string

	for idx, s := range top {
		rankPrefix := fmt.Sprintf("`#%d`", idx+1)
		if idx < len(medals) {
			rankPrefix = medals[idx]
		}
		lines = append(lines, fmt.Sprintf("%s <@%s> — **%d** davet *(%d katılan, %d ayrılan, %d bonus)*",
			rankPrefix, s.UserID, s.Net(), s.Regular, s.Left, s.Bonus))
	}

	em := &discordgo.MessageEmbed{
		Title:       "🏆 Davet Lider Tablosu (Top 10)",
		Description: strings.Join(lines, "\n\n"),
		Color:       colorPrimary,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Adash Bot • Davet Sıralaması",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return c.embed(em)
}
