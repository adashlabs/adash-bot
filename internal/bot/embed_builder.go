package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) embedComponent(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 3 || parts[2] != userOf(i).ID {
		return fmt.Errorf("bu embed taslağı sana ait değil")
	}
	b.mu.Lock()
	d := b.drafts[parts[2]]
	b.mu.Unlock()
	if d == nil || time.Since(d.Updated) > 15*time.Minute {
		return fmt.Errorf("embed taslağının süresi doldu")
	}
	switch parts[1] {
	case "edit":
		inputs := []discordgo.MessageComponent{row(discordgo.TextInput{CustomID: "title", Label: "Başlık", Style: discordgo.TextInputShort, Required: false, MaxLength: 256, Value: d.Title}), row(discordgo.TextInput{CustomID: "description", Label: "Açıklama", Style: discordgo.TextInputParagraph, Required: false, MaxLength: 4000, Value: d.Description}), row(discordgo.TextInput{CustomID: "color", Label: "Renk (#5865F2)", Style: discordgo.TextInputShort, Required: false, MaxLength: 7, Value: valueOr(d.Color, "#5865F2")}), row(discordgo.TextInput{CustomID: "footer", Label: "Alt bilgi", Style: discordgo.TextInputShort, Required: false, MaxLength: 2048, Value: d.Footer}), row(discordgo.TextInput{CustomID: "extras", Label: "URL / Görsel / Küçük görsel / Yazar", Style: discordgo.TextInputParagraph, Required: false, MaxLength: 4000, Value: strings.Join([]string{d.URL, d.Image, d.Thumbnail, d.Author}, "\n")})}
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{CustomID: "embed_builder:modal:" + parts[2], Title: "Embed Builder", Components: inputs}})
	case "send":
		if d.Title == "" && d.Description == "" {
			return fmt.Errorf("başlık veya açıklama ekle")
		}
		if _, e := s.ChannelMessageSendEmbed(d.ChannelID, buildDraftEmbed(d)); e != nil {
			return e
		}
		b.mu.Lock()
		delete(b.drafts, parts[2])
		b.mu.Unlock()
		return ephemeral(s, i, "Embed kanala gönderildi.")
	case "cancel":
		b.mu.Lock()
		delete(b.drafts, parts[2])
		b.mu.Unlock()
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: "Embed taslağı iptal edildi.", Components: []discordgo.MessageComponent{}}})
	}
	return nil
}
func parseColor(s string) int {
	v := strings.TrimPrefix(strings.TrimSpace(s), "#")
	n, e := strconv.ParseInt(v, 16, 32)
	if e != nil {
		return colorPrimary
	}
	return int(n)
}
func buildDraftEmbed(d *embedDraft) *discordgo.MessageEmbed {
	em := embed(d.Title, d.Description, parseColor(d.Color))
	if d.Footer != "" {
		em.Footer = &discordgo.MessageEmbedFooter{Text: d.Footer}
	}
	if strings.HasPrefix(d.URL, "http") {
		em.URL = d.URL
	}
	if strings.HasPrefix(d.Image, "http") {
		em.Image = &discordgo.MessageEmbedImage{URL: d.Image}
	}
	if strings.HasPrefix(d.Thumbnail, "http") {
		em.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: d.Thumbnail}
	}
	if d.Author != "" {
		em.Author = &discordgo.MessageEmbedAuthor{Name: d.Author}
	}
	return em
}
func (b *Bot) saveEmbedModal(s *discordgo.Session, i *discordgo.InteractionCreate, v map[string]string) error {
	id := strings.TrimPrefix(i.ModalSubmitData().CustomID, "embed_builder:modal:")
	if id != userOf(i).ID {
		return fmt.Errorf("bu taslak sana ait değil")
	}
	b.mu.Lock()
	d := b.drafts[id]
	if d != nil {
		d.Title = v["title"]
		d.Description = v["description"]
		d.Color = v["color"]
		d.Footer = v["footer"]
		extra := strings.Split(v["extras"], "\n")
		if len(extra) > 0 {
			d.URL = strings.TrimSpace(extra[0])
		}
		if len(extra) > 1 {
			d.Image = strings.TrimSpace(extra[1])
		}
		if len(extra) > 2 {
			d.Thumbnail = strings.TrimSpace(extra[2])
		}
		if len(extra) > 3 {
			d.Author = strings.TrimSpace(extra[3])
		}
		d.Updated = time.Now()
	}
	b.mu.Unlock()
	if d == nil {
		return fmt.Errorf("taslağın süresi doldu")
	}
	em := buildDraftEmbed(d)
	if em.Title == "" && em.Description == "" {
		em.Description = "Önizleme için başlık veya açıklama ekle."
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "Embed önizlemesi hazır.", Embeds: []*discordgo.MessageEmbed{em}, Components: []discordgo.MessageComponent{row(button("embed_builder:edit:"+id, "İçeriği Düzenle", discordgo.PrimaryButton, "✏️"), button("embed_builder:send:"+id, "Kanala Gönder", discordgo.SuccessButton, "✅"), button("embed_builder:cancel:"+id, "İptal", discordgo.DangerButton, "✖️"))}, Flags: discordgo.MessageFlagsEphemeral}})
}
