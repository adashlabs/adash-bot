package bot

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) embedComponent(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 3 || parts[2] != userOf(i).ID {
		return fmt.Errorf("bu embed taslağı sana ait değil")
	}
	id, action := parts[2], parts[1]
	b.mu.Lock()
	d := b.drafts[id]
	b.mu.Unlock()
	if d == nil || time.Since(d.Updated) > 15*time.Minute {
		return fmt.Errorf("embed taslağının süresi doldu")
	}

	switch action {
	case "edit", "basic":
		return showEmbedModal(s, i, "basic", id, "Temel içerik", []discordgo.MessageComponent{
			textInput("content", "Mesaj metni (embed dışında)", discordgo.TextInputParagraph, 2000, d.Content),
			textInput("title", "Başlık", discordgo.TextInputShort, 256, d.Title),
			textInput("description", "Açıklama", discordgo.TextInputParagraph, 4000, d.Description),
			textInput("color", "Renk (#5865F2)", discordgo.TextInputShort, 7, valueOr(d.Color, "#5865F2")),
			textInput("url", "Başlık bağlantısı (https://)", discordgo.TextInputShort, 2000, d.URL),
		})
	case "media":
		return showEmbedModal(s, i, "media", id, "Görseller", []discordgo.MessageComponent{
			textInput("image", "Büyük görsel URL'si", discordgo.TextInputShort, 2000, d.Image),
			textInput("thumbnail", "Küçük görsel URL'si", discordgo.TextInputShort, 2000, d.Thumbnail),
		})
	case "details":
		return showEmbedModal(s, i, "details", id, "Yazar ve alt bilgi", []discordgo.MessageComponent{
			textInput("author", "Yazar adı", discordgo.TextInputShort, 256, d.Author),
			textInput("author_icon", "Yazar simgesi URL'si", discordgo.TextInputShort, 2000, d.AuthorIcon),
			textInput("footer", "Alt bilgi", discordgo.TextInputShort, 2048, d.Footer),
			textInput("footer_icon", "Alt bilgi simgesi URL'si", discordgo.TextInputShort, 2000, d.FooterIcon),
			textInput("timestamp", "Tarih gösterilsin mi? (evet/hayır)", discordgo.TextInputShort, 5, str(d.Timestamp, "evet", "hayır")),
		})
	case "field":
		if len(d.Fields) >= 25 {
			return fmt.Errorf("en fazla 25 alan eklenebilir")
		}
		return showEmbedModal(s, i, "field", id, "Yeni alan", []discordgo.MessageComponent{
			textInputRequired("field_name", "Alan adı", discordgo.TextInputShort, 256, ""),
			textInputRequired("field_value", "Alan içeriği", discordgo.TextInputParagraph, 1024, ""),
			textInput("field_inline", "Yan yana gösterilsin mi? (evet/hayır)", discordgo.TextInputShort, 5, "hayır"),
		})
	case "clear_fields":
		b.mu.Lock()
		d.Fields = nil
		d.Updated = time.Now()
		b.mu.Unlock()
		return updateEmbedPanel(s, i, d, id, "Alanlar temizlendi.")
	case "send":
		if err := validateDraft(d); err != nil {
			return err
		}
		msg := &discordgo.MessageSend{Content: d.Content, Embeds: []*discordgo.MessageEmbed{buildDraftEmbed(d)}, AllowedMentions: &discordgo.MessageAllowedMentions{}}
		if _, err := s.ChannelMessageSendComplex(d.ChannelID, msg); err != nil {
			return err
		}
		b.mu.Lock()
		delete(b.drafts, id)
		b.mu.Unlock()
		return ephemeral(s, i, "Embed kanala gönderildi.")
	case "cancel":
		b.mu.Lock()
		delete(b.drafts, id)
		b.mu.Unlock()
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: "Embed taslağı iptal edildi.", Embeds: []*discordgo.MessageEmbed{}, Components: []discordgo.MessageComponent{}}})
	}
	return nil
}

func textInput(id, label string, style discordgo.TextInputStyle, maxLength int, value string) discordgo.MessageComponent {
	return row(discordgo.TextInput{CustomID: id, Label: label, Style: style, Required: false, MaxLength: maxLength, Value: value})
}

