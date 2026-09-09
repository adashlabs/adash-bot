package bot

import (
	"fmt"
	"math"
	mathrand "math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/adashlabs/adash-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

func giveawayChance(entries, winners int) string {
	if entries < 1 || winners < 1 {
		return "%0.0"
	}
	chance := float64(winners) / float64(entries) * 100
	if chance > 100 {
		chance = 100
	}
	return fmt.Sprintf("%%%0.1f", chance)
}

func giveawayChanceDetail(entries, winners int) string {
	if entries < 1 || winners < 1 {
		return "%0.0 (1 / 0)"
	}
	chance := math.Min(100, (float64(winners)/float64(entries))*100)
	effectiveWinners := winners
	if effectiveWinners > entries {
		effectiveWinners = entries
	}
	ratio := int(math.Ceil(float64(entries) / float64(effectiveWinners)))
	return fmt.Sprintf("%%%0.1f (Yaklaşık 1 / %d)", chance, ratio)
}

func (b *Bot) giveawayEmbed(g database.Giveaway, entries int, ended bool, winners []string) *discordgo.MessageEmbed {
	prize := safeText(trunc(strings.TrimSpace(g.Prize), 1000))
	em := &discordgo.MessageEmbed{
		Title:       str(ended, "🏆 Çekiliş Sonucu", "🎉 Çekiliş"),
		Description: fmt.Sprintf("🎁 **Ödül:** **%s**", prize),
		Color:       strColor(ended, strColor(len(winners) > 0, colorSuccess, colorDanger), colorPrimary),
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Çekiliş #%d • Adash Çekiliş Sistemi", g.ID)},
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if ended {
		winnerText := "Kazanan belirlenemedi (yeterli katılım yok)"
		if len(winners) > 0 {
			mentions := make([]string, len(winners))
			for index, userID := range winners {
				mentions[index] = "<@" + userID + ">"
			}
			winnerText = strings.Join(mentions, ", ")
		}
		em.Fields = append(em.Fields,
			&discordgo.MessageEmbedField{Name: "🏆 Kazananlar", Value: winnerText, Inline: false},
			&discordgo.MessageEmbedField{Name: "👥 Toplam Katılım", Value: fmt.Sprintf("**%d** kişi", entries), Inline: true},
			&discordgo.MessageEmbedField{Name: "👑 Düzenleyen", Value: "<@" + g.HostID + ">", Inline: true},
			&discordgo.MessageEmbedField{Name: "🏁 Bitiş Tarihi", Value: fmt.Sprintf("<t:%d:F>", g.EndsAt/1000), Inline: false},
		)
		return em
	}

	em.Fields = append(em.Fields,
		&discordgo.MessageEmbedField{Name: "⏰ Bitiş Zamanı", Value: fmt.Sprintf("<t:%d:R>\n<t:%d:F>", g.EndsAt/1000, g.EndsAt/1000), Inline: true},
		&discordgo.MessageEmbedField{Name: "🏆 Kazanan Sayısı", Value: fmt.Sprintf("**%d** kişi", g.WinnerCount), Inline: true},
		&discordgo.MessageEmbedField{Name: "👥 Katılımcı", Value: fmt.Sprintf("**%d** kişi", entries), Inline: true},
	)

	var conditions []string
	if g.RequiredRoleID.Valid && g.RequiredRoleID.String != "" {
		conditions = append(conditions, "• Gerekli Rol: <@&"+g.RequiredRoleID.String+">")
	}
	if g.MinAccountAgeDays > 0 {
		conditions = append(conditions, fmt.Sprintf("• Minimum Hesap Yaşı: **%d** gün", g.MinAccountAgeDays))
	}
	if len(conditions) > 0 {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "🛡️ Katılım Koşulları", Value: strings.Join(conditions, "\n"), Inline: false})
	} else {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "🛡️ Katılım Koşulları", Value: "• Herkes katılabilir", Inline: false})
	}

	em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "👑 Düzenleyen", Value: "<@" + g.HostID + ">", Inline: true})
	return em
}

func strColor(v bool, a, b int) int {
	if v {
		return a
	}
	return b
}

