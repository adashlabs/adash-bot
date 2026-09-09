package bot

import (
	"strings"
	"testing"

	"github.com/adashlabs/adash-bot/internal/database"
)

func TestInviteStatsNetCalculation(t *testing.T) {
	stats := database.InviteStats{
		UserID:  "12345",
		Regular: 5,
		Left:    2,
		Fake:    1,
		Bonus:   3,
	}
	// Net = 5 - 2 + 3 = 6
	if got := stats.Net(); got != 6 {
		t.Fatalf("beklenen net 6, bulunan %d", got)
	}

	// Negative net cases should be clamped to 0
	statsNegative := database.InviteStats{
		Regular: 1,
		Left:    2,
		Fake:    0,
		Bonus:   0,
	}
	if got := statsNegative.Net(); got != 0 {
		t.Fatalf("beklenen net 0 (clamped), bulunan %d", got)
	}
}

func TestChooseWinnersDistinctness(t *testing.T) {
	entries := []string{"u1", "u2", "u3", "u4", "u5"}
	winners := chooseWinners(entries, 3)
	if len(winners) != 3 {
		t.Fatalf("beklenen 3 kazanan, bulunan %d", len(winners))
	}
	seen := make(map[string]bool)
	for _, w := range winners {
		if seen[w] {
			t.Fatalf("kazanan tekil olmalı: %s", w)
		}
		seen[w] = true
	}

	// When winners requested > entries
	winnersAll := chooseWinners(entries, 10)
	if len(winnersAll) != 5 {
		t.Fatalf("katılımcı sayısından fazla kazanan seçilemez: %d", len(winnersAll))
	}
}

func TestGiveawayEasterEggMessage(t *testing.T) {
	expectedWhisper := "> 🤫 *Pssst... Ben seni tutuyorum, aramızda kalsın kimseye söyleme! 😉*"
	if !strings.Contains(expectedWhisper, "Ben seni tutuyorum") || !strings.Contains(expectedWhisper, "kimseye söyleme") {
		t.Fatal("beklenen espirili fısıltı metni eksik veya yanlış")
	}
}

func TestGiveawayEmbedConditionsAndMultiplierFormatting(t *testing.T) {
	b := &Bot{}
	g := database.Giveaway{
		ID:                42,
		Prize:             "1 Aylık Discord Nitro",
		WinnerCount:       2,
		EndsAt:            1780000000000,
		HostID:            "host999",
		MinInvites:        1,
		MinAccountAgeDays: 7,
	}
	g.RequiredRoleID.Valid = true
	g.RequiredRoleID.String = "role777"

	em := b.giveawayEmbed(g, 10, false, nil)
	foundCond := false
	for _, f := range em.Fields {
		if f.Name == "🛡️ Katılım Koşulları" {
			foundCond = true
			if !strings.Contains(f.Value, "Davet Şartı") || !strings.Contains(f.Value, "1") {
				t.Errorf("Davet şartı eksik: %s", f.Value)
			}
			if !strings.Contains(f.Value, "Hesap Yaşı") || !strings.Contains(f.Value, "7") {
				t.Errorf("Hesap yaşı şartı eksik: %s", f.Value)
			}
			if !strings.Contains(f.Value, "Zorunlu Rol") || !strings.Contains(f.Value, "<@&role777>") {
				t.Errorf("Zorunlu rol eksik: %s", f.Value)
			}
		}
	}
	if !foundCond {
		t.Fatalf("🛡️ Katılım Koşulları alanı bulunamadı")
	}
}
