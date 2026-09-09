package bot

import "testing"

func TestSlashCommandParity(t *testing.T) {
	commands := slashCommands()
	if len(commands) != 33 {
		t.Fatalf("beklenen 33 slash komutu, bulunan %d", len(commands))
	}
	seen := map[string]bool{}
	for _, command := range commands {
		if seen[command.Name] {
			t.Fatalf("yinelenen slash komutu: %s", command.Name)
		}
		seen[command.Name] = true
	}
	for _, name := range []string{"kurulum", "ticketsetup", "ticket", "cekilis", "cekilisyonet", "yetkili", "ban", "kick", "mute", "warn", "temizle", "tdk", "webara"} {
		if !seen[name] {
			t.Errorf("eksik slash komutu: %s", name)
		}
	}
}
