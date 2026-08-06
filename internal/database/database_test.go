//go:build cgo

package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaAndIntegrity(t *testing.T) {
	db, e := Open(filepath.Join(t.TempDir(), "adash.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = db.Integrity(); e != nil {
		t.Fatal(e)
	}
	counts, e := db.TableCounts()
	if e != nil {
		t.Fatal(e)
	}
	if len(counts) != 12 {
		t.Fatalf("beklenen 12 tablo, bulunan %d", len(counts))
	}
}
func TestExistingDatabaseCompatibility(t *testing.T) {
	source := filepath.Join("..", "..", "data")
	if _, e := os.Stat(filepath.Join(source, "adash.db")); e != nil {
		t.Skip("mevcut veritabanı yok")
	}
	target := t.TempDir()
	for _, name := range []string{"adash.db", "adash.db-wal", "adash.db-shm"} {
		raw, e := os.ReadFile(filepath.Join(source, name))
		if e != nil {
			t.Fatalf("%s okunamadı: %v", name, e)
		}
		if e = os.WriteFile(filepath.Join(target, name), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	db, e := Open(filepath.Join(target, "adash.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = db.Integrity(); e != nil {
		t.Fatal(e)
	}
	counts, e := db.TableCounts()
	if e != nil {
		t.Fatal(e)
	}
	if counts["guilds"] < 1 || counts["giveaways"] < 1 || counts["tickets"] < 1 {
		t.Fatalf("mevcut kayıtlar görünmedi: %#v", counts)
	}
}
