# Adash Bot — Go sürümü

Adash; moderasyon, ticket, çekiliş, kanal oyunları, karşılama, TDK/web araması ve OpenAI uyumlu sohbet özellikleri olan Discord botudur. Bu sürüm düşük RAM tüketimi için Go ile tek süreç ve tek shard olarak çalışır.

## Veri uyumluluğu

Yeni sürüm önceki Node.js botuyla aynı `data/adash.db` SQLite dosyasını ve aynı tablo/kolonları kullanır. Eski kaynak kod `eski_bot/` klasöründe korunur.

Canlı veriyi taşırken yalnızca `adash.db` dosyasını kopyalamayın. SQLite WAL kullanıldığı için botu durdurduktan sonra `adash.db`, `adash.db-wal` ve `adash.db-shm` birlikte yedeklenmeli ya da kontrollü WAL checkpoint alınmalıdır.

## Docker ile çalıştırma

1. `.env.example` dosyasını `.env` olarak kopyalayın ve `DISCORD_TOKEN` değerini doldurun.
2. Kalıcı volume'un `/app/data` yoluna bağlı kaldığından emin olun.
3. Çalıştırın:

```sh
docker compose up -d --build
```

Compose yapılandırması Go çalışma belleğini 64 MiB, konteyner sınırını 96 MiB olarak ayarlar. Çok büyük sunucularda bellek sınırına yaklaşılırsa `GOMEMLIMIT` ve `mem_limit` birlikte artırılmalıdır.

## Yerel geliştirme

SQLite sürücüsü CGO kullanır; sistemde Go ve bir C derleyicisi bulunmalıdır.

```sh
go test ./...
go run ./cmd/adash
```

## Güvenli geçiş

1. Node.js botunu durdurun.
2. SQLite ana dosyasıyla WAL/SHM dosyalarının yedeğini alın.
3. Yeni Go imajını aynı kalıcı `/app/data` volume'u ile başlatın.
4. Bot hazır olduktan sonra ticket, çekiliş ve kurulum panellerini bir test sunucusunda kontrol edin.

Eski sürümü geri çalıştırmak gerekirse `eski_bot/` içindeki kaynaklar kullanılabilir; aynı veritabanına iki bot aynı anda yazmamalıdır.

## Lisans ve sözlük

Proje MIT lisanslıdır. Yerel kelime oyunu sözlüğü, önceki sürümle eşleşmesi için MIT lisanslı `nlptoolkit-dictionary@1.0.16` paketindeki `turkish_dictionary.txt` verisini kullanır.
