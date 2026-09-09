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
		Color:       strColor(ended, strColor(len(winners) > 0, colorSuccess, colorDanger), colorPrimary),
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Çekiliş #%d • Adash Çekiliş Sistemi", g.ID)},
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if ended {
		em.Description = fmt.Sprintf("🎁 **Ödül:** **%s**\n⏰ **Bitiş:** <t:%d:F> *(<t:%d:R>)*", prize, g.EndsAt/1000, g.EndsAt/1000)
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
			&discordgo.MessageEmbedField{Name: "🏁 Bitiş Tarihi", Value: fmt.Sprintf("<t:%d:F>\n*(<t:%d:R>)*", g.EndsAt/1000, g.EndsAt/1000), Inline: false},
		)
		return em
	}

	em.Description = fmt.Sprintf("🎁 **Ödül:** **%s**\n⏰ **Kalan Süre:** <t:%d:R> *(<t:%d:F>)*", prize, g.EndsAt/1000, g.EndsAt/1000)

	em.Fields = append(em.Fields,
		&discordgo.MessageEmbedField{Name: "⏰ Bitiş Zamanı", Value: fmt.Sprintf("<t:%d:R>\n<t:%d:F>", g.EndsAt/1000, g.EndsAt/1000), Inline: true},
		&discordgo.MessageEmbedField{Name: "🏆 Kazanan Sayısı", Value: fmt.Sprintf("**%d** kişi", g.WinnerCount), Inline: true},
		&discordgo.MessageEmbedField{Name: "👥 Katılımcı", Value: fmt.Sprintf("**%d** kişi", entries), Inline: true},
	)

	var conditions []string
	if g.MinInvites > 0 {
		conditions = append(conditions, fmt.Sprintf("• 📨 **Davet Şartı:** En az **%d** geçerli davet yapmış olmak", g.MinInvites))
	}
	if g.MinAccountAgeDays > 0 {
		conditions = append(conditions, fmt.Sprintf("• ⏳ **Hesap Yaşı:** En az **%d** günlük Discord hesabı", g.MinAccountAgeDays))
	}
	if g.RequiredRoleID.Valid && g.RequiredRoleID.String != "" {
		conditions = append(conditions, "• 🏷️ **Zorunlu Rol:** <@&"+g.RequiredRoleID.String+"> rolüne sahip olmak")
	}
	if len(conditions) > 0 {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "🛡️ Katılım Koşulları", Value: strings.Join(conditions, "\n"), Inline: false})
	} else {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "🛡️ Katılım Koşulları", Value: "• 🔓 **Özel şart yok:** Tüm sunucu üyeleri katılabilir!", Inline: false})
	}

	if b != nil && b.db != nil && g.GuildID != "" {
		multRole := b.db.ConfigString(g.GuildID, "giveaway_multiplier_role_id", "")
		multVal := b.db.ConfigInt(g.GuildID, "giveaway_multiplier_value", 1)
		if multRole != "" && multVal > 1 {
			em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
				Name:   "🚀 Ekstra Şans & Çarpan Avantajı",
				Value:  fmt.Sprintf("• <@&%s> rolüne sahip üyeler çekilişte **%dx** daha fazla şans kazanır! *(%d kat bilet)*", multRole, multVal, multVal),
				Inline: false,
			})
		}
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

	if args[0] == "çarpan" || args[0] == "carpan" {
		if len(args) == 1 {
			roleID := b.db.ConfigString(c.guildID, "giveaway_multiplier_role_id", "")
			mult := b.db.ConfigInt(c.guildID, "giveaway_multiplier_value", 1)
			roleStr := "Rol ayarlı değil"
			if roleID != "" {
				roleStr = "<@&" + roleID + ">"
			}
			desc := fmt.Sprintf("🚀 **Çekiliş Çarpan / Booster Ayarları**\n\n"+
				"• **Rol:** %s\n"+
				"• **Kazanma Şansı Çarpanı:** **%dx**\n\n"+
				"📌 **Kullanım:**\n"+
				"`%sçekiliş çarpan @rol <çarpan>` (Örn: `%sçekiliş çarpan @Booster 2`)\n"+
				"`%sçekiliş çarpan sıfırla` (Çarpanı kaldırır)",
				roleStr, mult, b.db.Prefix(c.guildID), b.db.Prefix(c.guildID), b.db.Prefix(c.guildID))
			return c.embed(embed("🚀 Çekiliş Çarpan Durumu", desc, colorPrimary))
		}
		if args[1] == "sıfırla" || args[1] == "sifirla" || args[1] == "kaldır" || args[1] == "kaldir" {
			_ = b.db.SetConfig(c.guildID, "giveaway_multiplier_role_id", "")
			_ = b.db.SetConfig(c.guildID, "giveaway_multiplier_value", 1)
			return c.embed(successEmbed("✅ Çekiliş Çarpanı Sıfırlandı", "Rol çarpanı kaldırıldı. Artık tüm katılımcılar eşit kazanma şansına sahip."))
		}
		if len(args) < 3 {
			return fmt.Errorf("kullanım: `%sçekiliş çarpan @rol <sayı>` veya `%sçekiliş çarpan sıfırla`", b.db.Prefix(c.guildID), b.db.Prefix(c.guildID))
		}
		roleID := mentionID(args[1])
		if roleID == "" {
			return fmt.Errorf("lütfen geçerli bir rol etiketle veya rol ID'si gir")
		}
		mult, err := strconv.Atoi(args[2])
		if err != nil || mult < 2 || mult > 100 {
			return fmt.Errorf("çarpan 2–100 arasında bir tam sayı olmalı (örn: 2)")
		}
		_ = b.db.SetConfig(c.guildID, "giveaway_multiplier_role_id", roleID)
		_ = b.db.SetConfig(c.guildID, "giveaway_multiplier_value", mult)
		return c.embed(successEmbed("🚀 Çekiliş Çarpanı Ayarlandı",
			fmt.Sprintf("<@&%s> rolüne sahip üyeler çekilişlerde **%dx** kat kazanma şansı ve katılım hakkı elde edecek!", roleID, mult)))
	}

	if args[0] == "şart" || args[0] == "sart" {
		if len(args) == 1 {
			roleID := b.db.ConfigString(c.guildID, "giveaway_required_role_id", "")
			minDays := b.db.ConfigInt(c.guildID, "giveaway_min_account_age_days", 0)
			minInvites := b.db.ConfigInt(c.guildID, "giveaway_min_invites", 0)
			roleStr := "Yok"
			if roleID != "" {
				roleStr = "<@&" + roleID + ">"
			}
			desc := fmt.Sprintf("🛡️ **Çekiliş Katılım Şartları**\n\n"+
				"• **Zorunlu Rol:** %s\n"+
				"• **Minimum Hesap Yaşı:** %d gün\n"+
				"• **Minimum Geçerli Davet:** %d davet\n\n"+
				"📌 **Ayarlama Komutları:**\n"+
				"`%sçekiliş şart rol @rol`\n"+
				"`%sçekiliş şart yaş <gün>`\n"+
				"`%sçekiliş şart davet <sayı>`\n"+
				"`%sçekiliş şart sıfırla`",
				roleStr, minDays, minInvites, b.db.Prefix(c.guildID), b.db.Prefix(c.guildID), b.db.Prefix(c.guildID), b.db.Prefix(c.guildID))
			return c.embed(embed("🛡️ Çekiliş Şartları", desc, colorPrimary))
		}
		sub := strings.ToLower(args[1])
		switch sub {
		case "sıfırla", "sifirla":
			_ = b.db.SetConfig(c.guildID, "giveaway_required_role_id", "")
			_ = b.db.SetConfig(c.guildID, "giveaway_min_account_age_days", 0)
			_ = b.db.SetConfig(c.guildID, "giveaway_min_invites", 0)
			return c.embed(successEmbed("✅ Şartlar Sıfırlandı", "Tüm çekiliş katılım şartları kaldırıldı."))
		case "davet":
			if len(args) < 3 {
				return fmt.Errorf("kullanım: `%sçekiliş şart davet <sayı>`", b.db.Prefix(c.guildID))
			}
			inv, err := strconv.Atoi(args[2])
			if err != nil || inv < 0 || inv > 1000 {
				return fmt.Errorf("geçerli bir davet sayısı belirt (0–1000)")
			}
			_ = b.db.SetConfig(c.guildID, "giveaway_min_invites", inv)
			return c.embed(successEmbed("✅ Davet Şartı Ayarlandı", fmt.Sprintf("Çekilişler için minimum davet şartı **%d** olarak ayarlandı.", inv)))
		case "rol":
			if len(args) < 3 {
				return fmt.Errorf("kullanım: `%sçekiliş şart rol @rol`", b.db.Prefix(c.guildID))
			}
			roleID := mentionID(args[2])
			if roleID == "" {
				return fmt.Errorf("geçerli bir rol etiketle")
			}
			_ = b.db.SetConfig(c.guildID, "giveaway_required_role_id", roleID)
			return c.embed(successEmbed("✅ Rol Şartı Ayarlandı", fmt.Sprintf("Çekilişler için zorunlu rol <@&%s> olarak ayarlandı.", roleID)))
		case "yaş", "yas":
			if len(args) < 3 {
				return fmt.Errorf("kullanım: `%sçekiliş şart yaş <gün>`", b.db.Prefix(c.guildID))
			}
			days, err := strconv.Atoi(args[2])
			if err != nil || days < 0 || days > 365 {
				return fmt.Errorf("geçerli bir gün sayısı belirt (0–365)")
			}
			_ = b.db.SetConfig(c.guildID, "giveaway_min_account_age_days", days)
			return c.embed(successEmbed("✅ Hesap Yaşı Şartı Ayarlandı", fmt.Sprintf("Çekilişler için minimum hesap yaşı **%d** gün olarak ayarlandı.", days)))
		default:
			return fmt.Errorf("geçersiz şart türü. Kullanım: rol, yaş veya davet")
		}
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

func (b *Bot) giveawayWizardData(guildID string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	roleID := b.db.ConfigString(guildID, "giveaway_required_role_id", "")
	roleStr := "Zorunlu rol yok (Herkes)"
	if roleID != "" {
		roleStr = "<@&" + roleID + ">"
	}
	minDays := b.db.ConfigInt(guildID, "giveaway_min_account_age_days", 0)
	ageStr := "Hesap yaşı şartı yok"
	if minDays > 0 {
		ageStr = fmt.Sprintf("En az **%d** günlük hesap", minDays)
	}
	minInvites := b.db.ConfigInt(guildID, "giveaway_min_invites", 0)
	invitesStr := "Davet şartı yok"
	if minInvites > 0 {
		invitesStr = fmt.Sprintf("En az **%d** geçerli davet", minInvites)
	}
	multRole := b.db.ConfigString(guildID, "giveaway_multiplier_role_id", "")
	multVal := b.db.ConfigInt(guildID, "giveaway_multiplier_value", 1)
	multStr := "Çarpan aktif değil"
	if multRole != "" && multVal > 1 {
		multStr = fmt.Sprintf("<@&%s> (**%dx** şans & bilet)", multRole, multVal)
	}

	em := &discordgo.MessageEmbed{
		Title: "🎉 Çekiliş Yönetim ve Yapılandırma Paneli",
		Description: "Bu panel üzerinden tek tıkla çekiliş başlatabilir, katılım şartlarını (davet, hesap yaşı, zorunlu rol) ve **Booster / Çarpan** rollerini kolayca ayarlayabilirsiniz.\n\n" +
			"⚙️ **Mevcut Çekiliş Ayarları:**\n" +
			"• 🚀 **Booster / Çarpan Rolü:** " + multStr + "\n" +
			"• 📨 **Minimum Davet Şartı:** " + invitesStr + "\n" +
			"• ⏳ **Minimum Hesap Yaşı:** " + ageStr + "\n" +
			"• 🛡️ **Zorunlu Katılım Rolü:** " + roleStr + "\n\n" +
			"👇 Aşağıdaki menülerden rol seçebilir veya butonlarla formu açabilirsiniz.",
		Color: colorPrimary,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "📌 Hızlı Komut Kullanımı",
				Value: "`" + b.db.Prefix(guildID) + "çekiliş <süre/unix> <kazanan> <ödül>`\n" +
					"Örnekler:\n" +
					"• `" + b.db.Prefix(guildID) + "çekiliş 1h 1 Discord Nitro`\n" +
					"• `" + b.db.Prefix(guildID) + "çekiliş 2d12h 2 Steam Key --davet=1`\n" +
					"• `" + b.db.Prefix(guildID) + "çekiliş <t:1725900000:R> 1 Nitro Classic`\n" +
					"Çarpan: `" + b.db.Prefix(guildID) + "çekiliş çarpan @Booster 2`",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Adash Bot • Çekiliş ve Ödül Sistemi",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	btnStart := button("setup_giveaway_create_btn:"+guildID, "Çekiliş Başlat (Form)", discordgo.SuccessButton, "🎉")
	btnRules := button("setup_giveaway_rules:"+guildID, "Şartları Düzenle", discordgo.PrimaryButton, "⚙️")
	btnMult := button("setup_giveaway_mult:"+guildID, "Çarpan Katsayısı", discordgo.PrimaryButton, "🚀")
	btnClearMult := button("setup_clear:giveawaymult:"+guildID, "Çarpanı Kaldır", discordgo.SecondaryButton, "🔄")
	btnClearAll := button("setup_clear:giveawayall:"+guildID, "Tümünü Sıfırla", discordgo.DangerButton, "🧹")

	rowButtons := row(btnStart, btnRules, btnMult, btnClearMult, btnClearAll)
	rowMultRole := row(discordgo.SelectMenu{
		CustomID:    "setup_role:giveawaymultrole:" + guildID,
		Placeholder: "🚀 Booster / Çarpan Rolü Seç (Örn: Server Booster)",
		MenuType:    discordgo.RoleSelectMenu,
		MinValues:   intp(1),
		MaxValues:   1,
	})
	rowReqRole := row(discordgo.SelectMenu{
		CustomID:    "setup_role:giveawayrole:" + guildID,
		Placeholder: "🛡️ Zorunlu Katılım Rolü Seç (İsteğe bağlı)",
		MenuType:    discordgo.RoleSelectMenu,
		MinValues:   intp(1),
		MaxValues:   1,
	})

	return em, []discordgo.MessageComponent{rowButtons, rowMultRole, rowReqRole}
}

func (b *Bot) giveawayWizard(c *commandContext) error {
	em, comp := b.giveawayWizardData(c.guildID)
	return c.embed(em, comp...)
}

func (b *Bot) refreshGiveawayWizardIfApplicable(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if s == nil || i == nil || i.Message == nil || len(i.Message.Embeds) == 0 {
		return
	}
	if !strings.Contains(i.Message.Embeds[0].Title, "Çekiliş") {
		return
	}
	em, comp := b.giveawayWizardData(i.GuildID)
	_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    i.Message.ChannelID,
		ID:         i.Message.ID,
		Embeds:     &[]*discordgo.MessageEmbed{em},
		Components: &comp,
	})
}

