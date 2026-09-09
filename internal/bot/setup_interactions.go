package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) setupComponent(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	p, e := s.UserChannelPermissions(userOf(i).ID, i.ChannelID)
	if e != nil || p&discordgo.PermissionManageServer == 0 && p&discordgo.PermissionAdministrator == 0 {
		return fmt.Errorf("bu panel için Sunucuyu Yönet yetkisi gerekiyor")
	}
	id := i.MessageComponentData().CustomID
	values := i.MessageComponentData().Values
	if strings.HasPrefix(id, "setup_section:") {
		section := values[0]
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{b.setupEmbed(i.GuildID, section)}, Components: setupComponents(i.GuildID, section)}})
	}
	if strings.HasPrefix(id, "setup_edit_messages:") {
		return b.setupModal(s, i, "messages")
	}
	if strings.HasPrefix(id, "setup_ticket_texts:") {
		return b.setupModal(s, i, "ticket")
	}
	if strings.HasPrefix(id, "setup_ai_prompt:") {
		return b.setupModal(s, i, "ai")
	}
	if strings.HasPrefix(id, "setup_giveaway_rules:") {
		return b.setupModal(s, i, "giveaway")
	}
	if strings.HasPrefix(id, "setup_giveaway_mult:") {
		return b.setupModal(s, i, "giveaway_mult")
	}
	if strings.HasPrefix(id, "setup_giveaway_create_btn:") {
		return b.setupModal(s, i, "create")
	}
	if strings.HasPrefix(id, "ticketsetup_panelchan:") && len(values) > 0 {
		if e := b.db.SetConfig(i.GuildID, "ticket_panel_channel_id", values[0]); e != nil {
			return e
		}
		return ephemeral(s, i, "Ticket panel kanalı kaydedildi.")
	}
	if strings.HasPrefix(id, "ticketsetup_deploy:") {
		panel := b.db.ConfigString(i.GuildID, "ticket_panel_channel_id", i.ChannelID)
		if e := b.sendTicketPanel(i.GuildID, panel); e != nil {
			return e
		}
		return ephemeral(s, i, "Ticket paneli <#"+panel+"> kanalına gönderildi.")
	}
	if strings.HasPrefix(id, "setup_clear:") {
		subject := strings.Split(id, ":")[1]
		switch subject {
		case "ticketsupport":
			e = b.db.SetConfig(i.GuildID, "ticket_support_role_id", "")
		case "giveawayrole":
			e = b.db.SetConfig(i.GuildID, "giveaway_required_role_id", "")
			b.refreshGiveawayWizardIfApplicable(s, i)
			return ephemeral(s, i, "🛡️ Çekiliş zorunlu katılım rolü kaldırıldı.")
		case "giveawaymult":
			_ = b.db.SetConfig(i.GuildID, "giveaway_multiplier_role_id", "")
			_ = b.db.SetConfig(i.GuildID, "giveaway_multiplier_value", 1)
			b.refreshGiveawayWizardIfApplicable(s, i)
			return ephemeral(s, i, "🚀 Çekiliş çarpan rolü ve katsayısı kaldırıldı.")
		case "giveawayall":
			_ = b.db.SetConfig(i.GuildID, "giveaway_multiplier_role_id", "")
			_ = b.db.SetConfig(i.GuildID, "giveaway_multiplier_value", 1)
			_ = b.db.SetConfig(i.GuildID, "giveaway_required_role_id", "")
			_ = b.db.SetConfig(i.GuildID, "giveaway_min_account_age_days", 0)
			_ = b.db.SetConfig(i.GuildID, "giveaway_min_invites", 0)
			b.refreshGiveawayWizardIfApplicable(s, i)
			return ephemeral(s, i, "🧹 Çekiliş katılım şartları ve çarpan ayarlarının tamamı sıfırlandı.")
		case "aichannel":
			e = b.db.SetSetting(i.GuildID, "ai_channel_id", nil)
		}
		if e != nil {
			return e
		}
		return ephemeral(s, i, "Ayar temizlendi.")
	}
	parts := strings.Split(id, ":")
	if len(parts) < 3 {
		return nil
	}
	kind, subject := parts[0], parts[1]
	if kind == "setup_channel" && len(values) > 0 {
		val := values[0]
		settings := map[string]string{"welcome": "welcome_channel_id", "farewell": "farewell_channel_id", "counting": "counting_channel_id", "word": "word_chain_channel_id", "ticketcategory": "ticket_category_id", "ticketlog": "ticket_log_channel_id", "ai": "ai_channel_id", "modlog": "mod_log_channel_id"}
		if col := settings[subject]; col != "" {
			e = b.db.SetSetting(i.GuildID, col, val)
		} else {
			key := map[string]string{"giveawaylog": "giveaway_log_channel_id", "appeal": "appeal_channel_id", "stafflog": "staff_app_log_channel_id"}[subject]
			if key != "" {
				e = b.db.SetConfig(i.GuildID, key, val)
			}
		}
		if e != nil {
			return e
		}
		return ephemeral(s, i, "Kanal ayarı kaydedildi.")
	}
	if kind == "setup_role" && len(values) > 0 {
		if subject == "autorole" {
			e = b.db.SetSetting(i.GuildID, "autorole_id", values[0])
		} else {
			key := map[string]string{
				"ticketsupport":    "ticket_support_role_id",
				"giveawayrole":     "giveaway_required_role_id",
				"giveawaymultrole": "giveaway_multiplier_role_id",
			}[subject]
			if key != "" {
				e = b.db.SetConfig(i.GuildID, key, values[0])
			}
			if subject == "giveawaymultrole" {
				if b.db.ConfigInt(i.GuildID, "giveaway_multiplier_value", 1) < 2 {
					_ = b.db.SetConfig(i.GuildID, "giveaway_multiplier_value", 2)
				}
				mult := b.db.ConfigInt(i.GuildID, "giveaway_multiplier_value", 2)
				b.refreshGiveawayWizardIfApplicable(s, i)
				return ephemeral(s, i, fmt.Sprintf("🚀 Çekiliş çarpan rolü <@&%s> olarak ayarlandı! Çarpan katsayısı: **%dx**", values[0], mult))
			}
			if subject == "giveawayrole" {
				b.refreshGiveawayWizardIfApplicable(s, i)
				return ephemeral(s, i, fmt.Sprintf("🛡️ Çekiliş zorunlu katılım rolü <@&%s> olarak ayarlandı!", values[0]))
			}
		}
		if e != nil {
			return e
		}
		return ephemeral(s, i, "Rol ayarı kaydedildi.")
	}
	if kind == "setup_toggle" {
		if subject == "ai" {
			e = b.db.SetConfig(i.GuildID, "ai_enabled", !b.db.ConfigBool(i.GuildID, "ai_enabled", true))
		} else {
			s0, x := b.db.Settings(i.GuildID)
			if x != nil {
				return x
			}
			var col string
			var v bool
			switch subject {
			case "welcome":
				col = "welcome_enabled"
				v = !s0.WelcomeEnabled
			case "farewell":
				col = "farewell_enabled"
				v = !s0.FarewellEnabled
			case "counting":
				col = "counting_enabled"
				v = !s0.CountingEnabled
			case "word":
				col = "word_chain_enabled"
				v = !s0.WordChainEnabled
			}
			e = b.db.SetSetting(i.GuildID, col, v)
		}
		if e != nil {
			return e
		}
		return ephemeral(s, i, "Durum değiştirildi.")
	}
	if kind == "setup_reset" {
		if e = b.db.ResetGame(i.GuildID, subject); e != nil {
			return e
		}
		return ephemeral(s, i, "Oyun durumu sıfırlandı.")
	}
	return nil
}
func ephemeral(s *discordgo.Session, i *discordgo.InteractionCreate, text string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: text, Flags: discordgo.MessageFlagsEphemeral}})
}
func ephemeralEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, em *discordgo.MessageEmbed) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{em}, Flags: discordgo.MessageFlagsEphemeral}})
}
func (b *Bot) modalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	id := i.ModalSubmitData().CustomID
	v := modalValues(i)
	switch {
	case strings.HasPrefix(id, "setup_messages:"):
		_ = b.db.SetSetting(i.GuildID, "welcome_message", v["welcome_message"])
		_ = b.db.SetSetting(i.GuildID, "farewell_message", v["farewell_message"])
		return ephemeral(s, i, "Karşılama mesajları kaydedildi.")
	case strings.HasPrefix(id, "setup_ticket_modal:"):
		for _, k := range []string{"ticket_panel_title", "ticket_panel_description", "ticket_welcome_message", "ticket_panel_button_label", "ticket_panel_button_emoji"} {
			_ = b.db.SetConfig(i.GuildID, k, v[k])
		}
		return ephemeral(s, i, "Ticket metinleri kaydedildi.")
	case strings.HasPrefix(id, "setup_ai_prompt_modal:"):
		_ = b.db.SetConfig(i.GuildID, "ai_system_prompt", v["ai_system_prompt"])
		return ephemeral(s, i, "AI sistem promptu kaydedildi.")
	case strings.HasPrefix(id, "setup_giveaway_modal:"):
		n, e := strconv.Atoi(v["min_account_age_days"])
		if e != nil || n < 0 || n > 365 {
			return fmt.Errorf("hesap yaşı 0–365 olmalı")
		}
		_ = b.db.SetConfig(i.GuildID, "giveaway_min_account_age_days", n)
		if invStr, ok := v["min_invites"]; ok && invStr != "" {
			invN, err := strconv.Atoi(invStr)
			if err == nil && invN >= 0 && invN <= 1000 {
				_ = b.db.SetConfig(i.GuildID, "giveaway_min_invites", invN)
			}
		}
		b.refreshGiveawayWizardIfApplicable(s, i)
		return ephemeral(s, i, "Çekiliş kuralları kaydedildi.")
	case strings.HasPrefix(id, "setup_giveaway_mult_modal:"):
		m, err := strconv.Atoi(v["multiplier"])
		if err != nil || m < 2 || m > 100 {
			return fmt.Errorf("çarpan 2–100 arasında bir tam sayı olmalı (örn: 2)")
		}
		_ = b.db.SetConfig(i.GuildID, "giveaway_multiplier_value", m)
		roleID := b.db.ConfigString(i.GuildID, "giveaway_multiplier_role_id", "")
		roleText := ""
		if roleID != "" {
			roleText = fmt.Sprintf(" (<@&%s> rolü için)", roleID)
		}
		b.refreshGiveawayWizardIfApplicable(s, i)
		return ephemeral(s, i, fmt.Sprintf("🚀 Çekiliş kazanma şansı çarpanı **%dx** olarak ayarlandı%s.", m, roleText))
	case id == "setup_giveaway_create_modal":
		d, e := parseDuration(v["duration"])
		if e != nil || d < 10*time.Second {
			return fmt.Errorf("geçerli bir süre gir")
		}
		n, e := strconv.Atoi(v["winners"])
		if e != nil || n < 1 || n > 20 {
			return fmt.Errorf("kazanan sayısı 1–20 olmalı")
		}
		prize := v["prize"]
		if sInv, ok := v["min_invites"]; ok && strings.TrimSpace(sInv) != "" {
			prize += " --davet=" + strings.TrimSpace(sInv)
		}
		if sDays, ok := v["min_days"]; ok && strings.TrimSpace(sDays) != "" {
			prize += " --yas=" + strings.TrimSpace(sDays)
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource})
		c := &commandContext{b: b, s: s, guildID: i.GuildID, channelID: i.ChannelID, user: userOf(i), member: i.Member, interaction: i}
		return b.createGiveaway(c, d, n, prize)
	case strings.HasPrefix(id, "ticket_open_modal"):
		return b.openTicketFromModal(s, i, v)
	case strings.HasPrefix(id, "ticket_add_modal"):
		return b.ticketAddFromModal(s, i, v)
	case strings.HasPrefix(id, "ticket_rename_modal"):
		return b.ticketRenameFromModal(s, i, v)
	case strings.HasPrefix(id, "ticket_close_modal"):
		return b.closeTicketFromModal(s, i, v)
	case strings.HasPrefix(id, "embed_builder:modal:"):
		return b.saveEmbedModal(s, i, v)
	case id == "staff_apply_modal":
		return b.handleStaffModalSubmit(s, i, v)
	case strings.HasPrefix(id, "staff_app_reject_modal:"):
		return b.handleStaffRejectModalSubmit(s, i, v)
	}
	return nil
}