func textInputRequired(id, label string, style discordgo.TextInputStyle, maxLength int, value string) discordgo.MessageComponent {
	return row(discordgo.TextInput{CustomID: id, Label: label, Style: style, Required: true, MaxLength: maxLength, Value: value})
}

func showEmbedModal(s *discordgo.Session, i *discordgo.InteractionCreate, kind, id, title string, inputs []discordgo.MessageComponent) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{CustomID: "embed_builder:modal:" + kind + ":" + id, Title: title, Components: inputs}})
}

func embedBuilderControls(id string, fieldCount int) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		row(
			button("embed_builder:basic:"+id, "İçerik", discordgo.PrimaryButton, "✏️"),
			button("embed_builder:media:"+id, "Görseller", discordgo.SecondaryButton, "🖼️"),
			button("embed_builder:details:"+id, "Detaylar", discordgo.SecondaryButton, "⚙️"),
			button("embed_builder:field:"+id, fmt.Sprintf("Alan Ekle (%d/25)", fieldCount), discordgo.SecondaryButton, "➕"),
		),
		row(
			button("embed_builder:clear_fields:"+id, "Alanları Temizle", discordgo.SecondaryButton, "🧹"),
			button("embed_builder:send:"+id, "Kanala Gönder", discordgo.SuccessButton, "✅"),
			button("embed_builder:cancel:"+id, "İptal", discordgo.DangerButton, "✖️"),
		),
	}
}

func parseColor(s string) int {
	v := strings.TrimPrefix(strings.TrimSpace(s), "#")
	n, err := strconv.ParseInt(v, 16, 32)
	if err != nil || len(v) != 6 {
		return colorPrimary
	}
	return int(n)
}

func buildDraftEmbed(d *embedDraft) *discordgo.MessageEmbed {
	em := &discordgo.MessageEmbed{Title: d.Title, Description: d.Description, Color: parseColor(d.Color), Fields: d.Fields}
	if d.Footer != "" || validHTTPURL(d.FooterIcon) {
		em.Footer = &discordgo.MessageEmbedFooter{Text: d.Footer, IconURL: validURLOrEmpty(d.FooterIcon)}
	}
	if validHTTPURL(d.URL) {
		em.URL = d.URL
	}
	if validHTTPURL(d.Image) {
		em.Image = &discordgo.MessageEmbedImage{URL: d.Image}
	}
	if validHTTPURL(d.Thumbnail) {
		em.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: d.Thumbnail}
	}
	if d.Author != "" {
		em.Author = &discordgo.MessageEmbedAuthor{Name: d.Author, IconURL: validURLOrEmpty(d.AuthorIcon)}
	}
	if d.Timestamp {
		em.Timestamp = time.Now().Format(time.RFC3339)
	}
	return em
}

func (b *Bot) saveEmbedModal(s *discordgo.Session, i *discordgo.InteractionCreate, values map[string]string) error {
	parts := strings.Split(i.ModalSubmitData().CustomID, ":")
	if len(parts) != 4 || parts[0] != "embed_builder" || parts[1] != "modal" || parts[3] != userOf(i).ID {
		return fmt.Errorf("bu taslak sana ait değil")
	}
	kind, id := parts[2], parts[3]
	b.mu.Lock()
	d := b.drafts[id]
	if d != nil {
		switch kind {
		case "basic":
			d.Content = strings.TrimSpace(values["content"])
			d.Title = strings.TrimSpace(values["title"])
			d.Description = strings.TrimSpace(values["description"])
			d.Color = strings.TrimSpace(values["color"])
			d.URL = strings.TrimSpace(values["url"])
		case "media":
			d.Image = strings.TrimSpace(values["image"])
			d.Thumbnail = strings.TrimSpace(values["thumbnail"])
		case "details":
			d.Author = strings.TrimSpace(values["author"])
			d.AuthorIcon = strings.TrimSpace(values["author_icon"])
			d.Footer = strings.TrimSpace(values["footer"])
			d.FooterIcon = strings.TrimSpace(values["footer_icon"])
			d.Timestamp = parseYesNo(values["timestamp"])
		case "field":
			if len(d.Fields) < 25 {
				d.Fields = append(d.Fields, &discordgo.MessageEmbedField{Name: strings.TrimSpace(values["field_name"]), Value: strings.TrimSpace(values["field_value"]), Inline: parseYesNo(values["field_inline"])})
			}
		}
		d.Updated = time.Now()
	}
	b.mu.Unlock()
	if d == nil {
		return fmt.Errorf("taslağın süresi doldu")
	}
	if err := validateDraft(d); err != nil {
		return err
	}
	em := previewDraftEmbed(d)
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: previewStatus(d), Embeds: []*discordgo.MessageEmbed{em}, Components: embedBuilderControls(id, len(d.Fields)), Flags: discordgo.MessageFlagsEphemeral, AllowedMentions: &discordgo.MessageAllowedMentions{}}})
}

