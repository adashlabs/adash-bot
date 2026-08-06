package bot

import (
	"fmt"
	mathrand "math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/adashlabs/adash-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) giveawayEmbed(g database.Giveaway, entries int, ended bool, winners []string) *discordgo.MessageEmbed {
	prize := safeText(trunc(strings.TrimSpace(g.Prize), 1000))
	em := &discordgo.MessageEmbed{
		Title:       str(ended, "🏆 Çekiliş Sonucu", "🎉 Çekiliş"),
		Description: "**" + prize + "**",
		Color:       strColor(ended, colorSuccess, colorPrimary),
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Çekiliş #%d", g.ID)},
	}
	if ended {
		winnerText := "Kazanan belirlenemedi"
		if len(winners) > 0 {
			mentions := make([]string, len(winners))
			for index, userID := range winners {
				mentions[index] = "<@" + userID + ">"
			}
			winnerText = strings.Join(mentions, ", ")
		}
		em.Fields = append(em.Fields,
			&discordgo.MessageEmbedField{Name: "Kazananlar", Value: winnerText},
			&discordgo.MessageEmbedField{Name: "Katılım", Value: fmt.Sprintf("%d kişi", entries), Inline: true},
			&discordgo.MessageEmbedField{Name: "Düzenleyen", Value: "<@" + g.HostID + ">", Inline: true},
		)
		return em
	}

	em.Fields = append(em.Fields,
		&discordgo.MessageEmbedField{Name: "Bitiş", Value: fmt.Sprintf("<t:%d:F>\n<t:%d:R>", g.EndsAt/1000, g.EndsAt/1000), Inline: true},
		&discordgo.MessageEmbedField{Name: "Kazanan", Value: fmt.Sprintf("%d kişi", g.WinnerCount), Inline: true},
		&discordgo.MessageEmbedField{Name: "Katılım", Value: fmt.Sprintf("%d kişi", entries), Inline: true},
	)
	var conditions []string
	if g.RequiredRoleID.Valid {
		conditions = append(conditions, "Rol: <@&"+g.RequiredRoleID.String+">")
	}
	if g.MinAccountAgeDays > 0 {
		conditions = append(conditions, fmt.Sprintf("Hesap yaşı: en az %d gün", g.MinAccountAgeDays))
	}
	if len(conditions) > 0 {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "Katılım koşulları", Value: strings.Join(conditions, "\n")})
	}
	em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "Düzenleyen", Value: "<@" + g.HostID + ">"})
	return em
}
func strColor(v bool, a, b int) int {
	if v {
		return a
	}
	return b
}
func giveawayButtons(ended bool) []discordgo.MessageComponent {
	x := button("giveaway_join", str(ended, "Çekiliş Bitti", "Katıl / Ayrıl"), discordgo.SuccessButton, "🎉")
	x.Disabled = ended
	return []discordgo.MessageComponent{row(x)}
}
func (b *Bot) giveawayCommand(c *commandContext, args []string) error {
	if e := c.require(discordgo.PermissionManageServer); e != nil {
		return e
	}
	if len(args) == 0 {
		return c.embed(b.setupEmbed(c.guildID, "giveaway"), setupComponents(c.guildID, "giveaway")...)
	}
	if len(args) < 3 {
		return fmt.Errorf("kullanım: giveaway <10m|2h|3d> <kazanan> <ödül>")
	}
	d, e := parseDuration(args[0])
	if e != nil || d < 10*time.Second {
		return fmt.Errorf("geçerli bir çekiliş süresi belirt")
	}
	n, e := strconv.Atoi(args[1])
	if e != nil || n < 1 || n > 20 {
		return fmt.Errorf("kazanan sayısı 1–20 olmalı")
	}
	return b.createGiveaway(c, d, n, strings.Join(args[2:], " "))
}
func (b *Bot) createGiveaway(c *commandContext, d time.Duration, winners int, prize string) error {
	prize = trunc(strings.TrimSpace(prize), 1000)
	if prize == "" {
		return fmt.Errorf("ödül boş olamaz")
	}
	role := b.db.ConfigString(c.guildID, "giveaway_required_role_id", "")
	minDays := b.db.ConfigInt(c.guildID, "giveaway_min_account_age_days", 0)
	draft := database.Giveaway{GuildID: c.guildID, ChannelID: c.channelID, HostID: c.user.ID, Prize: prize, WinnerCount: winners, EndsAt: time.Now().Add(d).UnixMilli(), MinAccountAgeDays: minDays}
	if role != "" {
		draft.RequiredRoleID.Valid = true
		draft.RequiredRoleID.String = role
	}
	msg, e := c.s.ChannelMessageSendComplex(c.channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{b.giveawayEmbed(draft, 0, false, nil)}, Components: giveawayButtons(false), AllowedMentions: &discordgo.MessageAllowedMentions{}})
	if e != nil {
		return e
	}
	id, e := b.db.CreateGiveaway(c.guildID, c.channelID, msg.ID, c.user.ID, prize, winners, role, minDays, draft.EndsAt)
	if e != nil {
		return e
	}
	draft.ID = id
	draft.MessageID = msg.ID
	_, _ = c.s.ChannelMessageEditComplex(&discordgo.MessageEdit{Channel: msg.ChannelID, ID: msg.ID, Embeds: &[]*discordgo.MessageEmbed{b.giveawayEmbed(draft, 0, false, nil)}, Components: &[]discordgo.MessageComponent{giveawayButtons(false)[0]}})
	b.scheduleGiveaway(draft)
	return c.text(fmt.Sprintf("🎉 Çekiliş #%d başlatıldı.", id))
}
func (b *Bot) scheduleGiveaway(g database.Giveaway) {
	delay := time.Until(time.UnixMilli(g.EndsAt))
	if delay < 0 {
		delay = 0
	}
	b.mu.Lock()
	if old := b.giveawayTimers[g.ID]; old != nil {
		old.Stop()
	}
	b.giveawayTimers[g.ID] = time.AfterFunc(delay, func() {
		if _, e := b.finishGiveaway(g); e != nil {
			fmt.Printf("çekiliş bitirme: %v\n", e)
		}
	})
	b.mu.Unlock()
}
func (b *Bot) restoreGiveaways() {
	xs, e := b.db.ActiveGiveaways()
	if e != nil {
		return
	}
	for _, g := range xs {
		if g.EndsAt <= time.Now().UnixMilli() {
			go b.finishGiveaway(g)
		} else {
			b.scheduleGiveaway(g)
		}
	}
}
func chooseWinners(entries []string, n int) []string {
	pool := append([]string(nil), entries...)
	mathrand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}
