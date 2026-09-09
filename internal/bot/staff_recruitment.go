package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) staffCommand(c *commandContext, args []string) error {
	// Sunucuyu Yönet yetkisi ZORUNLUDUR
	if err := c.require(discordgo.PermissionManageServer); err != nil {
		return fmt.Errorf("bu komutu kullanmak için Sunucuyu Yönet (Manage Server) yetkisine sahip olmalısın")
	}

	// Kanal parametresi kontrolü (a!yetkili #kanal veya a!yetkili log #kanal veya a!yetkili ayarla #kanal)
	var targetChan string
	for _, arg := range args {
		if id := mentionID(arg); id != "" {
			targetChan = id
			break
		}
	}

	if targetChan != "" {
		ch, err := c.s.Channel(targetChan)
		if err != nil || ch.GuildID != c.guildID || (ch.Type != discordgo.ChannelTypeGuildText && ch.Type != discordgo.ChannelTypeGuildNews) {
			return fmt.Errorf("geçerli bir sunucu metin kanalı belirtmelisin")
		}
		if err := b.db.SetConfig(c.guildID, "staff_app_log_channel_id", targetChan); err != nil {
			return err
		}
		return c.embed(successEmbed("✅ Yetkili Başvuru Log Kanalı Ayarlandı",
			fmt.Sprintf("Doldurulan yetkili başvuru formları artık <#%s> kanalına iletilecek.\n\n"+
				"📋 **Başvuru Formunu Göndermek İçin:**\n"+
				"Başvuru formunun bulunmasını istediğiniz kanalda `a!yetkili` yazmanız yeterlidir.", targetChan)))
	}

	// Alt komut: durum kontrolü
	if len(args) > 0 && (strings.ToLower(args[0]) == "durum" || strings.ToLower(args[0]) == "status" || strings.ToLower(args[0]) == "bilgi") {
		logChan := b.db.ConfigString(c.guildID, "staff_app_log_channel_id", "")
		statusVal := "🔴 Ayarlı değil"
		if logChan != "" {
			statusVal = fmt.Sprintf("🟢 <#%s>", logChan)
		}
		em := embed("🛡️ Yetkili Alım Sistemi Durumu",
			fmt.Sprintf("• **Log Kanalı:** %s\n• **Prefix:** `%s`\n• **Komut:** `%syetkili`",
				statusVal, b.db.Prefix(c.guildID), b.db.Prefix(c.guildID)), colorPrimary)
		return c.embed(em)
	}

	// Log kanalı kontrolü
	logChan := b.db.ConfigString(c.guildID, "staff_app_log_channel_id", "")
	if logChan == "" {
		em := embed("⚙️ Yetkili Alım Sistemi Kurulumu",
			"Yetkili başvuru sistemini kullanabilmek için öncelikle doldurulan formların düşeceği **log kanalını** belirlemelisiniz.\n\n"+
				"📌 **Kanalı Ayarlamak İçin:**\n"+
				"• Komutla: `"+b.db.Prefix(c.guildID)+"yetkili #log-kanalı`\n"+
				"• Veya aşağıdaki menüden kanalı seçebilirsiniz.\n\n"+
				"Kanal ayarlandıktan sonra `"+b.db.Prefix(c.guildID)+"yetkili` yazarak başvuru formunu dilediğiniz kanala gönderebilirsiniz.",
			colorPrimary)

		selectRow := row(discordgo.SelectMenu{
			CustomID:     "setup_channel:stafflog:" + c.guildID,
			Placeholder:  "Başvuruların düşeceği log kanalını seçin",
			MenuType:     discordgo.ChannelSelectMenu,
			ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
			MinValues:    intp(1),
			MaxValues:    1,
		})

		return c.embed(em, selectRow)
	}

	// Log kanalı geçerli mi kontrol et
	if _, err := c.s.Channel(logChan); err != nil {
		em := embed("⚠️ Log Kanalı Bulunamadı",
			"Önceden ayarlanmış log kanalına erişilemiyor veya silinmiş.\nLütfen yeni bir kanal belirleyin:\n`"+b.db.Prefix(c.guildID)+"yetkili #yeni-kanal`",
			colorWarning)
		return c.embed(em)
	}

	// Yetkili Başvuru Paneli gönder
	panelEmbed, components := b.staffApplicationPanel(c.guildID)
	return c.embed(panelEmbed, components...)
}