func (b *Bot) createGiveaway(c *commandContext, d time.Duration, winners int, prize string) error {
	prize = trunc(strings.TrimSpace(prize), 1000)
	if prize == "" {
		return fmt.Errorf("ödül boş olamaz")
	}

	role := b.db.ConfigString(c.guildID, "giveaway_required_role_id", "")
	minDays := b.db.ConfigInt(c.guildID, "giveaway_min_account_age_days", 0)
	minInvites := b.db.ConfigInt(c.guildID, "giveaway_min_invites", 0)

	// Opsiyonel bayrakları parse et (--davet=X, --yas=X, --yaş=X)
	words := strings.Fields(prize)
	var filteredWords []string
	for _, w := range words {
		switch {
		case strings.HasPrefix(w, "--davet="):
			if v, err := strconv.Atoi(strings.TrimPrefix(w, "--davet=")); err == nil && v >= 0 {
				minInvites = v
			}
		case strings.HasPrefix(w, "--yas="):
			if v, err := strconv.Atoi(strings.TrimPrefix(w, "--yas=")); err == nil && v >= 0 {
				minDays = v
			}
		case strings.HasPrefix(w, "--yaş="):
			if v, err := strconv.Atoi(strings.TrimPrefix(w, "--yaş=")); err == nil && v >= 0 {
				minDays = v
			}
		default:
			filteredWords = append(filteredWords, w)
		}
	}
	prize = strings.Join(filteredWords, " ")

	draft := database.Giveaway{
		GuildID:           c.guildID,
		ChannelID:         c.channelID,
		HostID:            c.user.ID,
		Prize:             prize,
		WinnerCount:       winners,
		EndsAt:            time.Now().Add(d).UnixMilli(),
		MinAccountAgeDays: minDays,
		MinInvites:        minInvites,
	}
	if role != "" {
		draft.RequiredRoleID.Valid = true
		draft.RequiredRoleID.String = role
	}
	msg, e := c.s.ChannelMessageSendComplex(c.channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{b.giveawayEmbed(draft, 0, false, nil)}, Components: giveawayButtons(draft, 0, false), AllowedMentions: &discordgo.MessageAllowedMentions{}})
	if e != nil {
		return e
	}
	id, e := b.db.CreateGiveaway(c.guildID, c.channelID, msg.ID, c.user.ID, prize, winners, role, minDays, minInvites, draft.EndsAt)
	if e != nil {
		return e
	}
	draft.ID = id
	draft.MessageID = msg.ID
	_, _ = c.s.ChannelMessageEditComplex(&discordgo.MessageEdit{Channel: msg.ChannelID, ID: msg.ID, Embeds: &[]*discordgo.MessageEmbed{b.giveawayEmbed(draft, 0, false, nil)}, Components: &[]discordgo.MessageComponent{giveawayButtons(draft, 0, false)[0]}})
	b.scheduleGiveaway(draft)
	return c.text(fmt.Sprintf("🎉 Çekiliş #%d başarıyla başlatıldı! Bitiş: <t:%d:R> (<t:%d:F>)", id, draft.EndsAt/1000, draft.EndsAt/1000))
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

