package bot

import (
	"strings"
	"testing"
	"time"
)

func TestRequiredWordInitialFallsBackFromSoftG(t *testing.T) {
	if got := requiredWordInitial("ağ"); got != 'a' {
		t.Fatalf("ağ kelimesinden sonra %q istendi; beklenen 'a'", got)
	}
	if got := requiredWordInitial("dağ"); got != 'a' {
		t.Fatalf("dağ kelimesinden sonra %q istendi; beklenen 'a'", got)
	}
	if got := requiredWordInitial("kitap"); got != 'p' {
		t.Fatalf("kitap kelimesinden sonra %q istendi; beklenen 'p'", got)
	}
}

func TestModerationConfirmationEmbeds(t *testing.T) {
	item := confirmation{
		UserID: "123", GuildID: "456", Title: "Kullanıcıyı Yasakla",
		Target: "test (`789`)", Reason: "Kural ihlali", Details: "Mesajlar silinecek.",
		Expires: time.Now().Add(30 * time.Second),
	}
	confirmationEmbed := moderationConfirmationEmbed(item)
	if confirmationEmbed.Title != "⚠️ Onay Gerekiyor" || len(confirmationEmbed.Fields) < 5 {
		t.Fatal("moderasyon onay embedi eksik")
	}
	result := moderationResultEmbed(item, "success", "Başarıyla uygulandı.")
	if result.Color != colorSuccess || !strings.Contains(result.Title, "Uygulandı") {
		t.Fatal("moderasyon sonuç embedi başarı durumunu yansıtmıyor")
	}
}
