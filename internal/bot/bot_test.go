package bot

import (
	"fmt"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"10s":   10 * time.Second,
		"5m":    5 * time.Minute,
		"2h":    2 * time.Hour,
		"3d":    72 * time.Hour,
		"1w":    7 * 24 * time.Hour,
		"1h30m": 90 * time.Minute,
		"2d12h": 60 * time.Hour,
	}
	for raw, want := range cases {
		got, e := parseDuration(raw)
		if e != nil || got != want {
			t.Fatalf("%s: %v %v", raw, got, e)
		}
	}

	futureTs := time.Now().Add(2 * time.Hour).Unix()
	futureDur, err := parseDuration(fmt.Sprintf("<t:%d:R>", futureTs))
	if err != nil || futureDur < 119*time.Minute || futureDur > 121*time.Minute {
		t.Fatalf("discord timestamp parse hatası: %v (dur: %v)", err, futureDur)
	}

	rawDur, err := parseDuration(fmt.Sprintf("%d", futureTs))
	if err != nil || rawDur < 119*time.Minute || rawDur > 121*time.Minute {
		t.Fatalf("ham unix timestamp parse hatası: %v (dur: %v)", err, rawDur)
	}

	// Test past timestamp
	if _, e := parseDuration("<t:1500000000:R>"); e == nil {
		t.Fatal("geçmiş unix zamanı reddedilmeliydi")
	}

	if _, e := parseDuration("abc"); e == nil {
		t.Fatal("geçersiz süre kabul edildi")
	}
}
func TestTurkishDictionary(t *testing.T) {
	for _, word := range []string{"ankara", "kitap", "elma"} {
		if !isDictionaryWord(word) {
			t.Errorf("sözlük kelimeyi bulamadı: %s", word)
		}
	}
	if isDictionaryWord("xqzzq") {
		t.Fatal("uydurma kelime kabul edildi")
	}
}
