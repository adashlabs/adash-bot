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

## DPI Bypass Mimarisi (Türkiye / ISS Engelleri İçin)

Docker kurulumunda yer alan `dpi` servisi, Türkiye'deki servis sağlayıcıların uyguladığı Discord engellemelerini aşmak için özel olarak yapılandırılmıştır:
- **ByeDPI (`ciadpi` v0.17.3)**: C ile yazılmış, sadece ~2 MiB RAM tüketen ultra hafif DPI aşma motorudur. `--split 1 --disorder 3+s --auto=torst --tlsrec 1+s` parametreleri ile TLS SNI paketlerini sırasız ve TLS kayıt sınırında bölerek iletir; ISS'lerin derin paket inceleme (DPI) donanımları paketi birleştiremez.
- **DoH ve Köprü (`dpi-bridge`)**: Standart UDP 53 portu Türkiye'deki ISS'ler tarafından zehirlendiği için, Discord alan adları doğrudan HTTPS (port 443) üzerinden Cloudflare (`1.1.1.1`) ve Google (`8.8.8.8`) DoH sunucuları ile çözümlenir ve önbelleğe alınır.
- **HTTP CONNECT & SOCKS5**: Bot `http://dpi:8080` (HTTP CONNECT) üzerinden bağlanır; ayrıca doğrudan `1080` SOCKS5 portu da mevcuttur.
- **Düşük Kaynak Tüketimi**: Tüm DPI konteyneri toplamda yalnızca **~8 MiB RAM** kullanır ve Docker healthcheck ile sürekli denetlenir.

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
