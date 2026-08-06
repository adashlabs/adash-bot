package bot

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func (b *Bot) memberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	_ = b.db.RegisterUser(e.User.ID, e.User.Username, e.User.Discriminator)
	settings, err := b.db.Settings(e.GuildID)
	if err != nil {
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
	settings, err := b.db.Settings(e.GuildID)
	if err == nil && settings.FarewellEnabled && settings.FarewellChannelID.Valid {
		b.sendGreeting(s, e.GuildID, settings.FarewellChannelID.String, e.User, settings.FarewellMessage, false)
	}
}

func (b *Bot) sendGreeting(s *discordgo.Session, guildID, channelID string, u *discordgo.User, template string, welcome bool) {
	guild, _ := s.Guild(guildID)
	server, count := "Sunucu", 0
	if guild != nil {
		server, count = guild.Name, guild.MemberCount
	}
	displayName := valueOr(u.GlobalName, u.Username)
	text := strings.NewReplacer("{user}", "<@"+u.ID+">", "{username}", displayName, "{server}", server).Replace(template)
	title := "👋 Hoş geldin, " + displayName
	footer := fmt.Sprintf("%s • %d. üye", server, count)
	if !welcome {
		title = "👋 Görüşmek üzere, " + displayName
		footer = fmt.Sprintf("%s • Sunucuda %d üye", server, count)
	}
	em := embed(trunc(title, 256), text, strColor(welcome, colorSuccess, colorDanger))
	em.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: u.AvatarURL("256")}
	em.Footer = &discordgo.MessageEmbedFooter{Text: trunc(footer, 2048)}
	send := &discordgo.MessageSend{
		Embeds:          []*discordgo.MessageEmbed{em},
		AllowedMentions: &discordgo.MessageAllowedMentions{Users: []string{u.ID}},
	}
	if data, err := welcomeCard(u, server, count, welcome); err == nil {
		name := str(welcome, "welcome.png", "farewell.png")
		send.Files = []*discordgo.File{{Name: name, ContentType: "image/png", Reader: bytes.NewReader(data)}}
		em.Image = &discordgo.MessageEmbedImage{URL: "attachment://" + name}
	}
	_, _ = s.ChannelMessageSendComplex(channelID, send)
}

func welcomeCard(u *discordgo.User, server string, memberCount int, welcome bool) ([]byte, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Get(u.AvatarURL("512"))
	if err != nil {
		return renderMemberCard(nil, valueOr(u.GlobalName, u.Username), server, memberCount, welcome)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return renderMemberCard(nil, valueOr(u.GlobalName, u.Username), server, memberCount, welcome)
	}
	avatar, _, err := image.Decode(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		avatar = nil
	}
	return renderMemberCard(avatar, valueOr(u.GlobalName, u.Username), server, memberCount, welcome)
}

func renderMemberCard(avatar image.Image, displayName, server string, memberCount int, welcome bool) ([]byte, error) {
	const width, height = 1200, 420
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	accent := color.RGBA{35, 209, 139, 255}
	bgEnd := color.RGBA{18, 55, 42, 255}
	if !welcome {
		accent = color.RGBA{244, 82, 91, 255}
		bgEnd = color.RGBA{66, 31, 41, 255}
	}
	gradient(img, color.RGBA{17, 24, 39, 255}, bgEnd)
	for _, c := range []struct{ x, y, r int }{{850, 410, 190}, {1010, 390, 150}, {1130, 370, 115}, {710, 410, 135}} {
		fillCircle(img, c.x, c.y, c.r, color.RGBA{accent.R, accent.G, accent.B, 24})
	}
	fillRoundedRect(img, image.Rect(28, 26, 1172, 394), 28, color.RGBA{5, 10, 18, 145})
	fillRoundedRect(img, image.Rect(28, 26, 38, 394), 6, accent)

	avatarRect := image.Rect(84, 76, 352, 344)
	fillCircle(img, 218, 210, 147, color.RGBA{accent.R, accent.G, accent.B, 70})
	fillCircle(img, 218, 210, 142, accent)
	fillCircle(img, 218, 210, 134, color.RGBA{20, 27, 38, 255})
	if avatar != nil {
		square := centerCrop(avatar)
		scaled := image.NewRGBA(image.Rect(0, 0, avatarRect.Dx(), avatarRect.Dy()))
		xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), avatar, square, draw.Src, nil)
		mask := circleMask(avatarRect.Dx(), avatarRect.Dy())
		draw.DrawMask(img, avatarRect, scaled, image.Point{}, mask, image.Point{}, draw.Over)
	} else {
		face, _ := newCardFace(true, 90)
		if face != nil {
			defer face.Close()
			drawCenteredText(img, face, strings.ToUpper(firstRune(displayName)), 218, 239, color.White)
		}
	}

	regular, err := newCardFace(false, 27)
	if err != nil {
		return nil, err
	}
	defer regular.Close()
	label, err := newCardFace(true, 30)
	if err != nil {
		return nil, err
	}
	defer label.Close()
	nameFace, err := newCardFace(true, 58)
	if err != nil {
		return nil, err
	}
	defer nameFace.Close()
	serverFace, err := newCardFace(false, 36)
	if err != nil {
		return nil, err
	}
	defer serverFace.Close()

	drawText(img, label, str(welcome, "HOŞ GELDİN", "GÖRÜŞMEK ÜZERE"), 410, 116, accent)
	drawText(img, nameFace, fitText(nameFace, displayName, 710), 410, 194, color.White)
	drawText(img, serverFace, fitText(serverFace, server, 710), 410, 250, color.RGBA{205, 211, 221, 255})
	detail := fmt.Sprintf("Seninle birlikte %d üyeyiz.", memberCount)
	if !welcome {
		detail = fmt.Sprintf("Sunucuda %d üye kaldı.", memberCount)
	}
	drawText(img, regular, fitText(regular, detail, 710), 410, 308, color.RGBA{157, 166, 180, 255})
	drawText(img, regular, time.Now().Format("02.01.2006 • 15:04"), 410, 350, color.RGBA{125, 135, 151, 255})

	var out bytes.Buffer
	err = png.Encode(&out, img)
	return out.Bytes(), err
}

