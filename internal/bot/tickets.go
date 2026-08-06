package bot

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/adashlabs/adash-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) ticketPanel(guild string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	title := b.db.ConfigString(guild, "ticket_panel_title", "🎫 Destek Talebi")
	desc := b.db.ConfigString(guild, "ticket_panel_description", "Destek ekibimize ulaşmak için aşağıdaki düğmeye bas. Aynı anda yalnızca bir açık talebin olabilir.")
	label := b.db.ConfigString(guild, "ticket_panel_button_label", "Destek Talebi Aç")
	emoji := b.db.ConfigString(guild, "ticket_panel_button_emoji", "🎫")
	em := embed(trunc(title, 256), trunc(desc, 4000), colorPrimary)
	em.Footer = &discordgo.MessageEmbedFooter{Text: "Bir kullanıcı aynı anda yalnızca bir açık talep oluşturabilir."}
	return em, []discordgo.MessageComponent{row(button("ticket_open", trunc(label, 80), discordgo.PrimaryButton, emoji))}
}
func (b *Bot) sendTicketPanel(guild, channel string) error {
	em, components := b.ticketPanel(guild)
	_, e := b.dg.ChannelMessageSendComplex(channel, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{em}, Components: components})
	return e
}
func (b *Bot) ticketSetupCommand(c *commandContext, args []string) error {
	if e := c.require(discordgo.PermissionManageServer); e != nil {
		return e
	}
	if len(args) < 2 {
		em, components := b.ticketSetupWizard(c.guildID)
		return c.embed(em, components...)
	}
	category, panel := mentionID(args[0]), mentionID(args[1])
	cat, e := c.s.Channel(category)
	if e != nil || cat.Type != discordgo.ChannelTypeGuildCategory {
		return fmt.Errorf("geçerli bir kategori belirt")
	}
	ch, e := c.s.Channel(panel)
	if e != nil || ch.Type != discordgo.ChannelTypeGuildText {
		return fmt.Errorf("geçerli panel kanalı belirt")
	}
	if e = b.db.SetSetting(c.guildID, "ticket_category_id", category); e != nil {
		return e
	}
	if len(args) > 2 {
		if id := mentionID(args[2]); id != "" {
			_ = b.db.SetSetting(c.guildID, "ticket_log_channel_id", id)
		}
	}
	if len(args) > 3 {
		if id := mentionID(args[3]); id != "" {
			_ = b.db.SetConfig(c.guildID, "ticket_support_role_id", id)
		}
	}
	if e = b.sendTicketPanel(c.guildID, panel); e != nil {
		return e
	}
	return c.embed(successEmbed("🎫 Ticket Sistemi Kuruldu", fmt.Sprintf("Kategori: <#%s>\nPanel: <#%s>", category, panel)))
}
func (b *Bot) isSupport(c *commandContext) bool {
	p, e := c.permissions()
	if e == nil && (p&discordgo.PermissionManageChannels != 0 || p&discordgo.PermissionAdministrator != 0) {
		return true
	}
	role := b.db.ConfigString(c.guildID, "ticket_support_role_id", "")
	if role != "" && c.member != nil {
		for _, r := range c.member.Roles {
			if r == role {
				return true
			}
		}
	}
	return false
}
func (b *Bot) ticketCommand(c *commandContext, args []string) error {
	t, e := b.db.Ticket(c.channelID)
	if e != nil || t.ClosedAt.Valid {
		return fmt.Errorf("bu komut yalnızca açık ticket kanalında kullanılabilir")
	}
	if !b.isSupport(c) {
		return fmt.Errorf("ticket destek rolü veya Kanalları Yönet yetkisi gerekiyor")
	}
	if len(args) == 0 {
		return c.embed(embed("🎫 Ticket Yönetimi", "`ticket ekle @üye`\n`ticket çıkar @üye`\n`ticket adlandır yeni-ad`", colorPrimary))
	}
	action := strings.ToLower(args[0])
	if action == "ekle" || action == "add" || action == "çıkar" || action == "cikar" || action == "remove" {
		if len(args) < 2 {
			return fmt.Errorf("kullanıcı belirt")
		}
		u, m, e := b.target(c, args[1])
		if e != nil || m == nil {
			return fmt.Errorf("sunucudaki kullanıcı bulunamadı")
		}
		if action == "çıkar" || action == "cikar" || action == "remove" {
			if u.ID == t.OwnerID {
				return fmt.Errorf("ticket sahibi çıkarılamaz")
			}
			if e = c.s.ChannelPermissionDelete(c.channelID, u.ID); e != nil {
				return e
			}
			return c.text("👤 <@" + u.ID + "> ticket kanalından çıkarıldı.")
		}
		allow := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory)
		if e = c.s.ChannelPermissionSet(c.channelID, u.ID, discordgo.PermissionOverwriteTypeMember, allow, 0); e != nil {
			return e
		}
		return c.text("👤 <@" + u.ID + "> ticket kanalına eklendi.")
	}
	if action == "adlandır" || action == "adlandir" || action == "rename" {
		name := safeChannelName(strings.Join(args[1:], "-"))
		if len(name) < 2 {
			return fmt.Errorf("geçerli kanal adı belirt")
		}
		if _, e = c.s.ChannelEdit(c.channelID, &discordgo.ChannelEdit{Name: name}); e != nil {
			return e
		}
		return c.text("✏️ Ticket kanalının adı **" + name + "** oldu.")
	}
	return fmt.Errorf("geçersiz ticket işlemi")
}
func safeChannelName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9çğıöşü-]+`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return trunc(strings.Trim(s, "-"), 80)
}
func (b *Bot) ticketOpenModal(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	inputs := []discordgo.MessageComponent{row(discordgo.TextInput{CustomID: "subject", Label: "Konu", Style: discordgo.TextInputShort, Required: true, MaxLength: 100}), row(discordgo.TextInput{CustomID: "description", Label: "Sorunun / talebin", Style: discordgo.TextInputParagraph, Required: true, MinLength: 10, MaxLength: 1500}), row(discordgo.TextInput{CustomID: "priority", Label: "Öncelik: düşük / normal / yüksek / acil", Style: discordgo.TextInputShort, Required: true, Value: "normal", MaxLength: 10})}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{CustomID: "ticket_open_modal", Title: "Destek Talebi Oluştur", Components: inputs}})
}
func (b *Bot) openTicketFromModal(s *discordgo.Session, i *discordgo.InteractionCreate, v map[string]string) error {
	existing, e := b.db.OpenTicket(i.GuildID, userOf(i).ID)
	if e == nil {
		return ephemeral(s, i, "Zaten açık bir talebin var: <#"+existing.ChannelID+">")
	}
	priority := strings.ToLower(v["priority"])
	if priority == "dusuk" {
		priority = "düşük"
	}
	if priority == "yuksek" {
		priority = "yüksek"
	}
	valid := map[string]bool{"düşük": true, "normal": true, "yüksek": true, "acil": true}
	if !valid[priority] {
		return fmt.Errorf("öncelik düşük, normal, yüksek veya acil olmalı")
	}
	settings, e := b.db.Settings(i.GuildID)
	if e != nil {
		return e
	}
	category := ""
	if settings.TicketCategoryID.Valid {
		category = settings.TicketCategoryID.String
	}
	overwrites := []*discordgo.PermissionOverwrite{{ID: i.GuildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel}, {ID: userOf(i).ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory | discordgo.PermissionAttachFiles}, {ID: s.State.User.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory | discordgo.PermissionManageChannels | discordgo.PermissionManageMessages}}
	support := b.db.ConfigString(i.GuildID, "ticket_support_role_id", "")
	if support != "" {
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{ID: support, Type: discordgo.PermissionOverwriteTypeRole, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory})
	}
	name := "talep-" + safeChannelName(priority+"-"+userOf(i).Username)
	ch, e := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{Name: name, Type: discordgo.ChannelTypeGuildText, ParentID: category, Topic: fmt.Sprintf("Ticket sahibi: %s (%s) · %s", userOf(i).Username, userOf(i).ID, v["subject"]), PermissionOverwrites: overwrites})
	if e != nil {
		return e
	}
	if e = b.db.CreateTicket(ch.ID, i.GuildID, userOf(i).ID, category, "destek", priority, v["subject"], v["description"]); e != nil {
		_, _ = s.ChannelDelete(ch.ID)
		return e
	}
	welcome := b.db.ConfigString(i.GuildID, "ticket_welcome_message", "{user}, talebin oluşturuldu. Sorununu ayrıntılı biçimde anlat.")
	welcome = strings.ReplaceAll(strings.ReplaceAll(welcome, "{user}", "<@"+userOf(i).ID+">"), "{username}", userOf(i).Username)
	em := embed("🎫 "+trunc(v["subject"], 240), trunc(welcome, 2500)+"\n\n**Açıklama**\n"+trunc(v["description"], 1200), strColor(priority == "acil", colorDanger, strColor(priority == "yüksek", colorWarning, colorSuccess)))
	em.Fields = []*discordgo.MessageEmbedField{{Name: "Talep sahibi", Value: "<@" + userOf(i).ID + ">", Inline: true}, {Name: "Öncelik", Value: priority, Inline: true}, {Name: "Durum", Value: "🟢 Açık", Inline: true}}
	components := ticketControls()
	content := ""
	if support != "" {
		content = "<@&" + support + ">"
	}
	_, _ = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{Content: content, Embeds: []*discordgo.MessageEmbed{em}, Components: components, AllowedMentions: &discordgo.MessageAllowedMentions{Roles: []string{support}}})
	b.ticketLog(i.GuildID, "🎫 Ticket açıldı: <#"+ch.ID+"> · sahibi: <@"+userOf(i).ID+">", nil)
	return ephemeral(s, i, "Talebin oluşturuldu: <#"+ch.ID+">")
}
func ticketControls() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{row(button("ticket_claim", "Talebi Üstlen", discordgo.SuccessButton, "🙋"), button("ticket_add_btn", "Üye Ekle", discordgo.PrimaryButton, "➕"), button("ticket_rename_btn", "Adlandır", discordgo.SecondaryButton, "✏️")), row(button("ticket_status", "Durum Değiştir", discordgo.PrimaryButton, "📌"), button("ticket_close_btn", "Talebi Kapat", discordgo.DangerButton, "🔒"))}
}
func (b *Bot) ticketComponent(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	id := i.MessageComponentData().CustomID
	c := &commandContext{b: b, s: s, guildID: i.GuildID, channelID: i.ChannelID, user: userOf(i), member: i.Member}
	t, e := b.db.Ticket(i.ChannelID)
	if e != nil || t.ClosedAt.Valid {
		return fmt.Errorf("bu kanal açık ticket değil")
	}
	if id == "ticket_close_btn" {
		return ticketModal(s, i, "ticket_close_modal", "Ticket Talebini Kapat", "reason_input", "Kapanış Sebebi / Notu", discordgo.TextInputParagraph, "Çözüldü / Talebiniz tamamlandı.")
	}
	if !b.isSupport(c) {
		return fmt.Errorf("ticket destek rolü veya Kanalları Yönet yetkisi gerekiyor")
	}
	switch id {
	case "ticket_claim":
		ok, e := b.db.ClaimTicket(i.ChannelID, userOf(i).ID)
		if e != nil {
			return e
		}
		if !ok {
			return fmt.Errorf("talep zaten üstlenildi")
		}
		_ = ephemeral(s, i, "Talebi üstlendin.")
		_, _ = s.ChannelMessageSend(i.ChannelID, "🙋 Bu talebi <@"+userOf(i).ID+"> üstlendi.")
	case "ticket_add_btn":
		return ticketModal(s, i, "ticket_add_modal", "Kanala Üye Ekle", "user_input", "Üye ID", discordgo.TextInputShort, "")
	case "ticket_rename_btn":
		return ticketModal(s, i, "ticket_rename_modal", "Ticket Kanalını Adlandır", "name_input", "Yeni kanal adı", discordgo.TextInputShort, "")
	case "ticket_status":
		states := []string{"open", "in_progress", "waiting"}
		next := states[0]
		for x, v := range states {
			if t.Status == v {
				next = states[(x+1)%len(states)]
			}
		}
		_, e = b.db.SetTicketStatus(i.ChannelID, next)
		if e != nil {
			return e
		}
		return ephemeral(s, i, "Ticket durumu: **"+next+"**")
	}
	return nil
}
func ticketModal(s *discordgo.Session, i *discordgo.InteractionCreate, id, title, inputID, label string, style discordgo.TextInputStyle, value string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{CustomID: id, Title: title, Components: []discordgo.MessageComponent{row(discordgo.TextInput{CustomID: inputID, Label: label, Style: style, Required: id != "ticket_close_modal", Value: value, MaxLength: 500})}}})
}
func (b *Bot) ticketAddFromModal(s *discordgo.Session, i *discordgo.InteractionCreate, v map[string]string) error {
	c := &commandContext{b: b, s: s, guildID: i.GuildID, channelID: i.ChannelID, user: userOf(i), member: i.Member}
	u, m, e := b.target(c, v["user_input"])
	if e != nil || m == nil {
		return fmt.Errorf("kullanıcı bulunamadı")
	}
	allow := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory)
	if e = s.ChannelPermissionSet(i.ChannelID, u.ID, discordgo.PermissionOverwriteTypeMember, allow, 0); e != nil {
		return e
	}
	return ephemeral(s, i, "<@"+u.ID+"> ticket kanalına eklendi.")
}
func (b *Bot) ticketRenameFromModal(s *discordgo.Session, i *discordgo.InteractionCreate, v map[string]string) error {
	name := safeChannelName(v["name_input"])
	if len(name) < 2 {
		return fmt.Errorf("geçerli kanal adı belirt")
	}
	if _, e := s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{Name: name}); e != nil {
		return e
	}
	return ephemeral(s, i, "Kanal adı **"+name+"** oldu.")
}
func (b *Bot) closeTicketFromModal(s *discordgo.Session, i *discordgo.InteractionCreate, v map[string]string) error {
	t, e := b.db.Ticket(i.ChannelID)
	if e != nil || t.ClosedAt.Valid {
		return fmt.Errorf("bu kanal açık ticket değil")
	}
	c := &commandContext{b: b, s: s, guildID: i.GuildID, channelID: i.ChannelID, user: userOf(i), member: i.Member}
	if t.OwnerID != userOf(i).ID && !b.isSupport(c) {
		return fmt.Errorf("bu talebi kapatma yetkin yok")
	}
	reason := valueOr(v["reason_input"], "Sebep belirtilmedi")
	ok, e := b.db.CloseTicket(i.ChannelID, userOf(i).ID, reason)
	if e != nil || !ok {
		return fmt.Errorf("ticket zaten kapatılmış")
	}
	msgs, _ := s.ChannelMessages(i.ChannelID, 100, "", "", "")
	var lines []string
	for x := len(msgs) - 1; x >= 0; x-- {
		m := msgs[x]
		lines = append(lines, fmt.Sprintf("[%s] %s (%s): %s", m.Timestamp.Format(time.RFC3339), m.Author.Username, m.Author.ID, valueOr(m.Content, "[embed/boş mesaj]")))
	}
	data := []byte(strings.Join(lines, "\n"))
	b.ticketLog(i.GuildID, fmt.Sprintf("🔒 Ticket kapatıldı: **%s** · sahibi: <@%s> · kapatan: <@%s> · sebep: **%s**", i.ChannelID, t.OwnerID, userOf(i).ID, trunc(reason, 200)), &discordgo.File{Name: "ticket-" + i.ChannelID + ".txt", ContentType: "text/plain; charset=utf-8", Reader: bytes.NewReader(data)})
	_ = ephemeral(s, i, "Ticket kapatıldı; kanal 5 saniye içinde silinecek.")
	_, _ = s.ChannelMessageSendEmbed(i.ChannelID, embed("🔒 Ticket Kapatıldı", "**Sebep:** "+trunc(reason, 500), colorDanger))
	time.AfterFunc(5*time.Second, func() { _, _ = s.ChannelDelete(i.ChannelID) })
	return nil
}
func (b *Bot) ticketLog(guild, content string, file *discordgo.File) {
	settings, e := b.db.Settings(guild)
	if e != nil || !settings.TicketLogChannelID.Valid {
		return
	}
	send := &discordgo.MessageSend{Content: safeText(content)}
	if file != nil {
		send.Files = []*discordgo.File{file}
	}
	_, _ = b.dg.ChannelMessageSendComplex(settings.TicketLogChannelID.String, send)
}

var _ database.Ticket