func (b *Bot) staffApplicationPanel(guildID string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	logChan := b.db.ConfigString(guildID, "staff_app_log_channel_id", "")
	logNotice := ""
	if logChan != "" {
		logNotice = fmt.Sprintf("\n\n*Gönderilen formlar güvenli log kanalına iletilir.*")
	}

	em := &discordgo.MessageEmbed{
		Title: "🛡️ Yetkili Başvuru Formu",
		Description: "Sunucumuzun yetkili kadrosuna katılarak topluluğumuzu birlikte büyütüp geliştirmeye hazır mısın?\n\n" +
			"Adil, saygılı, aktif ve takım çalışmasına değer veren yeni takım arkadaşlarımızı aramızda görmekten mutluluk duyarız!\n\n" +
			"📋 **Genel Şartlar & Kriterler:**\n" +
			"• Sunucu kurallarına ve Discord Hizmet Şartlarına hakim olmak\n" +
			"• Günlük düzenli aktiflik sağlayabilmek\n" +
			"• Tarafsız, çözüm odaklı ve saygılı bir iletişim diline sahip olmak\n" +
			"• Başvuru formundaki soruları samimi ve eksiksiz doldurmak\n\n" +
			"👉 Aşağıdaki **Başvur** butonuna tıklayarak başvuru formunu doldurabilirsiniz. Başvurunuz yetkili ekibimize gizli olarak iletilecektir." + logNotice,
		Color: colorPrimary,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Adash Bot • Yetkili Alım Sistemi",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	btn := discordgo.Button{
		CustomID: "staff_apply_btn",
		Label:    "Başvur",
		Style:    discordgo.SuccessButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
	}

	return em, []discordgo.MessageComponent{row(btn)}
}

func (b *Bot) staffApplyModal(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	u := userOf(i)
	if u == nil {
		return fmt.Errorf("kullanıcı bilgisi alınamadı")
	}

	// Aktif bekleyen başvuru kontrolü
	pending, err := b.db.HasPendingStaffApplication(i.GuildID, u.ID)
	if err == nil && pending {
		return ephemeral(s, i, "⚠️ Zaten değerlendirme aşamasında olan aktif bir başvurunuz bulunmaktadır! Lütfen yetkili ekibin mevcut başvurunuzu sonuçlandırmasını bekleyin.")
	}

	// Log kanalı var mı?
	logChan := b.db.ConfigString(i.GuildID, "staff_app_log_channel_id", "")
	if logChan == "" {
		return ephemeral(s, i, "⚠️ Bu sunucuda yetkili başvuru kayıt kanalı henüz yapılandırılmamış. Lütfen sunucu yöneticileriyle iletişime geçin.")
	}

	// 5 TextInput alanı (Discord modal üst limiti)
	inputs := []discordgo.MessageComponent{
		row(discordgo.TextInput{
			CustomID:    "staff_name_age",
			Label:       "İsim ve Yaşınız",
			Placeholder: "Örn: Ahmet, 19",
			Style:       discordgo.TextInputShort,
			Required:    true,
			MinLength:   2,
			MaxLength:   60,
		}),
		row(discordgo.TextInput{
			CustomID:    "staff_activity",
			Label:       "Günlük Aktiflik Süreniz",
			Placeholder: "Örn: Günde ortalama 4-6 saat, akşamları aktifim",
			Style:       discordgo.TextInputShort,
			Required:    true,
			MinLength:   2,
			MaxLength:   100,
		}),
		row(discordgo.TextInput{
			CustomID:    "staff_experience",
			Label:       "Daha Önceki Yetkililik Deneyimleriniz",
			Placeholder: "Daha önce görev aldığınız sunucular, roller ve tecrübeleriniz",
			Style:       discordgo.TextInputParagraph,
			Required:    true,
			MinLength:   3,
			MaxLength:   600,
		}),
		row(discordgo.TextInput{
			CustomID:    "staff_reason",
			Label:       "Neden Yetkili Olmak İstiyorsunuz?",
			Placeholder: "Neden sizi seçmeliyiz? Sunucumuza ne gibi katkılarda bulunabilirsiniz?",
			Style:       discordgo.TextInputParagraph,
			Required:    true,
			MinLength:   10,
			MaxLength:   1000,
		}),
		row(discordgo.TextInput{
			CustomID:    "staff_about",
			Label:       "Kendinizden Bahsedin / Eklemek İstedikleriniz",
			Placeholder: "İlgi alanlarınız, becerileriniz veya iletmek istediğiniz ek notlar (opsiyonel)",
			Style:       discordgo.TextInputParagraph,
			Required:    false,
			MaxLength:   1000,
		}),
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "staff_apply_modal",
			Title:      "Yetkili Başvuru Formu",
			Components: inputs,
		},
	})
}