func (b *Bot) drawGiveawayWinners(guildID string, entries []string, n int) []string {
	if len(entries) == 0 || n <= 0 {
		return nil
	}

	multRole := ""
	multVal := 1
	if b != nil && b.db != nil && guildID != "" {
		multRole = b.db.ConfigString(guildID, "giveaway_multiplier_role_id", "")
		multVal = b.db.ConfigInt(guildID, "giveaway_multiplier_value", 1)
	}

	if multRole == "" || multVal <= 1 || b == nil || b.dg == nil {
		return chooseWinners(entries, n)
	}

	var pool []string
	for _, userID := range entries {
		weight := 1
		if member, err := b.dg.State.Member(guildID, userID); err == nil && member != nil {
			for _, r := range member.Roles {
				if r == multRole {
					weight = multVal
					break
				}
			}
		} else if member, err := b.dg.GuildMember(guildID, userID); err == nil && member != nil {
			for _, r := range member.Roles {
				if r == multRole {
					weight = multVal
					break
				}
			}
		}

		for w := 0; w < weight; w++ {
			pool = append(pool, userID)
		}
	}

	mathrand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	var winners []string
	seen := make(map[string]bool)
	for _, winnerID := range pool {
		if !seen[winnerID] {
			seen[winnerID] = true
			winners = append(winners, winnerID)
			if len(winners) == n {
				break
			}
		}
	}

	return winners
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
	winners := b.drawGiveawayWinners(g.GuildID, entries, g.WinnerCount)
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
	if g.MinInvites > 0 {
		userInvites := b.db.UserNetInvites(g.GuildID, userOf(i).ID)
		if userInvites < g.MinInvites {
			return b.followInteraction(s, i, fmt.Sprintf("❌ Bu çekilişe katılabilmek için en az **%d** geçerli davetinin olması gerekir!\n📊 Senin mevcut net davetin: **%d**\n💡 Bir arkadaşını sunucuya davet ederek hemen katılabilirsin!", g.MinInvites, userInvites))
		}
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
		multRole := b.db.ConfigString(g.GuildID, "giveaway_multiplier_role_id", "")
		multVal := b.db.ConfigInt(g.GuildID, "giveaway_multiplier_value", 1)
		userMultiplier := 1
		if multRole != "" && multVal > 1 && i.Member != nil {
			for _, r := range i.Member.Roles {
				if r == multRole {
					userMultiplier = multVal
					break
				}
			}
		}

		var lines []string
		lines = append(lines, "🎉 **Çekilişe katılımınız başarıyla kaydedildi!**", "")
		lines = append(lines, fmt.Sprintf("🎁 **Ödül:** **%s**", g.Prize))
		lines = append(lines, fmt.Sprintf("👥 **Toplam Katılımcı:** **%d** kişi", len(entries)))
		lines = append(lines, fmt.Sprintf("🎲 **Tahmini Kazanma Şansınız:** **%s**", giveawayChanceDetail(len(entries), g.WinnerCount)))
		if userMultiplier > 1 {
			lines = append(lines, fmt.Sprintf("🚀 **Çarpan Bonusu:** <@&%s> rolüne sahip olduğunuz için **%dx** kat şansınız var!", multRole, userMultiplier))
		} else if multRole != "" && multVal > 1 {
			lines = append(lines, fmt.Sprintf("💡 **Tavsiye:** <@&%s> rolünü alarak kazanma şansını **%dx** katına çıkarabilirsin!", multRole, multVal))
		}
		lines = append(lines, fmt.Sprintf("⏰ **Sonuç Tarihi:** <t:%d:R> *(<t:%d:F>)*", g.EndsAt/1000, g.EndsAt/1000))
		lines = append(lines, "")
		lines = append(lines, "> 🤫 *Pssst... Ben seni tutuyorum, aramızda kalsın kimseye söyleme! 😉*")
		text = strings.Join(lines, "\n")
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
		if g.MinInvites > 0 {
			userInvites := b.db.UserNetInvites(g.GuildID, user.ID)
			if userInvites < g.MinInvites {
				missingReqs = append(missingReqs, fmt.Sprintf("• En az **%d** geçerli davetiniz olmalı (Mevcut net davetiniz: **%d**). 💡 Sunucuya arkadaşlarınızı davet ederek hemen katılabilirsiniz!", g.MinInvites, userInvites))
			}
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

	multRole := b.db.ConfigString(g.GuildID, "giveaway_multiplier_role_id", "")
	multVal := b.db.ConfigInt(g.GuildID, "giveaway_multiplier_value", 1)
	userMultiplier := 1
	if multRole != "" && multVal > 1 && i.Member != nil {
		for _, r := range i.Member.Roles {
			if r == multRole {
				userMultiplier = multVal
				break
			}
		}
	}

	var statusLines []string
	statusLines = append(statusLines, "🎯 **Çekiliş Katılım Bilgileriniz**", "")
	statusLines = append(statusLines, "✅ **Durum:** Çekilişe katıldınız!")
	statusLines = append(statusLines, fmt.Sprintf("🎁 **Ödül:** **%s**", g.Prize))
	statusLines = append(statusLines, fmt.Sprintf("🏆 **Kazanan Sayısı:** **%d** kişi", g.WinnerCount))
	statusLines = append(statusLines, fmt.Sprintf("👥 **Toplam Katılımcı:** **%d** kişi", len(entries)))
	statusLines = append(statusLines, fmt.Sprintf("🎲 **Tahmini Kazanma Şansınız:** **%s**", giveawayChanceDetail(len(entries), g.WinnerCount)))
	if userMultiplier > 1 {
		statusLines = append(statusLines, fmt.Sprintf("🚀 **Çarpan Bonusu:** <@&%s> rolü sayesinde **%dx** kat şans!", multRole, userMultiplier))
	} else if multRole != "" && multVal > 1 {
		statusLines = append(statusLines, fmt.Sprintf("💡 **Tavsiye:** <@&%s> rolünü alarak kazanma şansını **%dx** katına çıkarabilirsin!", multRole, multVal))
	}
	statusLines = append(statusLines, fmt.Sprintf("⏰ **Sonuç:** <t:%d:R> *(<t:%d:F>)*", g.EndsAt/1000, g.EndsAt/1000))
	statusLines = append(statusLines, "")
	statusLines = append(statusLines, "> 🤫 *Pssst... Ben seni tutuyorum, aramızda kalsın kimseye söyleme! 😉*")

	return ephemeral(s, i, strings.Join(statusLines, "\n"))
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

	wins := b.drawGiveawayWinners(g.GuildID, entries, g.WinnerCount)
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
		wins := b.drawGiveawayWinners(g.GuildID, entries, n)
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
