package bot

import "github.com/bwmarrin/discordgo"

func (b *Bot) ticketSetupWizard(guild string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	s, _ := b.db.Settings(guild)
	panel := b.db.ConfigString(guild, "ticket_panel_channel_id", "")
	support := b.db.ConfigString(guild, "ticket_support_role_id", "")
	em := embed("🎫 Etkileşimli Ticket Kurulum Paneli", "Menülerden kategori, panel kanalı, log kanalı ve destek rolünü seç; ardından paneli gönder.", colorPrimary)
	em.Fields = []*discordgo.MessageEmbedField{{Name: "Kategori", Value: channelValue(s.TicketCategoryID.Valid, s.TicketCategoryID.String), Inline: true}, {Name: "Panel Kanalı", Value: channelValue(panel != "", panel), Inline: true}, {Name: "Log Kanalı", Value: channelValue(s.TicketLogChannelID.Valid, s.TicketLogChannelID.String), Inline: true}, {Name: "Destek Rolü", Value: roleValue(support != "", support), Inline: true}}
	selectChannel := func(id, placeholder string, typ discordgo.ChannelType) discordgo.ActionsRow {
		return row(discordgo.SelectMenu{CustomID: id + ":" + guild, Placeholder: placeholder, MenuType: discordgo.ChannelSelectMenu, ChannelTypes: []discordgo.ChannelType{typ}, MinValues: intp(1), MaxValues: 1})
	}
	selectRole := func(id, placeholder string) discordgo.ActionsRow {
		return row(discordgo.SelectMenu{CustomID: id + ":" + guild, Placeholder: placeholder, MenuType: discordgo.RoleSelectMenu, MinValues: intp(1), MaxValues: 1})
	}
	components := []discordgo.MessageComponent{selectChannel("setup_channel:ticketcategory", "1. Ticket kategorisini seç", discordgo.ChannelTypeGuildCategory), selectChannel("ticketsetup_panelchan", "2. Panel kanalını seç", discordgo.ChannelTypeGuildText), selectChannel("setup_channel:ticketlog", "3. Log kanalını seç", discordgo.ChannelTypeGuildText), selectRole("setup_role:ticketsupport", "4. Destek rolünü seç"), row(button("setup_ticket_texts:"+guild, "Metinleri Düzenle", discordgo.PrimaryButton, "📝"), button("ticketsetup_deploy:"+guild, "Paneli Gönder", discordgo.SuccessButton, "🚀"))}
	return em, components
}
