package bot

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/adashlabs/adash-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

func TestRenderMemberCard(t *testing.T) {
	avatar := image.NewRGBA(image.Rect(0, 0, 420, 240))
	draw.Draw(avatar, avatar.Bounds(), &image.Uniform{C: color.RGBA{80, 120, 220, 255}}, image.Point{}, draw.Src)
	data, err := renderMemberCard(avatar, "Çağrı İğde", "Yazılım Topluluğu", 1284, true)
	if err != nil {
		t.Fatalf("kart oluşturulamadı: %v", err)
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("geçersiz PNG: %v", err)
	}
	if got, want := decoded.Bounds(), image.Rect(0, 0, 1200, 420); got != want {
		t.Fatalf("kart boyutu %v, beklenen %v", got, want)
	}
}

func TestEmbedDraftSupportsProfessionalOptions(t *testing.T) {
	draft := &embedDraft{
		Title:       "Duyuru",
		Description: "Açıklama",
		Color:       "#23D18B",
		URL:         "https://example.com/duyuru",
		Image:       "https://example.com/image.png",
		Thumbnail:   "https://example.com/thumb.png",
		Author:      "Adash",
		AuthorIcon:  "https://example.com/author.png",
		Footer:      "Bilgi",
		FooterIcon:  "https://example.com/footer.png",
		Fields:      []*discordgo.MessageEmbedField{{Name: "Durum", Value: "Aktif", Inline: true}},
		Timestamp:   true,
	}
	if err := validateDraft(draft); err != nil {
		t.Fatalf("geçerli taslak reddedildi: %v", err)
	}
	em := buildDraftEmbed(draft)
	if em.Author == nil || em.Footer == nil || em.Image == nil || em.Thumbnail == nil || len(em.Fields) != 1 || em.Timestamp == "" {
		t.Fatal("embed seçeneklerinin tamamı çıktıya aktarılmadı")
	}
	if err := validateDraft(&embedDraft{Title: "Test", Color: "#5865F2", Image: "javascript:alert(1)"}); err == nil {
		t.Fatal("güvensiz görsel adresi kabul edildi")
	}
}

func TestGiveawayEmbedIsClean(t *testing.T) {
	g := database.Giveaway{ID: 42, HostID: "123", Prize: "Nitro", WinnerCount: 1, EndsAt: time.Now().Add(time.Hour).UnixMilli()}
	em := (&Bot{}).giveawayEmbed(g, 7, false, nil)
	if em.Title != "🎉 Çekiliş" || strings.Contains(em.Title, "Gelişmiş") {
		t.Fatalf("beklenmeyen çekiliş başlığı: %q", em.Title)
	}
	for _, field := range em.Fields {
		if strings.Contains(field.Value, "Yaklaşık") || strings.Contains(field.Name, "Şans") {
			t.Fatalf("gereksiz olasılık metni kaldı: %s", field.Value)
		}
	}
}
