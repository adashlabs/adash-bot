package bot

import (
	"testing"

	"github.com/adashlabs/adash-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

func TestGiveawayChance(t *testing.T) {
	tests := []struct {
		entries, winners int
		want             string
	}{{10, 1, "%10.0"}, {3, 2, "%66.7"}, {1, 5, "%100.0"}, {0, 1, "%0.0"}}
	for _, test := range tests {
		if got := giveawayChance(test.entries, test.winners); got != test.want {
			t.Fatalf("giveawayChance(%d, %d) = %s; beklenen %s", test.entries, test.winners, got, test.want)
		}
	}
}

func TestGiveawayChanceDetail(t *testing.T) {
	if got := giveawayChanceDetail(10, 1); got != "%10.0 (Yaklaşık 1 / 10)" {
		t.Fatalf("unexpected chance detail: %s", got)
	}
	if got := giveawayChanceDetail(0, 1); got != "%0.0 (1 / 0)" {
		t.Fatalf("unexpected zero chance: %s", got)
	}
}

func TestGiveawayButtons(t *testing.T) {
	g := database.Giveaway{ID: 99}
	activeBtns := giveawayButtons(g, 5, false)
	if len(activeBtns) != 1 {
		t.Fatalf("expected 1 action row, got %d", len(activeBtns))
	}
	row, ok := activeBtns[0].(discordgo.ActionsRow)
	if !ok || len(row.Components) != 3 {
		t.Fatalf("expected 3 buttons in row, got %d", len(row.Components))
	}

	endedBtns := giveawayButtons(g, 5, true)
	rowEnded, ok := endedBtns[0].(discordgo.ActionsRow)
	if !ok || len(rowEnded.Components) != 3 {
		t.Fatalf("expected 3 buttons in ended row, got %d", len(rowEnded.Components))
	}
}

func TestEmbedBuilderAllowsManagerMentions(t *testing.T) {
	allowed := embedBuilderAllowedMentions()
	want := map[discordgo.AllowedMentionType]bool{
		discordgo.AllowedMentionTypeUsers:    true,
		discordgo.AllowedMentionTypeRoles:    true,
		discordgo.AllowedMentionTypeEveryone: true,
	}
	for _, mentionType := range allowed.Parse {
		delete(want, mentionType)
	}
	if len(want) != 0 {
		t.Fatalf("eksik etiket türleri: %v", want)
	}
}