func (b *Bot) handleStaffModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, v map[string]string) error {
	u := userOf(i)
	if u == nil {
		return fmt.Errorf("kullanıcı bilgisi alınamadı")
	}

	// Mükerrer başvuru engeli
	pending, err := b.db.HasPendingStaffApplication(i.GuildID, u.ID)
	if err == nil && pending {
		return ephemeral(s, i, "⚠️ Zaten inceleme bekleyen bir başvurunuz bulunmaktadır.")
	}

	logChan := b.db.ConfigString(i.GuildID, "staff_app_log_channel_id", "")
	if logChan == "" {
		return ephemeral(s, i, "⚠️ Başvuru log kanalı bulunamadı. Lütfen yöneticilere bildiriniz.")
	}

	nameAge := strings.TrimSpace(v["staff_name_age"])
	activity := strings.TrimSpace(v["staff_activity"])
	experience := strings.TrimSpace(v["staff_experience"])
	reason := strings.TrimSpace(v["staff_reason"])
	about := strings.TrimSpace(v["staff_about"])

	if len(nameAge) < 2 || len(activity) < 2 || len(experience) < 3 || len(reason) < 5 {
		return fmt.Errorf("lütfen formdaki tüm zorunlu alanları eksiksiz doldurun")
	}

	// Veritabanına kaydet
	appID, err := b.db.CreateStaffApplication(i.GuildID, u.ID, nameAge, activity, experience, reason, about)
	if err != nil {
		return fmt.Errorf("başvuru kaydedilirken hata oluştu: %w", err)
	}

	// Log kanalına gönderilecek başvuru kartı
	joinedUnix := int64(0)
	if i.Member != nil && !i.Member.JoinedAt.IsZero() {
		joinedUnix = i.Member.JoinedAt.Unix()
	}
	createdUnix := snowflakeTime(u.ID).Unix()

	historyStr := fmt.Sprintf("• Hesap Açılışı: <t:%d:F> (<t:%d:R>)", createdUnix, createdUnix)
	if joinedUnix > 0 {
		historyStr += fmt.Sprintf("\n• Sunucuya Katılış: <t:%d:F> (<t:%d:R>)", joinedUnix, joinedUnix)
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "👤 Başvuran Kullanıcı",
			Value:  fmt.Sprintf("<@%s> (`%s` / ID: `%s`)", u.ID, u.Username, u.ID),
			Inline: false,
		},
		{
			Name:   "🎂 İsim & Yaş",
			Value:  safeText(nameAge),
			Inline: true,
		},
		{
			Name:   "⏰ Günlük Aktiflik",
			Value:  safeText(activity),
			Inline: true,
		},
		{
			Name:   "📊 Durum",
			Value:  "🟡 **İnceleniyor (Beklemede)**",
			Inline: true,
		},
		{
			Name:   "📜 Yetkililik Deneyimleri",
			Value:  trunc(safeText(experience), 1024),
			Inline: false,
		},
		{
			Name:   "💡 Neden Yetkili Olmak İstiyor?",
			Value:  trunc(safeText(reason), 1024),
			Inline: false,
		},
	}

	if about != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "📝 Ek Bilgiler / Notlar",
			Value:  trunc(safeText(about), 1024),
			Inline: false,
		})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "📅 Kullanıcı Geçmişi",
		Value:  historyStr,
		Inline: false,
	})

	logEmbed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📋 Yeni Yetkili Başvurusu (#%d)", appID),
		Description: fmt.Sprintf("<@%s> tarafından yetkili alım formu dolduruldu.", u.ID),
		Color:       colorPrimary,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: u.AvatarURL("256")},
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Başvuru ID: #%d • Başvuru Tarihi", appID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	actionRow := row(
		button(fmt.Sprintf("staff_app_action:accept:%d:%s", appID, u.ID), "Onayla", discordgo.SuccessButton, "✅"),
		button(fmt.Sprintf("staff_app_action:reject:%d:%s", appID, u.ID), "Reddet", discordgo.DangerButton, "❌"),
	)

	_, sendErr := s.ChannelMessageSendComplex(logChan, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{logEmbed},
		Components: []discordgo.MessageComponent{actionRow},
	})
	if sendErr != nil {
		// Log kanalına mesaj atılamadıysa kullanıcıya bildir
		return fmt.Errorf("başvuru kaydedildi ancak log kanalına iletilemedi (%v). Lütfen yöneticilere bildiriniz", sendErr)
	}

	return ephemeral(s, i, "✅ **Yetkili başvurunuz başarıyla alındı!**\nBaşvurunuz değerlendirilmek üzere yetkili ekibimize iletildi. İlginiz için teşekkür ederiz.")
}