func giveawayButtons(g database.Giveaway, entries int, ended bool) []discordgo.MessageComponent {
	if ended {
		btnEnded := discordgo.Button{
			CustomID: "giveaway_ended",
			Label:    "Çekiliş Sona Erdi",
			Style:    discordgo.SecondaryButton,
			Disabled: true,
			Emoji:    &discordgo.ComponentEmoji{Name: "🏆"},
		}
		btnReroll := discordgo.Button{
			CustomID: fmt.Sprintf("giveaway_reroll:%d", g.ID),
			Label:    "Yeniden Çek (Reroll)",
			Style:    discordgo.PrimaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "🔁"},
		}
		btnParticipants := discordgo.Button{
			CustomID: fmt.Sprintf("giveaway_participants:%d", g.ID),
			Label:    fmt.Sprintf("Katılımcılar (%d)", entries),
			Style:    discordgo.SecondaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "👥"},
		}
		return []discordgo.MessageComponent{row(btnEnded, btnReroll, btnParticipants)}
	}

	joinLabel := "Katıl"
	if entries > 0 {
		joinLabel = fmt.Sprintf("Katıl (%d)", entries)
	}
	btnJoin := discordgo.Button{
		CustomID: "giveaway_join",
		Label:    joinLabel,
		Style:    discordgo.SuccessButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "🎉"},
	}
	btnChance := discordgo.Button{
		CustomID: fmt.Sprintf("giveaway_mychance:%d", g.ID),
		Label:    "Şansım Ne?",
		Style:    discordgo.SecondaryButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "🎲"},
	}
	btnParticipants := discordgo.Button{
		CustomID: fmt.Sprintf("giveaway_participants:%d", g.ID),
		Label:    "Katılımcılar",
		Style:    discordgo.SecondaryButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "👥"},
	}
	return []discordgo.MessageComponent{row(btnJoin, btnChance, btnParticipants)}
}

func (b *Bot) giveawayCommand(c *commandContext, args []string) error {
	if e := c.require(discordgo.PermissionManageServer); e != nil {
		return e
	}
	if len(args) == 0 {
		return b.giveawayWizard(c)
	}
	if len(args) < 3 {
		return fmt.Errorf("kullanım: giveaway <süre> <kazanan> <ödül>\nÖrnek: a!çekiliş 1h 1 Discord Nitro")
	}
	d, e := parseDuration(args[0])
	if e != nil || d < 10*time.Second {
		return fmt.Errorf("geçerli bir çekiliş süresi belirt (örn: 10m, 2h, 3d)")
	}
	n, e := strconv.Atoi(args[1])
	if e != nil || n < 1 || n > 20 {
		return fmt.Errorf("kazanan sayısı 1–20 olmalı")
	}
	return b.createGiveaway(c, d, n, strings.Join(args[2:], " "))
}

