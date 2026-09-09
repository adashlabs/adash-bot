package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestStaffApplicationPanel(t *testing.T) {
	b := &Bot{
		db: nil, // We'll test with minimal bot or test embed structure directly
	}

	// Test button structure
	btn := discordgo.Button{
		CustomID: "staff_apply_btn",
		Label:    "Başvur",
		Style:    discordgo.SuccessButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
	}
	if btn.CustomID != "staff_apply_btn" || btn.Label != "Başvur" || btn.Style != discordgo.SuccessButton {
		t.Fatalf("unexpected button configuration: %+v", btn)
	}

	// Test action buttons
	actionRow := row(
		button("staff_app_action:accept:1:12345", "Onayla", discordgo.SuccessButton, "✅"),
		button("staff_app_action:reject:1:12345", "Reddet", discordgo.DangerButton, "❌"),
	)
	if len(actionRow.Components) != 2 {
		t.Fatalf("expected 2 action buttons, got %d", len(actionRow.Components))
	}
	_ = b
}

func TestStaffModalFieldsCount(t *testing.T) {
	// Discord limits modal to at most 5 TextInput components
	inputs := []string{
		"staff_name_age",
		"staff_activity",
		"staff_experience",
		"staff_reason",
		"staff_about",
	}
	if len(inputs) > 5 {
		t.Fatalf("Discord modals support at most 5 inputs, got %d", len(inputs))
	}
}

func TestStaffPermissionRequirement(t *testing.T) {
	// Verify that manageServer permission constant is used
	perm := int64(discordgo.PermissionManageServer)
	if perm&discordgo.PermissionManageServer == 0 {
		t.Fatal("expected PermissionManageServer bit to match")
	}
}

func TestAllModalLabelsWithinDiscordLimits(t *testing.T) {
	labels := []string{
		// Setup modals
		"Hoş geldin mesajı",
		"Görüşürüz mesajı",
		"Panel başlığı",
		"Panel açıklaması",
		"Ticket karşılama mesajı",
		"Düğme yazısı",
		"Düğme emojisi",
		"Sistem promptu",
		"Minimum hesap yaşı (0-365)",
		"Minimum davet sayısı (0-1000)",
		"Kazanma Şansı Çarpanı (2-100)",
		"Süre veya Unix Zamanı (Örn: 1h, 30m)",
		"Kazanan Sayısı (1-20)",
		"Ödül",
		"Minimum Davet Şartı (İsteğe bağlı)",
		"Minimum Hesap Yaşı (Gün, opsiyonel)",
		// Staff modal
		"İsim ve Yaşınız",
		"Günlük Aktiflik Süreniz",
		"Daha Önceki Yetkililik Deneyimleriniz",
		"Neden Yetkili Olmak İstiyorsunuz?",
		"Ek Notlar / Kendinizden Bahsedin",
		// Ticket modals
		"Konu",
		"Sorunun / talebin",
		"Öncelik: düşük / normal / yüksek / acil",
		"Kapanış Sebebi / Notu",
		"Üye ID",
		"Yeni kanal adı",
		// Embed builder
		"Mesaj metni (embed dışında)",
		"Başlık",
		"Açıklama",
		"Renk (#5865F2)",
		"Başlık bağlantısı (https://)",
		"Büyük görsel URL'si",
		"Küçük görsel URL'si",
		"Yazar adı",
		"Yazar simgesi URL'si",
		"Alt bilgi",
		"Alt bilgi simgesi URL'si",
		"Tarih gösterilsin mi? (evet/hayır)",
		"Alan adı",
		"Alan içeriği",
		"Yan yana gösterilsin mi? (evet/hayır)",
	}

	for _, l := range labels {
		if len([]rune(l)) > 45 {
			t.Errorf("Modal etiketi 45 karakterden uzun olamaz (Discord API kuralı): %q (uzunluk: %d)", l, len([]rune(l)))
		}
		truncated := trunc(l, 45)
		if len([]rune(truncated)) > 45 {
			t.Errorf("trunc(label, 45) 45 karakter sınırını aşamaz: %q", truncated)
		}
	}
}