func (b *Bot) handleStaffAction(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Yetki kontrolü: Sadece Sunucuyu Yönet yetkisine sahip yetkililer onaylayabilir/reddedebilir
	p, e := s.UserChannelPermissions(userOf(i).ID, i.ChannelID)
	if e != nil || (p&discordgo.PermissionManageServer == 0 && p&discordgo.PermissionAdministrator == 0) {
		return fmt.Errorf("bu başvuruyu değerlendirmek için Sunucuyu Yönet (Manage Server) yetkisine sahip olmalısın")
	}

	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		return fmt.Errorf("geçersiz işlem parametresi")
	}

	action := parts[1]
	appID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return fmt.Errorf("geçersiz başvuru ID")
	}
	applicantID := parts[3]

	app, err := b.db.StaffApplicationByID(appID)
	if err != nil || app.Status != "pending" {
		return ephemeral(s, i, "⚠️ Bu başvuru zaten daha önce sonuçlandırılmış veya bulunamadı.")
	}

	reviewer := userOf(i)

	if action == "accept" {
		ok, err := b.db.ReviewStaffApplication(appID, "accepted", reviewer.ID, "")
		if err != nil || !ok {
			return ephemeral(s, i, "⚠️ Başvuru güncellenirken bir hata oluştu veya zaten sonuçlandırılmış.")
		}

		// Log mesajındaki embed'i güncelle
		var updatedEmbed *discordgo.MessageEmbed
		if i.Message != nil && len(i.Message.Embeds) > 0 {
			emb := *i.Message.Embeds[0]
			emb.Color = colorSuccess
			// Durum alanını güncelle
			for _, f := range emb.Fields {
				if f.Name == "📊 Durum" {
					f.Value = fmt.Sprintf("🟢 **ONAYLANDI** (Yetkili: <@%s>)", reviewer.ID)
				}
			}
			emb.Fields = append(emb.Fields, &discordgo.MessageEmbedField{
				Name:   "✅ Sonuç: ONAYLANDI",
				Value:  fmt.Sprintf("• **İnceleyen Yetkili:** <@%s> (`%s`)\n• **Karar Tarihi:** <t:%d:F> (<t:%d:R>)", reviewer.ID, reviewer.Username, time.Now().Unix(), time.Now().Unix()),
				Inline: false,
			})
			updatedEmbed = &emb
		}

		disabledBtn := discordgo.Button{
			CustomID: "staff_action_done",
			Label:    fmt.Sprintf("Onaylandı (@%s)", reviewer.Username),
			Style:    discordgo.SuccessButton,
			Disabled: true,
			Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
		}

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{updatedEmbed},
				Components: []discordgo.MessageComponent{row(disabledBtn)},
			},
		})

		// Kullanıcıya DM bildirimi
		go b.sendStaffDecisionDM(s, i.GuildID, applicantID, true, "")

		return nil
	}

	if action == "reject" {
		// Reddetme gerekçesi sormak için modal aç
		modal := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("staff_app_reject_modal:%d:%s", appID, applicantID),
				Title:    "Yetkili Başvurusunu Reddet",
				Components: []discordgo.MessageComponent{
					row(discordgo.TextInput{
						CustomID:    "staff_reject_reason",
						Label:       "Reddedilme Sebebi (Adaya iletilecektir)",
						Placeholder: "Başvurunuz kriterlerimizi şu an için karşılamamaktadır.",
						Style:       discordgo.TextInputParagraph,
						Required:    true,
						Value:       "Başvurunuz kriterlerimizi şu an için karşılamamaktadır.",
						MaxLength:   500,
					}),
				},
			},
		}
		return s.InteractionRespond(i.Interaction, modal)
	}

	return nil
}