func (b *Bot) giveawayWizard(c *commandContext) error {
	roleStr := "Rol şartı yok (Herkes katılabilir)"
	roleID := b.db.ConfigString(c.guildID, "giveaway_required_role_id", "")
	if roleID != "" {
		roleStr = "<@&" + roleID + ">"
	}
	minDays := b.db.ConfigInt(c.guildID, "giveaway_min_account_age_days", 0)
	ageStr := "Hesap yaşı şartı yok"
	if minDays > 0 {
		ageStr = fmt.Sprintf("En az **%d** günlük hesap", minDays)
	}

	em := &discordgo.MessageEmbed{
		Title: "🎉 Çekiliş Yönetim ve Başlatma Paneli",
		Description: "Aşağıdaki butonları kullanarak hızlıca modal form ile çekiliş başlatabilir veya doğrudan komut kullanabilirsiniz.\n\n" +
			"📌 **Hızlı Komut Kullanımı:**\n" +
			"`" + b.db.Prefix(c.guildID) + "çekiliş <süre> <kazanan> <ödül>`\n" +
			"Örnek: `" + b.db.Prefix(c.guildID) + "çekiliş 1h 1 Discord Nitro`\n\n" +
			"⚙️ **Mevcut Katılım Şartları:**\n" +
			"• **Gerekli Rol:** " + roleStr + "\n" +
			"• **Minimum Hesap Yaşı:** " + ageStr + "\n\n" +
			"👉 Aşağıdaki **Çekiliş Başlat** butonuna basarak formu açabilirsiniz.",
		Color: colorPrimary,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Adash Bot • Çekiliş Sistemi",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	btnStart := button("setup_giveaway_create_btn:"+c.guildID, "Çekiliş Başlat (Form)", discordgo.SuccessButton, "🎉")
	btnRules := button("setup_giveaway_rules:"+c.guildID, "Şartları Düzenle", discordgo.PrimaryButton, "⚙️")

	return c.embed(em, row(btnStart, btnRules))
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
	msg, e := c.s.ChannelMessageSendComplex(c.channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{b.giveawayEmbed(draft, 0, false, nil)}, Components: giveawayButtons(draft, 0, false), AllowedMentions: &discordgo.MessageAllowedMentions{}})
	if e != nil {
		return e
	}
	id, e := b.db.CreateGiveaway(c.guildID, c.channelID, msg.ID, c.user.ID, prize, winners, role, minDays, draft.EndsAt)
	if e != nil {
		return e
	}
	draft.ID = id
	draft.MessageID = msg.ID
	_, _ = c.s.ChannelMessageEditComplex(&discordgo.MessageEdit{Channel: msg.ChannelID, ID: msg.ID, Embeds: &[]*discordgo.MessageEmbed{b.giveawayEmbed(draft, 0, false, nil)}, Components: &[]discordgo.MessageComponent{giveawayButtons(draft, 0, false)[0]}})
	b.scheduleGiveaway(draft)
	return c.text(fmt.Sprintf("🎉 Çekiliş #%d başarıyla başlatıldı!", id))
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
	_, _ = b.dg.ChannelMessageEditComplex(&discordgo.MessageEdit{Channel: g.ChannelID, ID: g.MessageID, Embeds: &[]*discordgo.MessageEmbed{em}, Components: &[]discordgo.MessageComponent{giveawayButtons(g, len(entries), true)[0]}})
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

	// Kazananlara DM gönder
	for _, winnerID := range winners {
		go b.sendGiveawayWinDM(g, winnerID)
	}

	b.mu.Lock()
	delete(b.giveawayTimers, g.ID)
	b.mu.Unlock()
	return winners, nil
}

func (b *Bot) sendGiveawayWinDM(g database.Giveaway, userID string) {
	dmChan, err := b.dg.UserChannelCreate(userID)
	if err != nil {
		return
	}
	guild, _ := b.dg.Guild(g.GuildID)
	guildName := "Sunucu"
	if guild != nil {
		guildName = guild.Name
	}
	dmEmbed := embed("🎉 Tebrikler, Çekilişi Kazandınız!",
		fmt.Sprintf("Harika haber! **%s** sunucusunda düzenlenen **%s** çekilişini kazandınız!\n\n"+
			"🎁 **Ödülünüz:** **%s**\n"+
			"👑 **Düzenleyen:** <@%s>\n\n"+
			"Ödülünüzü teslim almak için lütfen çekilişi düzenleyen yetkili ile iletişime geçin.",
			guildName, g.Prize, g.Prize, g.HostID),
		colorSuccess)
	_, _ = b.dg.ChannelMessageSendEmbed(dmChan.ID, dmEmbed)
}

