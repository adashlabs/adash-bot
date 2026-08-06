package bot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	mathrand "math/rand/v2"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type commandContext struct {
	b                  *Bot
	s                  *discordgo.Session
	guildID, channelID string
	user               *discordgo.User
	member             *discordgo.Member
	interaction        *discordgo.InteractionCreate
}

func (c *commandContext) send(content string, e *discordgo.MessageEmbed, components ...discordgo.MessageComponent) error {
	if c.interaction != nil {
		edit := &discordgo.WebhookEdit{Content: &content, Components: &components}
		if e != nil {
			edit.Embeds = &[]*discordgo.MessageEmbed{e}
		}
		_, err := c.s.InteractionResponseEdit(c.interaction.Interaction, edit)
		return err
	}
	p := &discordgo.MessageSend{Content: content, Components: components}
	if e != nil {
		p.Embeds = []*discordgo.MessageEmbed{e}
	}
	_, err := c.s.ChannelMessageSendComplex(c.channelID, p)
	return err
}
func (c *commandContext) text(s string) error { return c.send(s, nil) }
func (c *commandContext) embed(e *discordgo.MessageEmbed, components ...discordgo.MessageComponent) error {
	return c.send("", e, components...)
}
func (c *commandContext) permissions() (int64, error) {
	return c.s.UserChannelPermissions(c.user.ID, c.channelID)
}
func (c *commandContext) require(p int64) error {
	v, e := c.permissions()
	if e != nil {
		return e
	}
	if v&p == 0 && v&discordgo.PermissionAdministrator == 0 {
		return fmt.Errorf("bu komut için gerekli yetkin yok")
	}
	return nil
}
func (b *Bot) runPrefix(s *discordgo.Session, m *discordgo.MessageCreate, name string, args []string) error {
	member := m.Member
	if member == nil {
		member, _ = s.GuildMember(m.GuildID, m.Author.ID)
	}
	return b.runCommand(&commandContext{b: b, s: s, guildID: m.GuildID, channelID: m.ChannelID, user: m.Author, member: member}, name, args)
}

