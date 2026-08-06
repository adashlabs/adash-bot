package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/adashlabs/adash-bot/internal/bot"
	"github.com/adashlabs/adash-bot/internal/config"
	"github.com/adashlabs/adash-bot/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	if cfg.Token == "" {
		log.Fatal("HATA: DISCORD_TOKEN bulunamadı")
	}
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("veritabanı açılamadı: %v", err)
	}
	defer db.Close()
	if err = db.Integrity(); err != nil {
		log.Fatalf("veritabanı bütünlük hatası: %v", err)
	}
	b, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("bot oluşturulamadı: %v", err)
	}
	if err = b.Start(); err != nil {
		log.Fatalf("bot başlatılamadı: %v", err)
	}
	log.Println("Adash Go hazır — tek süreç, tek shard")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	b.Close()
}