func (b *Bot) toggleGiveaway(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	g, e := b.db.GiveawayByMessage(i.Message.ID)
	if e != nil || g.EndedAt.Valid || g.EndsAt <= time.Now().UnixMilli() {
		return b.followInteraction(s, i, "Bu çekiliş artık aktif değil.")
	}
	if g.RequiredRoleID.Valid && g.RequiredRoleID.String != "" {
		found := false
		if i.Member != nil {
			for _, r := range i.Member.Roles {
				if r == g.RequiredRoleID.String {
					found = true
					break
				}
			}
		}
		if !found {
			return b.followInteraction(s, i, "❌ Katılmak için <@&"+g.RequiredRoleID.String+"> rolüne sahip olmalısın.")
		}
	}
	age := int(time.Since(snowflakeTime(userOf(i).ID)).Hours() / 24)
	if age < g.MinAccountAgeDays {
		return b.followInteraction(s, i, fmt.Sprintf("❌ Hesabın en az %d günlük olmalı. Mevcut hesap yaşın: %d gün.", g.MinAccountAgeDays, age))
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
	_, e = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{b.giveawayEmbed(g, len(entries), false, nil)}, Components: &[]discordgo.MessageComponent{giveawayButtons(g, len(entries), false)[0]}})
	if e != nil {
		return e
	}
	text := "🚪 Çekilişten ayrıldınız. Çekiliş bitmeden önce dilediğiniz zaman tekrar katılabilirsiniz."
	if joined {
		text = fmt.Sprintf("🎉 **Çekilişe katılımınız kaydedildi!**\n\n"+
			"🎁 **Ödül:** **%s**\n"+
			"👥 **Toplam Katılımcı:** **%d** kişi\n"+
			"🎲 **Tahmini Kazanma Şansınız:** **%s**\n"+
			"⏰ **Sonuç Tarihi:** <t:%d:R>",
			g.Prize, len(entries), giveawayChanceDetail(len(entries), g.WinnerCount), g.EndsAt/1000)
	}
	return b.followInteraction(s, i, text)
}

func (b *Bot) handleGiveawayMyChance(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) < 2 {
		return fmt.Errorf("geçersiz çekiliş ID")
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("geçersiz çekiliş ID")
	}

	g, err := b.db.GiveawayByID(id)
	if err != nil {
		return ephemeral(s, i, "⚠️ Çekiliş bulunamadı veya silinmiş.")
	}

	entries, err := b.db.GiveawayEntries(g.ID)
	if err != nil {
		return err
	}

	user := userOf(i)
	isEntered := false
	for _, uid := range entries {
		if uid == user.ID {
			isEntered = true
			break
		}
	}

	if !isEntered {
		var missingReqs []string
		if g.RequiredRoleID.Valid && g.RequiredRoleID.String != "" {
			hasRole := false
			if i.Member != nil {
				for _, r := range i.Member.Roles {
					if r == g.RequiredRoleID.String {
						hasRole = true
						break
					}
				}
			}
			if !hasRole {
				missingReqs = append(missingReqs, "• <@&"+g.RequiredRoleID.String+"> rolüne sahip olmalısınız.")
			}
		}
		age := int(time.Since(snowflakeTime(user.ID)).Hours() / 24)
		if age < g.MinAccountAgeDays {
			missingReqs = append(missingReqs, fmt.Sprintf("• Hesabınız en az %d günlük olmalı (Mevcut: %d gün).", g.MinAccountAgeDays, age))
		}

		if len(missingReqs) > 0 {
			msg := fmt.Sprintf("⚠️ **Bu çekilişe şu an için katılamazsınız:**\n%s", strings.Join(missingReqs, "\n"))
			return ephemeral(s, i, msg)
		}

		msg := fmt.Sprintf("ℹ️ **Bu çekilişe henüz katılmadınız!**\n\n"+
			"🎁 **Ödül:** **%s**\n"+
			"Katılmak için aşağıdaki **🎉 Katıl** butonuna tıklayabilirsiniz.\n"+
			"Şu an katılırsanız kazanma şansınız yaklaşık **%s** olacaktır.",
			g.Prize, giveawayChanceDetail(len(entries)+1, g.WinnerCount))
		return ephemeral(s, i, msg)
	}

	statusDesc := fmt.Sprintf("🎯 **Çekiliş Katılım Bilgileriniz**\n\n"+
		"✅ **Durum:** Çekilişe katıldınız!\n"+
		"🎁 **Ödül:** **%s**\n"+
		"🏆 **Kazanan Sayısı:** **%d** kişi\n"+
		"👥 **Toplam Katılımcı:** **%d** kişi\n"+
		"🎲 **Tahmini Kazanma Şansınız:** **%s**\n"+
		"⏰ **Sonuç:** <t:%d:R>",
		g.Prize, g.WinnerCount, len(entries), giveawayChanceDetail(len(entries), g.WinnerCount), g.EndsAt/1000)

	return ephemeral(s, i, statusDesc)
}

