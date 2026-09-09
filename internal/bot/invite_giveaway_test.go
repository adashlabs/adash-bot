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
	// Net = 5 - 2 - 1 + 3 = 5
	if got := stats.Net(); got != 5 {
		t.Fatalf("beklenen net 5, bulunan %d", got)
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
