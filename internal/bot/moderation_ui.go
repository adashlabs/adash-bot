package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func moderationConfirmationEmbed(x confirmation) *discordgo.MessageEmbed {
	em := &discordgo.MessageEmbed{
		Title:       "⚠️ Onay Gerekiyor",
		Description: "**" + x.Title + "** işlemini uygulamadan önce bilgileri kontrol et.",
		Color:       colorWarning,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Hedef", Value: valueOr(x.Target, "—"), Inline: true},
			{Name: "Yetkili", Value: "<@" + x.UserID + ">", Inline: true},
			{Name: "Sebep", Value: trunc(valueOr(x.Reason, "Sebep belirtilmedi"), 1024)},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "Bu onay yalnızca işlemi başlatan yetkili tarafından kullanılabilir."},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if x.Details != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "Ayrıntı", Value: trunc(x.Details, 1024)})
	}
	em.Fields = append(em.Fields, &discordgo.MessageEmbedField{Name: "Onay süresi", Value: fmt.Sprintf("<t:%d:R>", x.Expires.Unix())})
	return em
}

func moderationResultEmbed(x confirmation, state, detail string) *discordgo.MessageEmbed {
	title, colorValue := "✅ İşlem Uygulandı", colorSuccess
	if state == "cancelled" {
		title, colorValue = "↩️ İşlem İptal Edildi", colorNeutral
	}
	if state == "failed" {
		title, colorValue = "❌ İşlem Uygulanamadı", colorDanger
	}
	em := &discordgo.MessageEmbed{
		Title:       title,
		Description: detail,
		Color:       colorValue,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "İşlem", Value: x.Title, Inline: true},
			{Name: "Hedef", Value: valueOr(x.Target, "—"), Inline: true},
			{Name: "Yetkili", Value: "<@" + x.UserID + ">", Inline: true},
			{Name: "Sebep", Value: trunc(valueOr(x.Reason, "Sebep belirtilmedi"), 1024)},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return em
}

func completedModerationButton(state string) discordgo.ActionsRow {
	label, style, emoji := "Uygulandı", discordgo.SuccessButton, "✅"
	if state == "cancelled" {
		label, style, emoji = "İptal Edildi", discordgo.SecondaryButton, "✖️"
	}
	if state == "failed" {
		label, style, emoji = "Başarısız", discordgo.DangerButton, "⚠️"
	}
	b := button("moderation_complete", label, style, emoji)
	b.Disabled = true
	return row(b)
}

func moderationActionMeta(action string) (string, string, int) {
	switch action {
	case "ban":
		return "Kullanıcı Yasaklandı", "🔨", colorDanger
	case "unban":
		return "Yasak Kaldırıldı", "🔓", colorSuccess
	case "kick":
		return "Kullanıcı Sunucudan Atıldı", "👢", colorDanger
	case "mute":
		return "Kullanıcı Susturuldu", "🔇", colorWarning
	case "unmute":
		return "Susturma Kaldırıldı", "🔊", colorSuccess
	case "warn":
		return "Kullanıcı Uyarıldı", "⚠️", colorWarning
	case "clearwarns":
		return "Uyarılar Temizlendi", "🧹", colorSuccess
	case "lock":
		return "Kanal Kilitlendi", "🔒", colorWarning
	case "unlock":
		return "Kanal Kilidi Açıldı", "🔓", colorSuccess
	case "slowmode":
		return "Yavaş Mod Güncellendi", "⏱️", colorPrimary
	default:
		return strings.ToUpper(action), "🛡️", colorNeutral
	}
}
func moderationSuccessEmbed(content string) *discordgo.MessageEmbed {
	content = strings.TrimSpace(content)
	return successEmbed("✅ İşlem Tamamlandı", valueOr(content, "İşlem başarıyla tamamlandı."))
}
