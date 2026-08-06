package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) target(c *commandContext, input string) (*discordgo.User, *discordgo.Member, error) {
	id := mentionID(input)
	if id == "" {
		return nil, nil, fmt.Errorf("geçerli bir kullanıcı belirt")
	}
	m, e := c.s.GuildMember(c.guildID, id)
	if e == nil {
		return m.User, m, nil
	}
	u, e := c.s.User(id)
	if e != nil {
		return nil, nil, fmt.Errorf("kullanıcı bulunamadı")
	}
	return u, nil, nil
}
func highest(g *discordgo.Guild, m *discordgo.Member) int {
	best := 0
	for _, id := range m.Roles {
		for _, r := range g.Roles {
			if r.ID == id && r.Position > best {
				best = r.Position
			}
		}
	}
	return best
}
func (b *Bot) moderationCheck(c *commandContext, target *discordgo.Member, perm int64) error {
	if e := c.require(perm); e != nil {
		return e
	}
	botID := c.s.State.User.ID
	bp, e := c.s.UserChannelPermissions(botID, c.channelID)
	if e != nil || bp&perm == 0 && bp&discordgo.PermissionAdministrator == 0 {
		return fmt.Errorf("botun gerekli Discord yetkisi yok")
	}
	if target == nil {
		return nil
	}
	g, e := c.s.Guild(c.guildID)
	if e != nil {
		return e
	}
	if target.User.ID == c.user.ID {
		return fmt.Errorf("bu işlemi kendine uygulayamazsın")
	}
	if target.User.ID == g.OwnerID {
		return fmt.Errorf("sunucu sahibine işlem uygulanamaz")
	}
	if target.User.ID == botID {
		return fmt.Errorf("bu işlemi kendime uygulayamam")
	}
	actor := c.member
	if actor == nil {
		actor, _ = c.s.GuildMember(c.guildID, c.user.ID)
	}
	if actor == nil || actor.User == nil {
		return fmt.Errorf("yetkili üye bilgisi alınamadı")
	}
	bot, _ := c.s.GuildMember(c.guildID, botID)
	if actor.User.ID != g.OwnerID && highest(g, actor) <= highest(g, target) {
		return fmt.Errorf("kendi en yüksek rolüne eşit veya üstteki üyeye işlem uygulayamazsın")
	}
	if bot == nil || highest(g, bot) <= highest(g, target) {
		return fmt.Errorf("hedef üyenin rolü botun en yüksek rolüne eşit veya üstte")
	}
	return nil
}
func (b *Bot) requestConfirmation(c *commandContext, title, target, reason, details string, action func() error) error {
	id := token()
	item := confirmation{
		UserID: c.user.ID, GuildID: c.guildID, Title: title, Target: target,
		Reason: reason, Details: details, Expires: time.Now().Add(30 * time.Second), Action: action,
	}
	b.mu.Lock()
	b.confirms[id] = item
	b.mu.Unlock()
	return c.embed(
		moderationConfirmationEmbed(item),
		row(
			button("modconfirm:yes:"+id, "İşlemi Onayla", discordgo.DangerButton, "✅"),
			button("modconfirm:no:"+id, "Vazgeç", discordgo.SecondaryButton, "✖️"),
		),
	)
}
func (b *Bot) runModeration(c *commandContext, name string, args []string) error {
	if name == "lock" {
		return b.lock(c, args)
	}
	if name == "slowmode" {
		return b.slowmode(c, args)
	}
	if name == "unban" {
		if len(args) == 0 {
			return fmt.Errorf("kullanıcı ID belirt")
		}
		u, _, e := b.target(c, args[0])
		if e != nil {
			return e
		}
		if e = b.moderationCheck(c, nil, discordgo.PermissionBanMembers); e != nil {
			return e
		}
		reason := reasonFrom(args, 1)
		return b.requestConfirmation(c, "Kullanıcının Yasağını Kaldır", u.Username+" (`"+u.ID+"`)", reason, "", func() error {
			if e := c.s.GuildBanDelete(c.guildID, u.ID); e != nil {
				return e
			}
			_ = b.db.LogMod(c.guildID, u.ID, c.user.ID, "unban", reason, nil)
			b.sendModLog(c.guildID, "unban", u.ID, c.user.ID, reason, "")
			return b.followup(c, "🔓 Yasak kaldırıldı.", false)
		})
	}
	if len(args) == 0 {
		return fmt.Errorf("bir kullanıcı belirt")
	}
	u, m, e := b.target(c, args[0])
	if e != nil {
		return e
	}
	perm := int64(discordgo.PermissionModerateMembers)
	if name == "ban" {
		perm = discordgo.PermissionBanMembers
	}
	if name == "kick" {
		perm = discordgo.PermissionKickMembers
	}
	if e = b.moderationCheck(c, m, perm); e != nil {
		return e
	}
	if m == nil && name != "ban" {
		return fmt.Errorf("kullanıcı artık sunucuda değil")
	}
	switch name {
	case "ban":
		days := 0
		kept := []string{}
		for _, x := range args[1:] {
			if strings.HasPrefix(x, "--days=") {
				days, _ = strconv.Atoi(strings.TrimPrefix(x, "--days="))
				if days < 0 {
					days = 0
				}
				if days > 7 {
					days = 7
				}
			} else {
				kept = append(kept, x)
			}
		}
		reason := valueOr(strings.Join(kept, " "), "Sebep belirtilmedi")
		return b.requestConfirmation(c, "Kullanıcıyı Yasakla", u.Username+" (`"+u.ID+"`)", reason, fmt.Sprintf("Son %d günlük mesajlar silinecek.", days), func() error {
			if e := c.s.GuildBanCreateWithReason(c.guildID, u.ID, reason, days); e != nil {
				return e
			}
			_ = b.db.LogMod(c.guildID, u.ID, c.user.ID, "ban", reason, nil)
			b.sendModLog(c.guildID, "ban", u.ID, c.user.ID, reason, "")
			return b.followup(c, "🔨 Kullanıcı yasaklandı.", false)
		})
	case "kick":
		reason := reasonFrom(args, 1)
		return b.requestConfirmation(c, "Kullanıcıyı Sunucudan At", u.Username+" (`"+u.ID+"`)", reason, "", func() error {
			if e := c.s.GuildMemberDeleteWithReason(c.guildID, u.ID, reason); e != nil {
				return e
			}
			_ = b.db.LogMod(c.guildID, u.ID, c.user.ID, "kick", reason, nil)
			b.sendModLog(c.guildID, "kick", u.ID, c.user.ID, reason, "")
			return b.followup(c, "👢 Kullanıcı sunucudan atıldı.", false)
		})
	case "mute":
		if len(args) < 2 {
			return fmt.Errorf("1 saniye–28 gün arasında süre belirt")
		}
		d, e := parseDuration(args[1])
		if e != nil || d < time.Second || d > 28*24*time.Hour {
			return fmt.Errorf("1 saniye–28 gün arasında süre belirt")
		}
		reason := reasonFrom(args, 2)
		return b.requestConfirmation(c, "Kullanıcıyı Sustur", u.Username+" (`"+u.ID+"`)", reason, "Süre: "+formatDuration(d), func() error {
			until := time.Now().Add(d)
			if e := c.s.GuildMemberTimeout(c.guildID, u.ID, &until); e != nil {
				return e
			}
			ms := d.Milliseconds()
			_ = b.db.LogMod(c.guildID, u.ID, c.user.ID, "mute", reason, &ms)
			b.sendModLog(c.guildID, "mute", u.ID, c.user.ID, reason, "Süre: "+formatDuration(d))
			return b.followup(c, "🔇 Kullanıcı susturuldu.", false)
		})
	case "unmute":
		reason := reasonFrom(args, 1)
		return b.requestConfirmation(c, "Susturmayı Kaldır", u.Username+" (`"+u.ID+"`)", reason, "", func() error {
			if e := c.s.GuildMemberTimeout(c.guildID, u.ID, nil); e != nil {
				return e
			}
			_ = b.db.LogMod(c.guildID, u.ID, c.user.ID, "unmute", reason, nil)
			b.sendModLog(c.guildID, "unmute", u.ID, c.user.ID, reason, "")
			return b.followup(c, "🔊 Susturma kaldırıldı.", false)
		})
	case "warn":
		reason := reasonFrom(args, 1)
		count, _ := b.db.WarningCount(c.guildID, u.ID)
		threshold := b.db.ConfigInt(c.guildID, "warn_auto_threshold", 3)
		return b.requestConfirmation(c, "Kullanıcıya Uyarı Ver", u.Username+" (`"+u.ID+"`)", reason, fmt.Sprintf("Aktif uyarı: %d → %d", count, count+1), func() error {
			_, n, e := b.db.AddWarning(c.guildID, u.ID, c.user.ID, reason)
			if e != nil {
				return e
			}
			_ = b.db.LogMod(c.guildID, u.ID, c.user.ID, "warn", reason, nil)
			b.sendModLog(c.guildID, "warn", u.ID, c.user.ID, reason, fmt.Sprintf("Aktif uyarı: %d", n))
			if e = b.followup(c, fmt.Sprintf("⚠️ Kullanıcı uyarıldı. Aktif uyarı: **%d**", n), false); e != nil {
				return e
			}
			if n == threshold {
				ms := b.db.ConfigInt(c.guildID, "warn_auto_timeout_ms", 600000)
				d := time.Duration(ms) * time.Millisecond
				until := time.Now().Add(d)
				if e = c.s.GuildMemberTimeout(c.guildID, u.ID, &until); e == nil {
					v := d.Milliseconds()
					_ = b.db.LogMod(c.guildID, u.ID, c.s.State.User.ID, "mute", fmt.Sprintf("%d aktif uyarı sonrası otomatik susturma", threshold), &v)
					_ = b.followup(c, "🔇 Uyarı eşiğine ulaşıldığı için "+formatDuration(d)+" susturuldu.", false)
				}
			}
			return nil
		})
	case "clearwarns":
		return b.requestConfirmation(c, "Uyarıları Temizle", u.Username+" (`"+u.ID+"`)", "Tüm aktif uyarılar kapatılacak.", "", func() error {
			n, e := b.db.ClearWarnings(c.guildID, u.ID)
			if e != nil {
				return e
			}
			_ = b.db.LogMod(c.guildID, u.ID, c.user.ID, "clearwarns", fmt.Sprintf("%d uyarı temizlendi", n), nil)
			return b.followup(c, fmt.Sprintf("✅ **%d** aktif uyarı temizlendi.", n), false)
		})
	}
	return nil
}
func reasonFrom(args []string, start int) string {
	if start >= len(args) {
		return "Sebep belirtilmedi"
	}
	return trunc(valueOr(strings.Join(args[start:], " "), "Sebep belirtilmedi"), 500)
}
func valueOr(v, f string) string {
	if strings.TrimSpace(v) == "" {
		return f
	}
	return v
}
func (b *Bot) followup(c *commandContext, content string, ephemeral bool) error {
	em := moderationSuccessEmbed(content)
	if c.interaction != nil {
		flags := discordgo.MessageFlags(0)
		if ephemeral {
			flags = discordgo.MessageFlagsEphemeral
		}
		_, err := c.s.FollowupMessageCreate(c.interaction.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{em}, Flags: flags,
		})
		return err
	}
	_, err := c.s.ChannelMessageSendEmbed(c.channelID, em)
	return err
}
func (b *Bot) sendModLog(guild, action, target, moderator, reason, extra string) {
	settings, err := b.db.Settings(guild)
	if err != nil || !settings.ModLogChannelID.Valid {
		return
	}
	label, icon, colorValue := moderationActionMeta(action)
	em := &discordgo.MessageEmbed{
		Title: icon + " " + label,
		Color: colorValue,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Hedef", Value: str(target != "", "<@"+target+">", "Kanal işlemi"), Inline: true},
			{Name: "Yetkili", Value: "<@" + moderator + ">", Inline: true},
			{Name: "Sebep", Value: trunc(valueOr(reason, "Sebep belirtilmedi"), 1024)},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "Adash Moderasyon • " + action},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if extra != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "Ayrıntı", Value: trunc(extra, 1024)})
	}
	_, _ = b.dg.ChannelMessageSendEmbed(settings.ModLogChannelID.String, em)
}
func (b *Bot) showWarnings(c *commandContext, id string) error {
	u, _, e := b.target(c, id)
	if e != nil {
		return e
	}
	xs, e := b.db.Warnings(c.guildID, u.ID)
	if e != nil {
		return e
	}
	if len(xs) == 0 {
		return c.embed(embed("ℹ️ Aktif Uyarı Yok", "**"+u.Username+"** kullanıcısının aktif uyarısı bulunmuyor.", colorNeutral))
	}
	lines := make([]string, 0, len(xs))
	for _, x := range xs {
		lines = append(lines, fmt.Sprintf("`#%d` <@%s> · %s · <t:%d:R>", x.ID, x.ModeratorID, trunc(x.Reason, 180), x.CreatedAt/1000))
	}
	return c.embed(embed("⚠️ "+u.Username+" Uyarıları", strings.Join(lines, "\n"), colorWarning))
}
func (b *Bot) showCases(c *commandContext, id string) error {
	if e := c.require(discordgo.PermissionModerateMembers); e != nil {
		return e
	}
	if id != "" {
		id = mentionID(id)
	}
	xs, e := b.db.RecentModLogs(c.guildID, id, 10)
	if e != nil {
		return e
	}
	if len(xs) == 0 {
		return c.embed(embed("ℹ️ Kayıt Bulunamadı", "Bu filtreye uygun moderasyon kaydı bulunmuyor.", colorNeutral))
	}
	lines := make([]string, 0, len(xs))
	for _, x := range xs {
		reason := "Sebep belirtilmedi"
		if x.Reason.Valid {
			reason = x.Reason.String
		}
		lines = append(lines, fmt.Sprintf("`#%d` **%s** · <@%s> · %s · <t:%d:R>", x.ID, x.Action, x.UserID, trunc(reason, 120), x.Timestamp/1000))
	}
	return c.embed(embed("📚 Moderasyon Vakaları", strings.Join(lines, "\n"), colorNeutral))
}
func (b *Bot) clearMessages(c *commandContext, args []string) error {
	if e := c.require(discordgo.PermissionManageMessages); e != nil {
		return e
	}
	if len(args) == 0 {
		return fmt.Errorf("1–100 arasında sayı belirt")
	}
	n, e := strconv.Atoi(args[0])
	if e != nil || n < 1 || n > 100 {
		return fmt.Errorf("1–100 arasında sayı belirt")
	}
	target := ""
	if len(args) > 1 {
		target = mentionID(args[1])
	}
	msgs, e := c.s.ChannelMessages(c.channelID, 100, "", "", "")
	if e != nil {
		return e
	}
	cut := time.Now().Add(-14 * 24 * time.Hour)
	ids := []string{}
	for _, m := range msgs {
		if len(ids) >= n {
			break
		}
		if m.Timestamp.Before(cut) || target != "" && m.Author.ID != target {
			continue
		}
		ids = append(ids, m.ID)
	}
	if len(ids) == 0 {
		return fmt.Errorf("silinecek uygun mesaj bulunamadı")
	}
	if len(ids) == 1 {
		e = c.s.ChannelMessageDelete(c.channelID, ids[0])
	} else {
		e = c.s.ChannelMessagesBulkDelete(c.channelID, ids)
	}
	if e != nil {
		return e
	}
	count := int64(len(ids))
	_ = b.db.LogMod(c.guildID, valueOr(target, c.channelID), c.user.ID, "clear", "Kanal mesajları", &count)
	return c.embed(successEmbed("🗑️ Mesajlar Temizlendi", fmt.Sprintf("**%d** mesaj başarıyla kaldırıldı.", len(ids))))
}
func (b *Bot) lock(c *commandContext, args []string) error {
	if e := c.require(discordgo.PermissionManageChannels); e != nil {
		return e
	}
	unlock := len(args) > 0 && (args[0] == "aç" || args[0] == "ac" || args[0] == "unlock")
	return b.requestConfirmation(c, str(unlock, "Kanal Kilidini Aç", "Kanalı Kilitle"), "<#"+c.channelID+">", str(unlock, "Üyeler tekrar mesaj gönderebilecek.", "Üyeler mesaj gönderemeyecek."), "", func() error {
		allow, deny := int64(0), int64(discordgo.PermissionSendMessages)
		if unlock {
			allow = discordgo.PermissionSendMessages
			deny = 0
		}
		if e := c.s.ChannelPermissionSet(c.channelID, c.guildID, discordgo.PermissionOverwriteTypeRole, allow, deny); e != nil {
			return e
		}
		b.sendModLog(c.guildID, str(unlock, "unlock", "lock"), "", c.user.ID, "<#"+c.channelID+">", "")
		return b.followup(c, str(unlock, "🔓 Kanal kilidi açıldı.", "🔒 Kanal kilitlendi."), false)
	})
}
func (b *Bot) slowmode(c *commandContext, args []string) error {
	if e := c.require(discordgo.PermissionManageChannels); e != nil {
		return e
	}
	if len(args) == 0 {
		return fmt.Errorf("bir süre belirt")
	}
	d, e := parseDuration(args[0])
	if e != nil || d > 6*time.Hour {
		return fmt.Errorf("0–6 saat arasında süre belirt")
	}
	seconds := int(d.Seconds())
	_, e = c.s.ChannelEditComplex(c.channelID, &discordgo.ChannelEdit{RateLimitPerUser: &seconds})
	if e != nil {
		return e
	}
	b.sendModLog(c.guildID, "slowmode", "", c.user.ID, formatDuration(d), "")
	return c.embed(successEmbed("⏱️ Yavaş Mod Güncellendi", "Yeni süre: **"+formatDuration(d)+"**"))
}
func (b *Bot) modConfig(c *commandContext, args []string) error {
	if e := c.require(discordgo.PermissionManageServer); e != nil {
		return e
	}
	if len(args) < 1 {
		return fmt.Errorf("kullanım: modconfig warn <1-10> <10m-28d> veya appeal <#kanal|kapalı>")
	}
	if args[0] == "warn" {
		if len(args) < 3 {
			return fmt.Errorf("eşik ve süre belirt")
		}
		n, e := strconv.Atoi(args[1])
		d, x := parseDuration(args[2])
		if e != nil || x != nil || n < 1 || n > 10 || d < 10*time.Second || d > 28*24*time.Hour {
			return fmt.Errorf("geçersiz eşik veya süre")
		}
		if e = b.db.SetConfig(c.guildID, "warn_auto_threshold", n); e != nil {
			return e
		}
		if e = b.db.SetConfig(c.guildID, "warn_auto_timeout_ms", d.Milliseconds()); e != nil {
			return e
		}
		return c.embed(successEmbed("⚙️ Uyarı Otomasyonu Güncellendi", fmt.Sprintf("**%d. uyarıda** kullanıcıya **%s** susturma uygulanacak.", n, formatDuration(d))))
	}
	if args[0] == "appeal" {
		v := first(args[1:], "")
		if v == "kapalı" || v == "kapali" || v == "off" {
			_ = b.db.SetConfig(c.guildID, "appeal_channel_id", "")
			return c.embed(successEmbed("📨 İtiraz Kanalı Kapatıldı", "Yeni itirazlar bir kanala yönlendirilmeyecek."))
		}
		id := mentionID(v)
		ch, e := c.s.Channel(id)
		if e != nil || ch.Type == discordgo.ChannelTypeGuildCategory {
			return fmt.Errorf("geçerli metin kanalı belirt")
		}
		_ = b.db.SetConfig(c.guildID, "appeal_channel_id", id)
		return c.embed(successEmbed("📨 İtiraz Kanalı Güncellendi", "Yeni kanal: <#"+id+">"))
	}
	return fmt.Errorf("geçersiz modconfig işlemi")
}
func (b *Bot) appeal(c *commandContext, args []string) error {
	text := trunc(strings.Join(args, " "), 1500)
	if len([]rune(text)) < 20 {
		return fmt.Errorf("itirazını en az 20 karakterle yaz")
	}
	channel := b.db.ConfigString(c.guildID, "appeal_channel_id", "")
	if channel == "" {
		return fmt.Errorf("bu sunucuda itiraz kanalı ayarlanmamış")
	}
	em := embed("📨 Moderasyon İtirazı", safeText(text), colorWarning)
	em.Fields = []*discordgo.MessageEmbedField{{Name: "Gönderen", Value: c.user.Username + " (`" + c.user.ID + "`)"}}
	if _, e := c.s.ChannelMessageSendEmbed(channel, em); e != nil {
		return e
	}
	return c.embed(successEmbed("📨 İtiraz Gönderildi", "İtirazın yetkili ekibe güvenli şekilde iletildi."))
}
