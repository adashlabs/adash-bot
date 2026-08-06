package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) setupEmbed(guild, section string) *discordgo.MessageEmbed {
	s, _ := b.db.Settings(guild)
	title := "⚙️ Adash Kurulum"
	desc := "Menüden bir bölüm seç. Değişiklikler anında SQLite veritabanına kaydedilir."
	fields := []*discordgo.MessageEmbedField{}
	switch section {
	case "welcome":
		fields = []*discordgo.MessageEmbedField{{Name: "Hoş geldin", Value: boolIcon(s.WelcomeEnabled) + " · " + channelValue(s.WelcomeChannelID.Valid, s.WelcomeChannelID.String)}, {Name: "Görüşürüz", Value: boolIcon(s.FarewellEnabled) + " · " + channelValue(s.FarewellChannelID.Valid, s.FarewellChannelID.String)}, {Name: "Otomatik rol", Value: roleValue(s.AutoroleID.Valid, s.AutoroleID.String)}}
	case "games":
		fields = []*discordgo.MessageEmbedField{{Name: "Sayı saymaca", Value: boolIcon(s.CountingEnabled) + " · " + channelValue(s.CountingChannelID.Valid, s.CountingChannelID.String)}, {Name: "Kelime türetmece", Value: boolIcon(s.WordChainEnabled) + " · " + channelValue(s.WordChainChannelID.Valid, s.WordChainChannelID.String)}}
	case "ticket":
		fields = []*discordgo.MessageEmbedField{{Name: "Kategori", Value: channelValue(s.TicketCategoryID.Valid, s.TicketCategoryID.String)}, {Name: "Log kanalı", Value: channelValue(s.TicketLogChannelID.Valid, s.TicketLogChannelID.String)}, {Name: "Destek rolü", Value: roleValue(b.db.ConfigString(guild, "ticket_support_role_id", "") != "", b.db.ConfigString(guild, "ticket_support_role_id", ""))}}
	case "giveaway":
		fields = []*discordgo.MessageEmbedField{{Name: "Log kanalı", Value: channelValue(b.db.ConfigString(guild, "giveaway_log_channel_id", "") != "", b.db.ConfigString(guild, "giveaway_log_channel_id", ""))}, {Name: "Gerekli rol", Value: roleValue(b.db.ConfigString(guild, "giveaway_required_role_id", "") != "", b.db.ConfigString(guild, "giveaway_required_role_id", ""))}, {Name: "Minimum hesap yaşı", Value: fmt.Sprintf("%d gün", b.db.ConfigInt(guild, "giveaway_min_account_age_days", 0))}}
	case "ai":
		fields = []*discordgo.MessageEmbedField{{Name: "Durum", Value: boolIcon(b.db.ConfigBool(guild, "ai_enabled", true))}, {Name: "Sohbet kanalı", Value: channelValue(s.AIChannelID.Valid, s.AIChannelID.String)}, {Name: "Model", Value: valueOr(b.cfg.OpenAIModel, "Ayarlanmamış")}}
	case "modlog":
		fields = []*discordgo.MessageEmbedField{{Name: "Moderasyon logu", Value: channelValue(s.ModLogChannelID.Valid, s.ModLogChannelID.String)}, {Name: "İtiraz kanalı", Value: channelValue(b.db.ConfigString(guild, "appeal_channel_id", "") != "", b.db.ConfigString(guild, "appeal_channel_id", ""))}}
	default:
		g, u, c := b.db.Stats()
		fields = []*discordgo.MessageEmbedField{{Name: "Sunucu ayarları", Value: "Karşılama, oyun, ticket, çekiliş, AI ve ModLog bölümlerini menüden yönet."}, {Name: "Bot verileri", Value: fmt.Sprintf("%d sunucu · %d kullanıcı · %d komut", g, u, c)}}
	}
	return &discordgo.MessageEmbed{Title: title, Description: desc, Color: colorPrimary, Fields: fields, Footer: &discordgo.MessageEmbedFooter{Text: "Bölüm: " + section}}
}
func channelValue(ok bool, id string) string {
	if ok && id != "" {
		return "<#" + id + ">"
	}
	return "Ayarlı değil"
}
func roleValue(ok bool, id string) string {
	if ok && id != "" {
		return "<@&" + id + ">"
	}
	return "Ayarlı değil"
}
func setupMenu(guild string) discordgo.ActionsRow {
	return row(discordgo.SelectMenu{CustomID: "setup_section:" + guild, Placeholder: "Kurulum bölümü seç", Options: []discordgo.SelectMenuOption{{Label: "Genel Bakış", Value: "genel", Emoji: &discordgo.ComponentEmoji{Name: "🏠"}}, {Label: "Karşılama", Value: "welcome", Emoji: &discordgo.ComponentEmoji{Name: "👋"}}, {Label: "Oyunlar", Value: "games", Emoji: &discordgo.ComponentEmoji{Name: "🎮"}}, {Label: "Ticket", Value: "ticket", Emoji: &discordgo.ComponentEmoji{Name: "🎫"}}, {Label: "Çekiliş", Value: "giveaway", Emoji: &discordgo.ComponentEmoji{Name: "🎉"}}, {Label: "Yapay Zekâ", Value: "ai", Emoji: &discordgo.ComponentEmoji{Name: "🤖"}}, {Label: "ModLog", Value: "modlog", Emoji: &discordgo.ComponentEmoji{Name: "🛡️"}}}})
}
func setupComponents(guild, section string) []discordgo.MessageComponent {
	out := []discordgo.MessageComponent{setupMenu(guild)}
	channel := func(id, placeholder string, types ...discordgo.ChannelType) discordgo.ActionsRow {
		return row(discordgo.SelectMenu{CustomID: id + ":" + guild, Placeholder: placeholder, MenuType: discordgo.ChannelSelectMenu, ChannelTypes: types, MinValues: intp(1), MaxValues: 1})
	}
	role := func(id, placeholder string) discordgo.ActionsRow {
		return row(discordgo.SelectMenu{CustomID: id + ":" + guild, Placeholder: placeholder, MenuType: discordgo.RoleSelectMenu, MinValues: intp(1), MaxValues: 1})
	}
	switch section {
	case "welcome":
		out = append(out, channel("setup_channel:welcome", "Hoş geldin kanalı", discordgo.ChannelTypeGuildText), channel("setup_channel:farewell", "Görüşürüz kanalı", discordgo.ChannelTypeGuildText), role("setup_role:autorole", "Otomatik rol"), row(button("setup_toggle:welcome:"+guild, "Hoş geldin Aç/Kapat", discordgo.SuccessButton, "👋"), button("setup_toggle:farewell:"+guild, "Görüşürüz Aç/Kapat", discordgo.SecondaryButton, "👋"), button("setup_edit_messages:"+guild, "Mesajları Düzenle", discordgo.PrimaryButton, "✏️")))
	case "games":
		out = append(out, channel("setup_channel:counting", "Sayı saymaca kanalı", discordgo.ChannelTypeGuildText), channel("setup_channel:word", "Kelime türetmece kanalı", discordgo.ChannelTypeGuildText), row(button("setup_toggle:counting:"+guild, "Sayı Aç/Kapat", discordgo.SuccessButton, "🔢"), button("setup_toggle:word:"+guild, "Kelime Aç/Kapat", discordgo.SuccessButton, "🔤")), row(button("setup_reset:counting:"+guild, "Sayıyı Sıfırla", discordgo.DangerButton, "🧹"), button("setup_reset:word:"+guild, "Kelimeleri Sıfırla", discordgo.DangerButton, "🧹")))
	case "ticket":
		out = append(out, channel("setup_channel:ticketcategory", "Ticket kategorisi", discordgo.ChannelTypeGuildCategory), channel("setup_channel:ticketlog", "Ticket log kanalı", discordgo.ChannelTypeGuildText), role("setup_role:ticketsupport", "Destek rolü"), row(button("setup_ticket_texts:"+guild, "Panel Metinleri", discordgo.PrimaryButton, "✏️"), button("ticketsetup_deploy:"+guild, "Paneli Bu Kanala Gönder", discordgo.SuccessButton, "🚀"), button("setup_clear:ticketsupport:"+guild, "Destek Rolünü Temizle", discordgo.SecondaryButton, "🧹")))
	case "giveaway":
		out = append(out, channel("setup_channel:giveawaylog", "Çekiliş log kanalı", discordgo.ChannelTypeGuildText), role("setup_role:giveawayrole", "Gerekli rol"), row(button("setup_giveaway_rules:"+guild, "Katılım Kuralları", discordgo.PrimaryButton, "⚙️"), button("setup_giveaway_create_btn:"+guild, "Çekiliş Başlat", discordgo.SuccessButton, "🎉"), button("setup_clear:giveawayrole:"+guild, "Rol Şartını Temizle", discordgo.SecondaryButton, "🧹")))
	case "ai":
		out = append(out, channel("setup_channel:ai", "AI sohbet kanalı", discordgo.ChannelTypeGuildText), row(button("setup_ai_prompt:"+guild, "Sistem Promptu", discordgo.PrimaryButton, "✏️"), button("setup_toggle:ai:"+guild, "AI Aç/Kapat", discordgo.SuccessButton, "🤖"), button("setup_clear:aichannel:"+guild, "Sohbet Kanalını Temizle", discordgo.SecondaryButton, "🧹")))
	case "modlog":
		out = append(out, channel("setup_channel:modlog", "Moderasyon log kanalı", discordgo.ChannelTypeGuildText), channel("setup_channel:appeal", "İtiraz kanalı", discordgo.ChannelTypeGuildText))
	}
	return out
}
func intp(v int) *int { return &v }
func (b *Bot) setupModal(ses *discordgo.Session, i *discordgo.InteractionCreate, kind string) error {
	guild := i.GuildID
	var title, id string
	var inputs []discordgo.MessageComponent
	add := func(cid, label, value string, style discordgo.TextInputStyle, max int) {
		inputs = append(inputs, row(discordgo.TextInput{CustomID: cid, Label: label, Style: style, Required: true, MaxLength: max, Value: trunc(value, max)}))
	}
	switch kind {
	case "messages":
		s, _ := b.db.Settings(guild)
		title = "Karşılama Mesajları"
		id = "setup_messages:" + guild
		add("welcome_message", "Hoş geldin mesajı", s.WelcomeMessage, discordgo.TextInputParagraph, 1000)
		add("farewell_message", "Görüşürüz mesajı", s.FarewellMessage, discordgo.TextInputParagraph, 1000)
	case "ticket":
		title = "Ticket Metinleri"
		id = "setup_ticket_modal:" + guild
		add("ticket_panel_title", "Panel başlığı", b.db.ConfigString(guild, "ticket_panel_title", "🎫 Destek Talebi"), discordgo.TextInputShort, 256)
		add("ticket_panel_description", "Panel açıklaması", b.db.ConfigString(guild, "ticket_panel_description", "Destek ekibimize ulaşmak için aşağıdaki düğmeye bas."), discordgo.TextInputParagraph, 1500)
		add("ticket_welcome_message", "Ticket karşılama mesajı", b.db.ConfigString(guild, "ticket_welcome_message", "{user}, talebin oluşturuldu. Sorununu ayrıntılı biçimde anlat."), discordgo.TextInputParagraph, 1500)
		add("ticket_panel_button_label", "Düğme yazısı", b.db.ConfigString(guild, "ticket_panel_button_label", "Destek Talebi Aç"), discordgo.TextInputShort, 80)
		add("ticket_panel_button_emoji", "Düğme emojisi", b.db.ConfigString(guild, "ticket_panel_button_emoji", "🎫"), discordgo.TextInputShort, 20)
	case "ai":
		title = "Yapay Zekâ Sistem Promptu"
		id = "setup_ai_prompt_modal:" + guild
		add("ai_system_prompt", "Sistem promptu", b.db.ConfigString(guild, "ai_system_prompt", defaultSystemPrompt), discordgo.TextInputParagraph, 4000)
	case "giveaway":
		title = "Çekiliş Katılım Kuralları"
		id = "setup_giveaway_modal:" + guild
		add("min_account_age_days", "Minimum hesap yaşı (0-365)", strconv.Itoa(b.db.ConfigInt(guild, "giveaway_min_account_age_days", 0)), discordgo.TextInputShort, 3)
	case "create":
		title = "🎉 Çekiliş Oluştur"
		id = "setup_giveaway_create_modal"
		add("duration", "Süre (10m, 2h, 3d)", "1h", discordgo.TextInputShort, 20)
		add("winners", "Kazanan sayısı", "1", discordgo.TextInputShort, 2)
		add("prize", "Ödül", "", discordgo.TextInputParagraph, 1000)
	default:
		return nil
	}
	return ses.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{CustomID: id, Title: title, Components: inputs}})
}
func modalValues(i *discordgo.InteractionCreate) map[string]string {
	out := map[string]string{}
	for _, r := range i.ModalSubmitData().Components {
		if row, ok := r.(*discordgo.ActionsRow); ok {
			for _, x := range row.Components {
				if in, ok := x.(*discordgo.TextInput); ok {
					out[in.CustomID] = strings.TrimSpace(in.Value)
				}
			}
		}
	}
	return out
}