func (b *Bot) finishGiveaway(g database.Giveaway) ([]string, error) {
	ok, e := b.db.EndGiveaway(g.ID)
	if e != nil {
		return nil, e
	}
	if !ok {
		return nil, nil
	}
	entries, e := b.db.GiveawayEntries(g.ID)
	if e != nil {
		return nil, e
	}
	winners := chooseWinners(entries, g.WinnerCount)
	em := b.giveawayEmbed(g, len(entries), true, winners)
	_, _ = b.dg.ChannelMessageEditComplex(&discordgo.MessageEdit{Channel: g.ChannelID, ID: g.MessageID, Embeds: &[]*discordgo.MessageEmbed{em}, Components: &[]discordgo.MessageComponent{giveawayButtons(true)[0]}})
	text := "🎉 **" + g.Prize + "** çekilişi katılımcı olmadığı için sonuçlanamadı."
	if len(winners) > 0 {
		xs := make([]string, len(winners))
		for i, x := range winners {
			xs[i] = "<@" + x + ">"
		}
		text = "🎉 Tebrikler " + strings.Join(xs, ", ") + "! **" + g.Prize + "** ödülünü kazandınız."
	}
	text = safeText(text)
	_, _ = b.dg.ChannelMessageSendComplex(g.ChannelID, &discordgo.MessageSend{Content: text, AllowedMentions: &discordgo.MessageAllowedMentions{Users: winners}})
	b.giveawayLog(g, "🏁 Çekiliş sonuçlandı · "+text)
	b.mu.Lock()
	delete(b.giveawayTimers, g.ID)
	b.mu.Unlock()
	return winners, nil
}
func (b *Bot) toggleGiveaway(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	g, e := b.db.GiveawayByMessage(i.Message.ID)
	if e != nil || g.EndedAt.Valid || g.EndsAt <= time.Now().UnixMilli() {
		return b.followInteraction(s, i, "Bu çekiliş artık aktif değil.")
	}
	if g.RequiredRoleID.Valid {
		found := false
		for _, r := range i.Member.Roles {
			if r == g.RequiredRoleID.String {
				found = true
			}
		}
		if !found {
			return b.followInteraction(s, i, "Katılmak için <@&"+g.RequiredRoleID.String+"> rolüne sahip olmalısın.")
		}
	}
	age := int(time.Since(snowflakeTime(userOf(i).ID)).Hours() / 24)
	if age < g.MinAccountAgeDays {
		return b.followInteraction(s, i, fmt.Sprintf("Hesabın en az %d günlük olmalı. Mevcut: %d gün.", g.MinAccountAgeDays, age))
	}
	joined, e := b.db.JoinGiveaway(g.ID, userOf(i).ID)
	if e != nil {
		return e
	}
	if !joined {
		_ = b.db.LeaveGiveaway(g.ID, userOf(i).ID)
	}
	entries, e := b.db.GiveawayEntries(g.ID)
	if e != nil {
		return e
	}
	_, e = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{b.giveawayEmbed(g, len(entries), false, nil)}, Components: &[]discordgo.MessageComponent{giveawayButtons(false)[0]}})
	if e != nil {
		return e
	}
	text := "Çekilişten ayrıldın."
	if joined {
		text = fmt.Sprintf("🎉 Katılımın kaydedildi. Toplam **%d** katılımcı var.", len(entries))
	}
	return b.followInteraction(s, i, text)
}
func (b *Bot) followInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, text string) error {
	_, e := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: text, Flags: discordgo.MessageFlagsEphemeral})
	return e
}
func (b *Bot) giveawayManage(c *commandContext, args []string) error {
	if e := c.require(discordgo.PermissionManageServer); e != nil {
		return e
	}
	if len(args) < 2 {
		return fmt.Errorf("bitir/yeniden ve çekiliş ID belirt")
	}
	id, e := strconv.ParseInt(args[1], 10, 64)
	if e != nil {
		return fmt.Errorf("geçersiz çekiliş ID")
	}
	g, e := b.db.GiveawayByID(id)
	if e != nil {
		return fmt.Errorf("çekiliş bulunamadı")
	}
	if args[0] == "bitir" {
		if g.EndedAt.Valid {
			return fmt.Errorf("çekiliş zaten bitmiş")
		}
		w, e := b.finishGiveaway(g)
		if e != nil {
			return e
		}
		return c.text(fmt.Sprintf("Çekiliş bitirildi; %d kazanan.", len(w)))
	}
	if args[0] == "yeniden" || args[0] == "reroll" {
		if !g.EndedAt.Valid {
			return fmt.Errorf("önce çekilişi bitir")
		}
		n := 1
		if len(args) > 2 {
			n, _ = strconv.Atoi(args[2])
			if n < 1 {
				n = 1
			}
		}
		entries, e := b.db.GiveawayEntries(id)
		if e != nil {
			return e
		}
		wins := chooseWinners(entries, n)
		xs := make([]string, len(wins))
		for i, x := range wins {
			xs[i] = "<@" + x + ">"
		}
		text := "Yeniden çekilecek katılımcı yok."
		if len(xs) > 0 {
			text = "🔁 Yeni kazananlar: " + strings.Join(xs, ", ")
		}
		text = safeText(text)
		_, _ = b.dg.ChannelMessageSendComplex(g.ChannelID, &discordgo.MessageSend{Content: text, AllowedMentions: &discordgo.MessageAllowedMentions{Users: wins}})
		b.giveawayLog(g, text)
		return c.text(text)
	}
	return fmt.Errorf("işlem bitir veya yeniden olmalı")
}
func (b *Bot) giveawayLog(g database.Giveaway, text string) {
	channel := b.db.ConfigString(g.GuildID, "giveaway_log_channel_id", "")
	if channel != "" {
		_, _ = b.dg.ChannelMessageSendComplex(channel, &discordgo.MessageSend{Content: text, AllowedMentions: &discordgo.MessageAllowedMentions{}})
	}
}