func newCardFace(bold bool, size float64) (font.Face, error) {
	data := goregular.TTF
	if bold {
		data = gobold.TTF
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

func gradient(dst *image.RGBA, from, to color.RGBA) {
	h := dst.Bounds().Dy()
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		c := color.RGBA{uint8(float64(from.R)*(1-t) + float64(to.R)*t), uint8(float64(from.G)*(1-t) + float64(to.G)*t), uint8(float64(from.B)*(1-t) + float64(to.B)*t), 255}
		draw.Draw(dst, image.Rect(0, y, dst.Bounds().Dx(), y+1), &image.Uniform{c}, image.Point{}, draw.Src)
	}
}

func fillRoundedRect(dst *image.RGBA, rect image.Rectangle, radius int, c color.RGBA) {
	mask := image.NewAlpha(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			dx := max(radius-x, x-(rect.Dx()-radius-1), 0)
			dy := max(radius-y, y-(rect.Dy()-radius-1), 0)
			if dx*dx+dy*dy <= radius*radius {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			}
		}
	}
	draw.DrawMask(dst, rect, &image.Uniform{c}, image.Point{}, mask, image.Point{}, draw.Over)
}

func fillCircle(dst *image.RGBA, cx, cy, radius int, c color.RGBA) {
	rect := image.Rect(cx-radius, cy-radius, cx+radius, cy+radius)
	mask := circleMask(rect.Dx(), rect.Dy())
	draw.DrawMask(dst, rect, &image.Uniform{c}, image.Point{}, mask, image.Point{}, draw.Over)
}

func circleMask(w, h int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	cx, cy := float64(w-1)/2, float64(h-1)/2
	r := float64(min(w, h)) / 2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			if dx*dx+dy*dy <= r*r {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			}
		}
	}
	return mask
}

func centerCrop(src image.Image) image.Rectangle {
	b := src.Bounds()
	side := min(b.Dx(), b.Dy())
	x := b.Min.X + (b.Dx()-side)/2
	y := b.Min.Y + (b.Dy()-side)/2
	return image.Rect(x, y, x+side, y+side)
}

func drawText(dst draw.Image, face font.Face, text string, x, baseline int, c color.Color) {
	d := &font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, baseline)}
	d.DrawString(text)
}

func drawCenteredText(dst draw.Image, face font.Face, text string, centerX, baseline int, c color.Color) {
	w := font.MeasureString(face, text).Ceil()
	drawText(dst, face, text, centerX-w/2, baseline, c)
}

func fitText(face font.Face, text string, maxWidth int) string {
	text = strings.TrimSpace(text)
	if font.MeasureString(face, text).Ceil() <= maxWidth {
		return text
	}
	runes := []rune(text)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimSpace(string(runes)) + "…"
		if font.MeasureString(face, candidate).Ceil() <= maxWidth {
			return candidate
		}
	}
	return "…"
}

func firstRune(s string) string {
	for _, r := range strings.TrimSpace(s) {
		return string(r)
	}
	return "?"
}
