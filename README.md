# 🤖 Adash Bot — Modern & Gelişmiş Discord.js v14 Bot Mimarisi

[![Discord.js](https://img.shields.io/badge/discord.js-v14.27.0-5865F2?style=flat-square&logo=discord&logoColor=white)](https://discord.js.org/)
[![Node.js](https://img.shields.io/badge/node.js->=18.0.0-339933?style=flat-square&logo=nodedotjs&logoColor=white)](https://nodejs.org/)
[![SQLite](https://img.shields.io/badge/SQLite-WAL_Mode-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://www.sqlite.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white)](https://www.docker.com/)
[![Dokploy](https://img.shields.io/badge/Dokploy-Compatible-000000?style=flat-square)](https://dokploy.com/)

**Adash**, Discord.js v14 altyapısı üzerine inşa edilmiş; modüler, etkileşimli (Rich Embeds, Modals, Buttons, Select Menus), yüksek performanslı ve tam özelleştirilebilir açık kaynaklı bir Discord genel bot projesidir.

---

## 🌟 Öne Çıkan Özellikler

| Sistem | Açıklama |
| :--- | :--- |
| 🎫 **Gelişmiş Ticket Sistemi** | `/ticketsetup` veya `/kurulum` ile tek adımda kurulum. Özelleştirilebilir düğme metni/emojisi, kanal içi GUI kontrol paneli (`🙋 Üstlen`, `➕ Üye Ekle Modal`, `✏️ Adlandır Modal`, `📌 Durum Değiştir`, `🔒 Kapat Sebebi Modal`), otomatik .txt transcript ve log kaydı. |
| 🎉 **Gelişmiş Çekiliş Sistemi** | Gerçek zamanlı kazanma şansı hesaplaması (`%X.X (1 / N)`), hesap yaşı denetimi (0-365 gün), katılım rolü şartı, kalıcı veritabanı kayıtları, otomatik geri sayım ve butonla anında başlatma formları (`Modal`). |
| 🛡️ **Güçlü Moderasyon** | Düğmeli onay gerektiren `ban`, `kick`, `mute`, `unmute`, `warn`, `unban`, `lock`, `slowmode` işlemleri; onay beklemeden anında çalışan `clear` (`sil`) komutu; vaka geçmişi (`a!cases`) ve sunucu itiraz kanalı (`a!appeal`). |
| ⚙️ **Etkileşimli Kurulum Paneli** | `/kurulum` veya `a!setup` üzerinden 7 kategoride (Genel Bakış, Karşılama, Oyunlar, Ticket, Çekiliş, Yapay Zekâ, ModLog) açılır menüler (`ChannelSelectMenu`, `RoleSelectMenu`) ve butonlarla anlık yönetim. |
| 📖 **TDK & Web Araması** | `a!tdk <kelime>` resmî TDK kaynaklarını `tdk-all-api` paketiyle ayrıntılı gösterir; `a!wsearch <sorgu>` ise API anahtarı olmadan DuckDuckGo Instant Answer ve Türkçe Vikipedi sonuçlarını sayfalar. |
| 🎮 **Kanal Oyunları** | Sayı saymaca (çift paylaşım koruması) ve kelime türetmece (yerel TDK doğrulama, son harf kontrolü, tekrar engeli). |
| 🤖 **OpenAI v1 AI Asistanı** | `Yapay Zekâ` kurulum bölümünden seçilen özel kanalda otomatik yanıtlar; diğer kanallarda etiketle çalışır. Kanal başına son 12 iletiyi 30 dakika bağlamda tutar ve mention koruması uygular. |

---

## 🔒 Güvenlik & Mimarî Garantiler

1. **SQL Injection Koruması:** `better-sqlite3` hazırlanmış deyimleri (`prepared statements`) ve parametreli sorguları (`?`) kullanır.
2. **Mention Dezenfeksiyonu:** Tüm kullanıcı veya AI çıktılarında `@everyone`, `@here` ve rol etiketleri etkisizleştirilir (`allowedMentions: { parse: [] }`).
3. **Rol Hiyerarşisi:** Sunucu sahibi, üst roller ve bot rol sırası hem komut öncesi hem de buton onayı sırasında çift yönlü doğrulanır.
4. **Discord API Limit Uyumu:** Tüm dinamik paneller Discord API'sinin **en fazla 5 ActionRow** sınırına tam uyumludur.
5. **Anti-Crash:** İşlem düzeyinde `unhandledRejection` ve `uncaughtException` dinleyicileri ile beklenmeyen çökmeler engellenir.

---

## 🚀 Dokploy & Docker Kurulumu (Veri Kaybı Yaşanmadan)

Bot, **SQLite WAL (Write-Ahead Logging)** modunu kullanır. Güncellemelerde ve konteyner yeniden başlatmalarında veri kaybı yaşanmaması için veritabanının `/app/data` dizininde kalıcı olarak saklanması gerekir.

### GHCR ile Docker Compose ve Dokploy Dağıtımı
1. `main` dalına yapılan her push, [GitHub Actions](.github/workflows/publish-ghcr.yml) ile `ghcr.io/adashlabs/adash-bot:latest` imajını yayımlar.
2. İlk yayımdan sonra GitHub deposunun **Packages** alanından `adash-bot` paketini **Public** yapın. Özel paket kullanacaksanız Dokploy/Docker sunucusunda GHCR oturumu açılması gerekir.
3. Dokploy panelinde **Create Application** oluşturun; Build Type olarak **Docker Compose** seçin ve depodaki `compose.yaml` dosyasını kullanın.
4. Dokploy **Environment Variables** bölümüne `.env.example` içindeki değerleri, özellikle `DISCORD_TOKEN` değerini ekleyin.
5. **Persistent Volumes** altında `/app/data` mount path'ine kalıcı bir volume bağlayın. Compose varsayılanı `adash-data` volume'üdür.
6. Deploy edin. `pull_policy: always`, her dağıtımda GHCR'daki güncel imajı alır; `/app/data` volume'ü SQLite verisini korur.

### Standart Docker Compose Kurulumu
```bash
# Değişkenler dosyasını oluşturun ve Discord Token'ınızı girin
cp .env.example .env
nano .env

# GHCR'dan en güncel imajı çekip başlatın
docker compose pull
docker compose up -d
```

Varsayılan imaj `ghcr.io/adashlabs/adash-bot:latest`'tir. Başka bir sürüm, SHA etiketi veya kendi registry'niz için `.env` içine örneğin `ADASH_IMAGE=ghcr.io/adashlabs/adash-bot:sha-<commit>` yazabilirsiniz.

### Kalıcı Veri Dizinleri
- `adash-data:/app/data`: Sunucu ayarları, uyarilar, moderasyon logları, ticket kayıtları, çekilişler ve oyun durumları `adash.db` dosyasında güvenle saklanır.

---

## 💻 Yerel (Local) Kurulum

```bash
# Depoyu klonlayın
git clone https://github.com/kullanici/adash-bot.git
cd adash-bot

# Bağımlılıkları yükleyin
npm install

# .env dosyasını yapılandırın
cp .env.example .env

# Botu başlatın
npm start
```

---

## ⚙️ Çevre Değişkenleri (`.env`)

```env
DISCORD_TOKEN=your_bot_token_here
SHARD_COUNT=1

# İsteğe Bağlı Entegrasyonlar
SYNAPIC_API_KEY=
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_API_KEY=
OPENAI_MODEL=gpt-4o-mini
OPENAI_SYSTEM_PROMPT=
OPENAI_MAX_TOKENS=900
OPENAI_TEMPERATURE=0.7
OPENAI_TIMEOUT_MS=45000
```

### Yapay Zekâ Sohbet Kanalı

1. `.env` içinde `OPENAI_BASE_URL` ve `OPENAI_MODEL` değerlerini; servis gerektiriyorsa `OPENAI_API_KEY` değerini girin.
2. Sunucuda `/kurulum` veya `a!setup` açın, **Yapay Zekâ** bölümünde özel metin kanalını seçin.
3. Aynı bölümdeki **Sistem Promptunu Düzenle** düğmesiyle AI'ın kişiliğini, üslubunu ve konuşma kurallarını sunucuya özel olarak belirleyin. Bu ayar `.env` promptunun önüne geçer.
4. Varsayılan karakter, bilgi veren bir bot değil; Türkçe, samimi, doğal konuşan ve uygun zamanda sohbeti ilerleten bir Discord sohbet arkadaşıdır.
5. Bot bu kanaldaki her normal kullanıcı mesajına doğrudan yanıt verir. Başka kanallarda botu etiketlemek gerekir.
6. Her kanalın son 12 kullanıcı/asistan iletisi bellekte 30 dakika tutulur; bot yeniden başlatılırsa bellek güvenlik için temizlenir.

`a!wsearch` ve `/webara` için anahtar gerekmez. DuckDuckGo Instant Answer API ile Türkçe Vikipedi API'si kullanılır; HTML sayfa kazıma yapılmaz.

---

## 📜 Komut Listesi (31 Prefix & 31 Slash Komutu)

| Komut | Prefix | Slash | Açıklama |
| :--- | :--- | :--- | :--- |
| **Kurulum** | `a!setup` | `/kurulum` | Etkileşimli sunucu kurulum paneli |
| **Ticket Kurulum** | `a!ticketsetup` | `/ticketsetup` | Tek adımda butonlu ticket sistemini kurar |
| **Ticket Yönetim** | `a!ticket` | `/ticket` | Açık kanalda üye ekler/çıkarır veya adı değiştirir |
| **Çekiliş** | `a!giveaway` | `/cekilis` | Parametresiz sihirbaz veya anında çekiliş başlatma |
| **Çekiliş Yönetim** | `a!giveawaymanage` | `/cekilisyonet` | Çekilişi erken bitirir veya yeniden çeker |
| **Yasakla** | `a!ban` | `/ban` | Düğmeli onay ile kullanıcıyı yasaklar |
| **Yasak Aç** | `a!unban` | `/unban` | Düğmeli onay ile yasağı kaldırır |
| **At** | `a!kick` | `/kick` | Düğmeli onay ile kullanıcıyı atar |
| **Sustur** | `a!mute` | `/mute` | Düğmeli onay ile geçici timeout uygular |
| **Susturma Aç** | `a!unmute` | `/unmute` | Düğmeli onay ile timeout kaldırır |
| **Uyarı Ver** | `a!warn` | `/warn` | Düğmeli onay ile aktif uyarı kaydeder |
| **Uyarılar** | `a!warnings` | `/uyarilar` | Aktif uyarı geçmişini gösterir |
| **Uyarı Temizle** | `a!clearwarns` | `/uyaritemizle` | Aktif uyarıları onay ile temizler |
| **Mesaj Sil** | `a!clear` / `a!sil` | `/temizle` | 1-100 mesajı anında (onaysız) temizler |
| **Vakalar** | `a!cases` | `/cases` | Moderasyon kayıt geçmişini gösterir |
| **Mod Ayar** | `a!modconfig` | `/modconfig` | Uyarı eşiği, timeout süresi ve itiraz kanalını ayarlar |
| **İtiraz** | `a!appeal` | `/itiraz` | Yetkili ekibe gizli moderasyon itirazı gönderir |
| **Kilit** | `a!lock` | `/kilit` | Kanal kilidini açar veya kapatır |
| **Yavaş Mod** | `a!slowmode` | `/yavasmod` | Kanal mesaj süresini ayarlar |
| **TDK Sözlük** | `a!tdk` | `/tdk` | TDK tüm sözlüklerde ayrıntılı arama yapar |
| **Web Araması** | `a!wsearch` | `/webara` | Düğmeli ve sayfalı web araması yapar |
| **Kullanıcı Bilgi** | `a!userinfo` | `/kullanici` | Kullanıcı, hesap ve rol bilgilerini gösterir |
| **Sunucu Bilgi** | `a!serverinfo` | `/sunucu` | Sunucu, üye, kanal ve boost bilgilerini gösterir |
| **Avatar** | `a!avatar` | `/avatar` | Avatar resmini büyük boyutta gösterir |
| **Ping** | `a!ping` | `/ping` | Bot, RAM, CPU ve WebSocket durumunu gösterir |
| **Yardım** | `a!help` | `/yardim` | Etkileşimli kategorili yardım menüsü |
| **Oyunlar** | `a!games` | `/oyunlar` | Kanal oyunlarının durumunu ve bağlantılarını gösterir |
| **Zar** | `a!roll` | `/zar` | Zarları fırlatır |
| **Yazı Tura** | `a!coinflip` | `/yazitura` | Yazı tura atar |
| **8Ball** | `a!8ball` | `/sekiztop` | Sihirli 8Ball küresine soru sorar |
| **Prefix** | `a!prefix` | `/prefix` | Sunucunun ön ekini değiştirir |

---

## 📄 Lisans

Bu proje **MIT Lisansı** altında lisanslanmıştır. Dilediğiniz gibi geliştirebilir ve özgürce kullanabilirsiniz.
