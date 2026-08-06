package bot

import "github.com/bwmarrin/discordgo"

func textOption(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: name, Description: desc, Required: required}
}
func integerOption(name, desc string, required bool, min, max float64) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionInteger, Name: name, Description: desc, Required: required, MinValue: &min, MaxValue: max}
}
func userOption(required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionUser, Name: "kullanici", Description: "Hedef kullanıcı", Required: required}
}
func channelOption(name, desc string, required bool, types ...discordgo.ChannelType) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionChannel, Name: name, Description: desc, Required: required, ChannelTypes: types}
}
func roleOption(name, desc string) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionRole, Name: name, Description: desc}
}
func subcommand(name, desc string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: desc, Options: options}
}
func command(name, desc string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommand {
	dm := false
	return &discordgo.ApplicationCommand{Name: name, Description: desc, Options: options, DMPermission: &dm}
}
func permitted(c *discordgo.ApplicationCommand, p int64) *discordgo.ApplicationCommand {
	c.DefaultMemberPermissions = &p
	return c
}

func slashCommands() []*discordgo.ApplicationCommand {
	reason := func() *discordgo.ApplicationCommandOption { return textOption("sebep", "İşlem sebebi", false) }
	manageGuild := int64(discordgo.PermissionManageServer)
	manageChannels := int64(discordgo.PermissionManageChannels)
	manageMessages := int64(discordgo.PermissionManageMessages)
	moderate := int64(discordgo.PermissionModerateMembers)
	ban := int64(discordgo.PermissionBanMembers)
	return []*discordgo.ApplicationCommand{
		command("yardim", "Etkileşimli yardım menüsünü açar."), permitted(command("kurulum", "Gelişmiş sunucu kurulum panelini açar."), manageGuild), command("ping", "Botun bağlantı ve sistem durumunu gösterir."), permitted(command("embed", "Yöneticiler için embed oluşturucu açar."), manageGuild), command("sunucu", "Sunucu bilgilerini gösterir."),
		command("kullanici", "Kullanıcı bilgilerini gösterir.", userOption(false)), command("avatar", "Kullanıcının avatarını gösterir.", userOption(false)), command("oyunlar", "Kanal oyunlarının durumunu gösterir."), command("tdk", "TDK sözlükte ayrıntılı arama yapar.", textOption("kelime", "Aranacak kelime", true)), command("webara", "Web araması yapar.", textOption("sorgu", "Arama sorgusu", true)), command("zar", "Zar atar.", textOption("zar", "Örnek: 2d20", false)), command("yazitura", "Yazı tura atar."), command("sekiztop", "Sihirli küreye soru sorar.", textOption("soru", "Sorun", true)), permitted(command("prefix", "Komut ön ekini değiştirir.", textOption("deger", "Yeni prefix", true)), manageGuild),
		permitted(command("ticketsetup", "Ticket sistemini tek adımda kurar.", channelOption("kategori", "Ticket kategorisi", true, discordgo.ChannelTypeGuildCategory), channelOption("panel_kanali", "Panel kanalı", true, discordgo.ChannelTypeGuildText), channelOption("log_kanali", "Log kanalı", false, discordgo.ChannelTypeGuildText), roleOption("destek_rolu", "Destek rolü")), manageGuild),
		command("ticket", "Açık ticket kanalını yönetir.", subcommand("ekle", "Kullanıcı ekler", userOption(true)), subcommand("cikar", "Kullanıcı çıkarır", userOption(true)), subcommand("adlandir", "Kanalı adlandırır", textOption("ad", "Yeni kanal adı", true))),
		permitted(command("cekilis", "Gelişmiş çekiliş oluşturur.", textOption("sure", "10m, 2h, 3d", true), integerOption("kazanan", "Kazanan sayısı", true, 1, 20), textOption("odul", "Çekiliş ödülü", true)), manageGuild),
		permitted(command("cekilisyonet", "Çekilişi bitirir veya yeniden çeker.", subcommand("bitir", "Erken bitirir", integerOption("id", "Çekiliş ID", true, 1, 999999999)), subcommand("yeniden", "Yeniden çeker", integerOption("id", "Çekiliş ID", true, 1, 999999999), integerOption("kazanan", "Kazanan sayısı", false, 1, 20))), manageGuild),
		command("ban", "Kullanıcıyı onayla yasaklar.", userOption(true), reason(), integerOption("mesaj_sil", "Silinecek geçmiş mesaj günü", false, 0, 7)), command("kick", "Kullanıcıyı onayla sunucudan atar.", userOption(true), reason()), permitted(command("unban", "Yasağı onayla kaldırır.", textOption("kullanici_id", "Yasaklı kullanıcı ID", true), reason()), ban), command("mute", "Kullanıcıyı onayla susturur.", userOption(true), textOption("sure", "10m, 1h, 2d", true), reason()), command("unmute", "Susturmayı onayla kaldırır.", userOption(true), reason()), command("warn", "Kullanıcıya onayla uyarı verir.", userOption(true), reason()), command("uyarilar", "Aktif uyarıları gösterir.", userOption(false)), permitted(command("uyaritemizle", "Aktif uyarıları temizler.", userOption(true)), moderate), permitted(command("cases", "Moderasyon vaka geçmişini gösterir.", userOption(false)), moderate),
		permitted(command("modconfig", "Moderasyon otomasyonunu ayarlar.", subcommand("uyari", "Uyarı cezasını ayarla", integerOption("esik", "Uyarı eşiği", true, 1, 10), textOption("sure", "10m, 1h, 2d", true)), subcommand("itiraz", "İtiraz kanalını ayarla", channelOption("kanal", "İtiraz kanalı", false, discordgo.ChannelTypeGuildText))), manageGuild), command("itiraz", "Yetkili ekibe gizli itiraz gönderir.", textOption("metin", "Ayrıntılı itiraz", true)),
		permitted(command("kilit", "Kanal kilidini yönetir.", &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: "islem", Description: "İşlem", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "Kilitle", Value: "kilitle"}, {Name: "Kilidi Aç", Value: "aç"}}}), manageChannels), permitted(command("yavasmod", "Kanal yavaş modunu ayarlar.", textOption("sure", "0, 10s, 5m, 1h", true)), manageChannels), permitted(command("temizle", "Mesajları temizler.", integerOption("sayi", "Mesaj sayısı", true, 1, 100), userOption(false)), manageMessages),
	}
}
