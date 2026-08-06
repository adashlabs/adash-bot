package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var e error
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		e = b.applicationCommand(s, i)
	case discordgo.InteractionMessageComponent:
		e = b.component(s, i)
	case discordgo.InteractionModalSubmit:
		e = b.modalSubmit(s, i)
	}
	if e != nil {
		log.Printf("interaction %s: %v", interactionID(i), e)
		b.interactionError(s, i, e)
	}
}
func interactionID(i *discordgo.InteractionCreate) string {
	if i.Type == discordgo.InteractionApplicationCommand {
		return i.ApplicationCommandData().Name
	}
	if i.Type == discordgo.InteractionMessageComponent {
		return i.MessageComponentData().CustomID
	}
	return i.ModalSubmitData().CustomID
}
func (b *Bot) interactionError(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	em := errorEmbed(trunc(err.Error(), 1800))
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{em},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	}
	if responseErr := s.InteractionRespond(i.Interaction, response); responseErr != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{em},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
	}
}
func userOf(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}
func flattenOptions(xs []*discordgo.ApplicationCommandInteractionDataOption) (string, map[string]string) {
	sub := ""
	out := map[string]string{}
	var walk func([]*discordgo.ApplicationCommandInteractionDataOption)
	walk = func(items []*discordgo.ApplicationCommandInteractionDataOption) {
		for _, o := range items {
			if o.Type == discordgo.ApplicationCommandOptionSubCommand || o.Type == discordgo.ApplicationCommandOptionSubCommandGroup {
				sub = o.Name
				walk(o.Options)
				continue
			}
			switch v := o.Value.(type) {
			case string:
				out[o.Name] = v
			case float64:
				out[o.Name] = strconv.FormatInt(int64(v), 10)
			default:
				out[o.Name] = fmt.Sprint(v)
			}
		}
	}
	walk(xs)
	return sub, out
}
func (b *Bot) applicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	name := i.ApplicationCommandData().Name
	u := userOf(i)
	if name == "yardim" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{b.helpEmbed(i.GuildID, u.ID)}, Components: []discordgo.MessageComponent{helpMenu(u.ID)}, Flags: discordgo.MessageFlagsEphemeral}})
	}
	if name == "kurulum" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{b.setupEmbed(i.GuildID, "genel")}, Components: setupComponents(i.GuildID, "genel"), Flags: discordgo.MessageFlagsEphemeral}})
	}
	if e := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource}); e != nil {
		return e
	}
	sub, o := flattenOptions(i.ApplicationCommandData().Options)
	command, args := slashArgs(name, sub, o)
	b.db.LogCommand(i.GuildID, u.ID, command, strings.Join(args, " "))
	ctx := &commandContext{b: b, s: s, guildID: i.GuildID, channelID: i.ChannelID, user: u, member: i.Member, interaction: i}
	return b.runCommand(ctx, command, args)
}
func slashArgs(name, sub string, o map[string]string) (string, []string) {
	switch name {
	case "sunucu":
		return "serverinfo", nil
	case "kullanici":
		return "userinfo", compact(o["kullanici"])
	case "avatar":
		return "avatar", compact(o["kullanici"])
	case "oyunlar":
		return "games", nil
	case "tdk":
		return "tdk", []string{o["kelime"]}
	case "webara":
		return "wsearch", strings.Fields(o["sorgu"])
	case "zar":
		return "roll", compact(o["zar"])
	case "yazitura":
		return "coinflip", nil
	case "sekiztop":
		return "8ball", strings.Fields(o["soru"])
	case "prefix":
		return "prefix", []string{o["deger"]}
	case "ticketsetup":
		return "ticketsetup", compact(o["kategori"], o["panel_kanali"], o["log_kanali"], o["destek_rolu"])
	case "ticket":
		if sub == "adlandir" {
			return "ticket", []string{"adlandır", o["ad"]}
		}
		return "ticket", []string{str(sub == "ekle", "ekle", "çıkar"), o["kullanici"]}
	case "cekilis":
		return "giveaway", []string{o["sure"], o["kazanan"], o["odul"]}
	case "cekilisyonet":
		return "giveawaymanage", compact(sub, o["id"], o["kazanan"])
	case "ban":
		return "ban", compact(o["kullanici"], o["sebep"], "--days="+valueOr(o["mesaj_sil"], "0"))
	case "kick", "unmute", "warn":
		return name, compact(o["kullanici"], o["sebep"])
	case "mute":
		return "mute", compact(o["kullanici"], o["sure"], o["sebep"])
	case "unban":
		return "unban", compact(o["kullanici_id"], o["sebep"])
	case "uyarilar":
		return "warnings", compact(o["kullanici"])
	case "uyaritemizle":
		return "clearwarns", compact(o["kullanici"])
	case "cases":
		return "cases", compact(o["kullanici"])
	case "temizle":
		return "clear", compact(o["sayi"], o["kullanici"])
	case "kilit":
		return "lock", []string{o["islem"]}
	case "yavasmod":
		return "slowmode", []string{o["sure"]}
	case "itiraz":
		return "appeal", strings.Fields(o["metin"])
	case "modconfig":
		if sub == "uyari" {
			return "modconfig", []string{"warn", o["esik"], o["sure"]}
		}
		return "modconfig", []string{"appeal", valueOr(o["kanal"], "kapalı")}
	default:
		return name, nil
	}
}
func compact(xs ...string) []string {
	out := []string{}
	for _, x := range xs {
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}
func (b *Bot) component(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	id := i.MessageComponentData().CustomID
	u := userOf(i)
	if strings.HasPrefix(id, "help_menu:") {
		if !strings.HasSuffix(id, u.ID) {
			return fmt.Errorf("bu yardım paneli sana ait değil")
		}
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{b.helpCategory(i.GuildID, i.MessageComponentData().Values[0])}, Components: []discordgo.MessageComponent{helpMenu(u.ID)}}})
	}
	if strings.HasPrefix(id, "setup_") || strings.HasPrefix(id, "ticketsetup_") {
		return b.setupComponent(s, i)
	}
	if strings.HasPrefix(id, "modconfirm:") {
		return b.confirmComponent(s, i)
	}
	if id == "giveaway_join" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return b.toggleGiveaway(s, i)
	}
	if id == "ticket_open" {
		return b.ticketOpenModal(s, i)
	}
	if strings.HasPrefix(id, "ticket_") {
		return b.ticketComponent(s, i)
	}
	if strings.HasPrefix(id, "embed_builder:") {
		return b.embedComponent(s, i)
	}
	if strings.HasPrefix(id, "wsearch:") {
		return b.searchComponent(s, i)
	}
	if id == "ping_refresh" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{b.pingEmbed(i.GuildID)}, Components: []discordgo.MessageComponent{row(button("ping_refresh", "Yenile", discordgo.PrimaryButton, "🔄"))}}})
	}
	if id == "coinflip_retry" {
		em := embed("🪙 Yazı Tura", str(time.Now().UnixNano()%2 == 0, "**Yazı!**", "**Tura!**"), colorSuccess)
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{em}, Components: i.Message.Components}})
	}
	if strings.HasPrefix(id, "roll_retry:") {
		spec := strings.TrimPrefix(id, "roll_retry:")
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		ctx := &commandContext{b: b, s: s, guildID: i.GuildID, channelID: i.ChannelID, user: u, member: i.Member, interaction: i}
		return b.roll(ctx, spec)
	}
	return nil
}
func (b *Bot) confirmComponent(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 3 {
		return nil
	}
	userID := userOf(i).ID
	b.mu.Lock()
	item, ok := b.confirms[parts[2]]
	if ok && item.UserID == userID && item.GuildID == i.GuildID {
		delete(b.confirms, parts[2])
	}
	b.mu.Unlock()
	if !ok || time.Now().After(item.Expires) {
		return fmt.Errorf("onay süresi doldu")
	}
	if item.UserID != userID || item.GuildID != i.GuildID {
		return fmt.Errorf("bu onay sana ait değil")
	}
	if parts[1] != "yes" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{moderationResultEmbed(item, "cancelled", "İşlem uygulanmadan iptal edildi.")},
				Components: []discordgo.MessageComponent{completedModerationButton("cancelled")},
			},
		})
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return err
	}
	if err := item.Action(); err != nil {
		log.Printf("moderation confirmation %s: %v", parts[2], err)
		message := trunc(err.Error(), 1500)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{moderationResultEmbed(item, "failed", message)},
			Components: &[]discordgo.MessageComponent{completedModerationButton("failed")},
		})
		return nil
	}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{moderationResultEmbed(item, "success", "Yetkili onayı alındı ve işlem başarıyla uygulandı.")},
		Components: &[]discordgo.MessageComponent{completedModerationButton("success")},
	})
	return err
}
func (b *Bot) helpCategory(guild, section string) *discordgo.MessageEmbed {
	p := b.db.Prefix(guild)
	items := map[string]string{"genel": "`ping`, `yardim`, `kurulum`, `prefix`", "moderasyon": "`ban`, `kick`, `mute`, `unmute`, `warn`, `uyarilar`, `uyaritemizle`, `temizle`, `kilit`, `yavasmod`, `cases`, `itiraz`", "sistemler": "`ticketsetup`, `ticket`, `cekilis`, `cekilisyonet`, `oyunlar`", "araclar": "`tdk`, `webara`, `kullanici`, `sunucu`, `avatar`, `zar`, `yazitura`, `sekiztop`, `embed`"}
	return embed("📖 "+strings.ToUpper(section), items[section]+"\n\nPrefix: `"+p+"`", colorPrimary)
}