func updateEmbedPanel(s *discordgo.Session, i *discordgo.InteractionCreate, d *embedDraft, id, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: message + "\n" + previewStatus(d), Embeds: []*discordgo.MessageEmbed{previewDraftEmbed(d)}, Components: embedBuilderControls(id, len(d.Fields)), AllowedMentions: &discordgo.MessageAllowedMentions{}}})
}

func previewDraftEmbed(d *embedDraft) *discordgo.MessageEmbed {
	em := buildDraftEmbed(d)
	if em.Title == "" && em.Description == "" && len(em.Fields) == 0 && em.Image == nil {
		em.Description = "Önizleme burada görünecek. Başlamak için **İçerik** düğmesine bas."
	}
	return em
}

func previewStatus(d *embedDraft) string {
	parts := []string{"**Embed önizlemesi**"}
	if d.Content != "" {
		parts = append(parts, "Mesaj metni: "+trunc(d.Content, 300))
	}
	parts = append(parts, fmt.Sprintf("Alanlar: **%d/25** · Renk: `%s` · Tarih: **%s**", len(d.Fields), valueOr(d.Color, "#5865F2"), str(d.Timestamp, "Açık", "Kapalı")))
	return strings.Join(parts, "\n")
}

func validateDraft(d *embedDraft) error {
	if d.Title == "" && d.Description == "" && len(d.Fields) == 0 && d.Image == "" && d.Author == "" && d.Footer == "" {
		return fmt.Errorf("göndermeden önce mesaj veya embed içeriği ekle")
	}
	if utf8.RuneCountInString(d.Content) > 2000 {
		return fmt.Errorf("mesaj metni 2000 karakteri geçemez")
	}
	colorText := strings.TrimPrefix(strings.TrimSpace(d.Color), "#")
	if colorText != "" {
		if _, err := strconv.ParseUint(colorText, 16, 24); err != nil || len(colorText) != 6 {
			return fmt.Errorf("renk `#5865F2` biçiminde olmalı")
		}
	}
	for label, raw := range map[string]string{"başlık bağlantısı": d.URL, "büyük görsel": d.Image, "küçük görsel": d.Thumbnail, "yazar simgesi": d.AuthorIcon, "alt bilgi simgesi": d.FooterIcon} {
		if raw != "" && !validHTTPURL(raw) {
			return fmt.Errorf("%s için geçerli bir http/https adresi gir", label)
		}
	}
	total := utf8.RuneCountInString(d.Title) + utf8.RuneCountInString(d.Description) + utf8.RuneCountInString(d.Author) + utf8.RuneCountInString(d.Footer)
	for _, field := range d.Fields {
		if field.Name == "" || field.Value == "" {
			return fmt.Errorf("alan adı ve içeriği boş olamaz")
		}
		total += utf8.RuneCountInString(field.Name) + utf8.RuneCountInString(field.Value)
	}
	if total > 6000 {
		return fmt.Errorf("embed toplam 6000 karakter sınırını aşıyor")
	}
	return nil
}

func validHTTPURL(raw string) bool {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func validURLOrEmpty(raw string) string {
	if validHTTPURL(raw) {
		return raw
	}
	return ""
}

func parseYesNo(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "evet", "e", "yes", "true", "1", "açık", "acik":
		return true
	default:
		return false
	}
}