func (b *Bot) handleGiveawayParticipants(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) < 2 {
		return fmt.Errorf("geçersiz çekiliş ID")
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("geçersiz çekiliş ID")
	}

	g, err := b.db.GiveawayByID(id)
	if err != nil {
		return ephemeral(s, i, "⚠️ Çekiliş bulunamadı veya silinmiş.")
	}

	entries, err := b.db.GiveawayEntries(g.ID)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return ephemeral(s, i, "ℹ️ Bu çekilişte henüz hiç katılımcı bulunmuyor. İlk katılan sen ol!")
	}

	maxShow := 30
	showCount := len(entries)
	if showCount > maxShow {
		showCount = maxShow
	}

	mentions := make([]string, showCount)
	for idx := 0; idx < showCount; idx++ {
		mentions[idx] = "<@" + entries[idx] + ">"
	}

	desc := strings.Join(mentions, ", ")
	if len(entries) > maxShow {
		desc += fmt.Sprintf("\n\n*... ve %d katılımcı daha.*", len(entries)-maxShow)
	}

	em := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("👥 Çekiliş Katılımcıları (Toplam %d Kişi)", len(entries)),
		Description: desc,
		Color:       colorPrimary,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Çekiliş ID: #%d • Ödül: %s", g.ID, trunc(g.Prize, 40)),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{em},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleGiveawayRerollButton(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	p, e := s.UserChannelPermissions(userOf(i).ID, i.ChannelID)
	if e != nil || (p&discordgo.PermissionManageServer == 0 && p&discordgo.PermissionAdministrator == 0) {
		return ephemeral(s, i, "❌ Çekilişi yeniden çekmek için Sunucuyu Yönet (Manage Server) yetkisine sahip olmalısın.")
	}

	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) < 2 {
		return fmt.Errorf("geçersiz çekiliş ID")
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("geçersiz çekiliş ID")
	}

	g, err := b.db.GiveawayByID(id)
	if err != nil {
		return ephemeral(s, i, "⚠️ Çekiliş bulunamadı.")
	}

	if !g.EndedAt.Valid {
		return ephemeral(s, i, "⚠️ Bu çekiliş henüz sona ermemiş.")
	}

	entries, err := b.db.GiveawayEntries(g.ID)
	if err != nil || len(entries) == 0 {
		return ephemeral(s, i, "⚠️ Yeniden çekilecek katılımcı bulunamadı.")
	}

	wins := chooseWinners(entries, g.WinnerCount)
	if len(wins) == 0 {
		return ephemeral(s, i, "⚠️ Kazanan belirlenemedi.")
	}

	xs := make([]string, len(wins))
	for idx, x := range wins {
		xs[idx] = "<@" + x + ">"
	}

	text := "🔁 **ÇEKİLİŞ YENİDEN ÇEKİLDİ!**\n🎉 Tebrikler " + strings.Join(xs, ", ") + "! **" + g.Prize + "** çekilişinin yeni kazananı oldunuz."
	text = safeText(text)

	// Orijinal mesajı güncelle
	updatedEmbed := b.giveawayEmbed(g, len(entries), true, wins)
	_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    g.ChannelID,
		ID:         g.MessageID,
		Embeds:     &[]*discordgo.MessageEmbed{updatedEmbed},
		Components: &[]discordgo.MessageComponent{giveawayButtons(g, len(entries), true)[0]},
	})

	// Kanala duyur
	_, _ = b.dg.ChannelMessageSendComplex(g.ChannelID, &discordgo.MessageSend{
		Content:         text,
		AllowedMentions: &discordgo.MessageAllowedMentions{Users: wins},
	})

	// Log
	b.giveawayLog(g, "🔁 Çekiliş yeniden çekildi (#"+strconv.FormatInt(g.ID, 10)+") · "+text)

	// Yeni kazananlara DM
	for _, winnerID := range wins {
		go b.sendGiveawayWinDM(g, winnerID)
	}

	return ephemeral(s, i, "✅ Çekiliş başarıyla yeniden çekildi ve yeni kazananlar kanala duyuruldu!")
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
		return c.text(fmt.Sprintf("🎉 Çekiliş bitirildi; %d kazanan belirlendi.", len(w)))
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
