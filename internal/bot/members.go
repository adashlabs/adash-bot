package bot

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func (b *Bot) memberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	_ = b.db.RegisterUser(e.User.ID, e.User.Username, e.User.Discriminator)
	settings, x := b.db.Settings(e.GuildID)
	if x != nil {
		return
	}
	if settings.AutoroleID.Valid {
		_ = s.GuildMemberRoleAdd(e.GuildID, e.User.ID, settings.AutoroleID.String)
	}
	if settings.WelcomeEnabled && settings.WelcomeChannelID.Valid {
		b.sendGreeting(s, e.GuildID, settings.WelcomeChannelID.String, e.User, settings.WelcomeMessage, true)
	}
}
func (b *Bot) memberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	settings, x := b.db.Settings(e.GuildID)
	if x == nil && settings.FarewellEnabled && settings.FarewellChannelID.Valid {
		b.sendGreeting(s, e.GuildID, settings.FarewellChannelID.String, e.User, settings.FarewellMessage, false)
	}
}
func (b *Bot) sendGreeting(s *discordgo.Session, guild, channel string, u *discordgo.User, template string, welcome bool) {
	g, _ := s.Guild(guild)
	server := "Sunucu"
	count := 0
	if g != nil {
		server = g.Name
		count = g.MemberCount
	}
	text := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(template, "{user}", "<@"+u.ID+">"), "{username}", u.Username), "{server}", server)
	title := str(welcome, "👋 Hoş Geldin!", "👋 Görüşürüz!")
	em := embed(title, text, strColor(welcome, colorSuccess, colorNeutral))
	em.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: u.AvatarURL("256")}
	em.Footer = &discordgo.MessageEmbedFooter{Text: "Üye sayısı: " + itoa64(int64(count))}
	send := &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{em}}
	if data, e := welcomeCard(u, server, welcome); e == nil {
		send.Files = []*discordgo.File{{Name: str(welcome, "welcome.png", "farewell.png"), ContentType: "image/png", Reader: bytes.NewReader(data)}}
		em.Image = &discordgo.MessageEmbedImage{URL: "attachment://" + str(welcome, "welcome.png", "farewell.png")}
	}
	_, _ = s.ChannelMessageSendComplex(channel, send)
}
func welcomeCard(u *discordgo.User, server string, welcome bool) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 1200, 420))
	bg := color.RGBA{35, 39, 42, 255}
	accent := color.RGBA{88, 101, 242, 255}
	if !welcome {
		accent = color.RGBA{237, 66, 69, 255}
	}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, 18, 420), &image.Uniform{accent}, image.Point{}, draw.Src)
	res, e := http.Get(u.AvatarURL("256"))
	if e == nil {
		defer res.Body.Close()
		avatar, _, x := image.Decode(res.Body)
		if x == nil {
			xdraw.CatmullRom.Scale(img, image.Rect(55, 65, 345, 355), avatar, avatar.Bounds(), draw.Over, nil)
		}
	}
	d := &font.Drawer{Dst: img, Src: image.NewUniform(color.White), Face: basicfont.Face7x13}
	d.Dot = fixed.P(390, 145)
	d.DrawString(str(welcome, "HOŞ GELDİN", "GÖRÜŞÜRÜZ"))
	d.Dot = fixed.P(390, 210)
	d.DrawString(trunc(u.Username, 55))
	d.Src = image.NewUniform(color.RGBA{190, 195, 205, 255})
	d.Dot = fixed.P(390, 270)
	d.DrawString(trunc(server, 70))
	d.Dot = fixed.P(390, 315)
	d.DrawString(time.Now().Format("02.01.2006 15:04"))
	var out bytes.Buffer
	e = png.Encode(&out, img)
	return out.Bytes(), e
}
