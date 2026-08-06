package bot

import (
	"testing"

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
