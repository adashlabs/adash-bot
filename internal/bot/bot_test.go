package bot

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{"10s": 10 * time.Second, "5m": 5 * time.Minute, "2h": 2 * time.Hour, "3d": 72 * time.Hour, "1w": 7 * 24 * time.Hour}
	for raw, want := range cases {
		got, e := parseDuration(raw)
		if e != nil || got != want {
			t.Fatalf("%s: %v %v", raw, got, e)
		}
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