func (b *Bot) handleStaffRejectModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, v map[string]string) error {
	p, e := s.UserChannelPermissions(userOf(i).ID, i.ChannelID)
	if e != nil || (p&discordgo.PermissionManageServer == 0 && p&discordgo.PermissionAdministrator == 0) {
		return fmt.Errorf("bu başvuruyu değerlendirmek için Sunucuyu Yönet yetkisine sahip olmalısın")
	}

	customID := i.ModalSubmitData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return fmt.Errorf("geçersiz modal parametresi")
	}

	appID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("geçersiz başvuru ID")
	}
	applicantID := parts[2]
	reason := strings.TrimSpace(v["staff_reject_reason"])
	if reason == "" {
		reason = "Belirtilmedi."
	}

	reviewer := userOf(i)

	ok, err := b.db.ReviewStaffApplication(appID, "rejected", reviewer.ID, reason)
	if err != nil || !ok {
		return ephemeral(s, i, "⚠️ Başvuru güncellenirken bir hata oluştu veya zaten sonuçlandırılmış.")
	}

	// Modal'a yanıt ver
	_ = ephemeral(s, i, fmt.Sprintf("❌ #%d numaralı başvuru reddedildi.", appID))

	// Log mesajını güncelle
	if i.Message != nil && len(i.Message.Embeds) > 0 {
		emb := *i.Message.Embeds[0]
		emb.Color = colorDanger
		for _, f := range emb.Fields {
			if f.Name == "📊 Durum" {
				f.Value = fmt.Sprintf("🔴 **REDDEDİLDİ** (Yetkili: <@%s>)", reviewer.ID)
			}
		}
		emb.Fields = append(emb.Fields, &discordgo.MessageEmbedField{
			Name:   "❌ Sonuç: REDDEDİLDİ",
			Value:  fmt.Sprintf("• **İnceleyen Yetkili:** <@%s> (`%s`)\n• **Sebep:** %s\n• **Karar Tarihi:** <t:%d:F> (<t:%d:R>)", reviewer.ID, reviewer.Username, safeText(reason), time.Now().Unix(), time.Now().Unix()),
			Inline: false,
		})

		disabledBtn := discordgo.Button{
			CustomID: "staff_action_done",
			Label:    fmt.Sprintf("Reddedildi (@%s)", reviewer.Username),
			Style:    discordgo.DangerButton,
			Disabled: true,
			Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
		}

		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    i.ChannelID,
			ID:         i.Message.ID,
			Embeds:     &[]*discordgo.MessageEmbed{&emb},
			Components: &[]discordgo.MessageComponent{row(disabledBtn)},
		})
	}

	// Kullanıcıya DM bildirimi
	go b.sendStaffDecisionDM(s, i.GuildID, applicantID, false, reason)

	return nil
}

func (b *Bot) sendStaffDecisionDM(s *discordgo.Session, guildID, userID string, accepted bool, reason string) {
	dmChan, err := s.UserChannelCreate(userID)
	if err != nil {
		return
	}

	guild, _ := s.Guild(guildID)
	guildName := "Sunucu"
	if guild != nil {
		guildName = guild.Name
	}

	if accepted {
		dmEmbed := &discordgo.MessageEmbed{
			Title: "🎉 Yetkili Başvurunuz Onaylandı!",
			Description: fmt.Sprintf("Tebrikler! **%s** sunucusundaki yetkili başvurunuz yetkili ekibi tarafından **onaylandı**.\n\n"+
				"Yetkili ekibimiz en kısa sürede rolünüzü tanımlayacak ve sizinle iletişime geçecektir.", guildName),
			Color:     colorSuccess,
			Timestamp: time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: guildName + " Yetkili Ekibi",
			},
		}
		_, _ = s.ChannelMessageSendEmbed(dmChan.ID, dmEmbed)
	} else {
		dmEmbed := &discordgo.MessageEmbed{
			Title: "Yetkili Başvurunuz Hakkında",
			Description: fmt.Sprintf("**%s** sunucusundaki yetkili başvurunuz değerlendirilmiş ve şu an için **uygun görülmemiştir**.\n\n"+
				"📌 **Açıklama / Sebep:**\n%s\n\n"+
				"Topluluğumuzda aktif kalmaya devam edebilir, ilerleyen yetkili alımlarında şansınızı tekrar deneyebilirsiniz.", guildName, safeText(reason)),
			Color:     colorDanger,
			Timestamp: time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: guildName + " Yetkili Ekibi",
			},
		}
		_, _ = s.ChannelMessageSendEmbed(dmChan.ID, dmEmbed)
	}
}
