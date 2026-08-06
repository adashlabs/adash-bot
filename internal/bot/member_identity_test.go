package bot

import (
	"strings"
	"testing"
)

func TestRenderGreetingTemplateUsesCurrentMemberID(t *testing.T) {
	const currentID = "123456789012345678"
	template := "<@999999999999999999> {user} {member} ${user} · {username} · {server} · {memberCount}"
	got := renderGreetingTemplate(template, currentID, "Çağrı", "Adash Labs", 42)
	if strings.Contains(got, "999999999999999999") {
		t.Fatal("eski sabit kullanıcı ID'si şablonda kaldı")
	}
	if count := strings.Count(got, "<@"+currentID+">"); count != 4 {
		t.Fatalf("güncel kullanıcı etiketi %d kez üretildi; beklenen 4", count)
	}
	for _, want := range []string{"Çağrı", "Adash Labs", "42"} {
		if !strings.Contains(got, want) {
			t.Fatalf("şablon çıktısında %q bulunamadı: %s", want, got)
		}
	}
}

func TestValidDiscordID(t *testing.T) {
	if !validDiscordID("123456789012345678") {
		t.Fatal("geçerli Discord kimliği reddedildi")
	}
	for _, invalid := range []string{"", "123", "abc123", "123456789012345678901"} {
		if validDiscordID(invalid) {
			t.Fatalf("geçersiz Discord kimliği kabul edildi: %q", invalid)
		}
	}
}