func (b *Bot) runCommand(c *commandContext, name string, args []string) error {
	switch name {
	case "ping":
		return c.embed(b.pingEmbed(c.guildID), row(button("ping_refresh", "Yenile", discordgo.PrimaryButton, "🔄")))
	case "help":
		return c.embed(b.helpEmbed(c.guildID, c.user.ID), helpMenu(c.user.ID))
	case "serverinfo":
		return b.serverInfo(c)
	case "userinfo":
		return b.userInfo(c, first(args, c.user.ID))
	case "avatar":
		return b.avatar(c, first(args, c.user.ID))
	case "games":
		return b.gamesInfo(c)
	case "roll":
		return b.roll(c, first(args, "1d6"))
	case "coinflip":
		return c.embed(embed("🪙 Yazı Tura", str(mathrand.IntN(2) == 0, "**Yazı!**", "**Tura!**"), colorSuccess), row(button("coinflip_retry", "Tekrar Fırlat", discordgo.SuccessButton, "🪙")))
	case "8ball":
		if len(args) == 0 {
			return fmt.Errorf("bir soru yazmalısın")
		}
		answers := []string{"Kesinlikle evet.", "Büyük ihtimalle.", "Şimdilik hayır.", "Pek sanmıyorum.", "Bunu zaman gösterecek.", "İçime doğan: evet!", "Tekrar sorsan daha iyi."}
		return c.embed(embed("🎱 Sihirli 8Ball", fmt.Sprintf("**Soru:** %s\n**Yanıt:** %s", trunc(strings.Join(args, " "), 500), answers[mathrand.IntN(len(answers))]), colorPrimary), row(button("8ball_retry", "Yeniden Sor", discordgo.PrimaryButton, "🎱")))
	case "prefix":
		if e := c.require(discordgo.PermissionManageServer); e != nil {
			return e
		}
		if len(args) != 1 || len([]rune(args[0])) > 5 {
			return fmt.Errorf("prefix 1–5 karakter olmalı")
		}
		if e := b.db.SetPrefix(c.guildID, args[0]); e != nil {
			return e
		}
		return c.embed(successEmbed("✅ Prefix Güncellendi", fmt.Sprintf("Yeni prefix: `%s`", args[0])))
	case "tdk":
		if len(args) == 0 {
			return fmt.Errorf("bir kelime yazmalısın")
		}
		return b.tdk(c, strings.Join(args, " "))
	case "wsearch":
		if len(args) == 0 {
			return fmt.Errorf("bir arama sorgusu yazmalısın")
		}
		return b.webSearch(c, strings.Join(args, " "))
	case "setup":
		if e := c.require(discordgo.PermissionManageServer); e != nil {
			return e
		}
		return c.embed(b.setupEmbed(c.guildID, "genel"), setupMenu(c.guildID))
	case "embed":
		if e := c.require(discordgo.PermissionManageServer); e != nil {
			return e
		}
		return b.startEmbedBuilder(c)
	case "ban", "kick", "unban", "mute", "unmute", "warn", "clearwarns", "lock", "slowmode":
		return b.runModeration(c, name, args)
	case "warnings":
		return b.showWarnings(c, first(args, c.user.ID))
	case "cases":
		return b.showCases(c, first(args, ""))
	case "clear":
		return b.clearMessages(c, args)
	case "modconfig":
		return b.modConfig(c, args)
	case "appeal":
		return b.appeal(c, args)
	case "ticketsetup":
		return b.ticketSetupCommand(c, args)
	case "ticket":
		return b.ticketCommand(c, args)
	case "giveaway":
		return b.giveawayCommand(c, args)
	case "giveawaymanage":
		return b.giveawayManage(c, args)
	default:
		return fmt.Errorf("bilinmeyen komut: %s", name)
	}
}
func first(a []string, f string) string {
	if len(a) > 0 {
		return a[0]
	}
	return f
}
func (b *Bot) helpEmbed(guild, user string) *discordgo.MessageEmbed {
	p := b.db.Prefix(guild)
	return &discordgo.MessageEmbed{Title: "🤖 Adash Yardım", Description: "Aşağıdaki menüden kategori seçebilirsin. Prefix ve slash komutlarının tamamı kullanılabilir.", Color: colorPrimary, Fields: []*discordgo.MessageEmbedField{{Name: "🛡️ Moderasyon", Value: "ban, kick, mute, warn, temizle, cases"}, {Name: "🎫 Sistemler", Value: "ticket, çekiliş, kurulum, oyunlar"}, {Name: "🔎 Araçlar", Value: "tdk, webara, avatar, sunucu, kullanıcı"}, {Name: "🎮 Eğlence", Value: "zar, yazı tura, 8ball"}, {Name: "Prefix", Value: "`" + p + "`", Inline: true}}, Footer: &discordgo.MessageEmbedFooter{Text: "Yalnızca paneli açan kişi kullanabilir."}}
}
func helpMenu(user string) discordgo.ActionsRow {
	return row(discordgo.SelectMenu{CustomID: "help_menu:" + user, Placeholder: "Yardım kategorisi seç", Options: []discordgo.SelectMenuOption{{Label: "Genel", Value: "genel", Emoji: &discordgo.ComponentEmoji{Name: "🏠"}}, {Label: "Moderasyon", Value: "moderasyon", Emoji: &discordgo.ComponentEmoji{Name: "🛡️"}}, {Label: "Ticket ve Çekiliş", Value: "sistemler", Emoji: &discordgo.ComponentEmoji{Name: "🎫"}}, {Label: "Araçlar ve Eğlence", Value: "araclar", Emoji: &discordgo.ComponentEmoji{Name: "🧰"}}}})
}
func (b *Bot) serverInfo(c *commandContext) error {
	g, e := c.s.Guild(c.guildID)
	if e != nil {
		return e
	}
	owner := "<@" + g.OwnerID + ">"
	channels := len(g.Channels)
	members := g.MemberCount
	roles := len(g.Roles)
	em := &discordgo.MessageEmbed{Title: "🏠 " + g.Name, Color: colorPrimary, Thumbnail: &discordgo.MessageEmbedThumbnail{URL: g.IconURL("256")}, Fields: []*discordgo.MessageEmbedField{{Name: "Sahip", Value: owner, Inline: true}, {Name: "Üyeler", Value: fmt.Sprint(members), Inline: true}, {Name: "Kanallar", Value: fmt.Sprint(channels), Inline: true}, {Name: "Roller", Value: fmt.Sprint(roles), Inline: true}, {Name: "Sunucu ID", Value: g.ID, Inline: true}, {Name: "Oluşturulma", Value: "<t:" + strconv.FormatInt(snowflakeTime(g.ID).Unix(), 10) + ":F>", Inline: true}}, Timestamp: time.Now().Format(time.RFC3339)}
	return c.embed(em)
}
func snowflakeTime(id string) time.Time {
	n, _ := strconv.ParseInt(id, 10, 64)
	return time.UnixMilli((n >> 22) + 1420070400000)
}
func (b *Bot) userInfo(c *commandContext, id string) error {
	id = mentionID(id)
	if id == "" {
		id = c.user.ID
	}
	u, e := c.s.User(id)
	if e != nil {
		return e
	}
	m, _ := c.s.GuildMember(c.guildID, id)
	roles := "Yok"
	joined := "Bilinmiyor"
	if m != nil {
		if len(m.Roles) > 0 {
			xs := make([]string, len(m.Roles))
			for i, r := range m.Roles {
				xs[i] = "<@&" + r + ">"
			}
			roles = trunc(strings.Join(xs, " "), 900)
		}
		joined = "<t:" + strconv.FormatInt(m.JoinedAt.Unix(), 10) + ":F>"
	}
	em := &discordgo.MessageEmbed{Title: "👤 " + u.Username, Color: colorPrimary, Thumbnail: &discordgo.MessageEmbedThumbnail{URL: u.AvatarURL("256")}, Fields: []*discordgo.MessageEmbedField{{Name: "Kullanıcı", Value: "<@" + u.ID + ">", Inline: true}, {Name: "ID", Value: u.ID, Inline: true}, {Name: "Hesap Açılışı", Value: "<t:" + strconv.FormatInt(snowflakeTime(u.ID).Unix(), 10) + ":F>", Inline: false}, {Name: "Sunucuya Katılım", Value: joined, Inline: false}, {Name: "Roller", Value: roles}}}
	return c.embed(em)
}
func (b *Bot) avatar(c *commandContext, id string) error {
	id = mentionID(id)
	if id == "" {
		id = c.user.ID
	}
	u, e := c.s.User(id)
	if e != nil {
		return e
	}
	url := u.AvatarURL("1024")
	em := embed("🖼️ "+u.Username+" Avatarı", "[Avatarı tarayıcıda aç]"+"("+url+")", colorPrimary)
	em.Image = &discordgo.MessageEmbedImage{URL: url}
	return c.embed(em, row(discordgo.Button{Style: discordgo.LinkButton, Label: "Avatarı Aç", URL: url}))
}
func (b *Bot) gamesInfo(c *commandContext) error {
	s, e := b.db.Settings(c.guildID)
	if e != nil {
		return e
	}
	count := "Ayarlı değil"
	word := "Ayarlı değil"
	if s.CountingChannelID.Valid {
		count = "<#" + s.CountingChannelID.String + "> · " + boolIcon(s.CountingEnabled)
	}
	if s.WordChainChannelID.Valid {
		word = "<#" + s.WordChainChannelID.String + "> · " + boolIcon(s.WordChainEnabled)
	}
	return c.embed(&discordgo.MessageEmbed{Title: "🎮 Kanal Oyunları", Color: colorPrimary, Fields: []*discordgo.MessageEmbedField{{Name: "🔢 Sayı Saymaca", Value: count}, {Name: "🔤 Kelime Türetmece", Value: word}}})
}
func (b *Bot) roll(c *commandContext, spec string) error {
	re := regexp.MustCompile(`^(\d{1,2})d(\d{1,4})$`)
	x := re.FindStringSubmatch(strings.ToLower(spec))
	if x == nil {
		return fmt.Errorf("zar biçimi `2d20` gibi olmalı")
	}
	count, _ := strconv.Atoi(x[1])
	sides, _ := strconv.Atoi(x[2])
	if count < 1 || count > 20 || sides < 2 {
		return fmt.Errorf("1–20 zar ve en az 2 yüz kullan")
	}
	vals := make([]int, count)
	total := 0
	for i := range vals {
		vals[i] = mathrand.IntN(sides) + 1
		total += vals[i]
	}
	parts := make([]string, count)
	for i, v := range vals {
		parts[i] = strconv.Itoa(v)
	}
	return c.embed(embed("🎲 Zar Sonucu", fmt.Sprintf("`%s` → **%d**\n%s", spec, total, strings.Join(parts, ", ")), colorPrimary), row(button("roll_retry:"+spec, "Tekrar Zar At", discordgo.PrimaryButton, "🎲")))
}
func (b *Bot) startEmbedBuilder(c *commandContext) error {
	d := &embedDraft{ChannelID: c.channelID, Color: "#5865F2", Updated: time.Now()}
	b.mu.Lock()
	b.drafts[c.user.ID] = d
	b.mu.Unlock()
	return c.embed(previewDraftEmbed(d), embedBuilderControls(c.user.ID, 0)...)
}
func token() string { v := make([]byte, 8); _, _ = rand.Read(v); return hex.EncodeToString(v) }
func sortedKeys(m map[string]int64) []string {
	k := make([]string, 0, len(m))
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}

var _ = runtime.GOMAXPROCS
var _ = math.Min
