package bot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

const (
	colorPrimary = 0x5865F2
	colorSuccess = 0x57F287
	colorDanger  = 0xED4245
	colorWarning = 0xFEE75C
	colorNeutral = 0x2B2D31
)

func trunc(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}
func embed(title, desc string, color int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{Title: title, Description: desc, Color: color, Timestamp: time.Now().Format(time.RFC3339)}
}
func errorEmbed(desc string) *discordgo.MessageEmbed {
	return embed("❌ İşlem Başarısız", desc, colorDanger)
}
func successEmbed(title, desc string) *discordgo.MessageEmbed {
	return embed(title, desc, colorSuccess)
}
func button(id, label string, style discordgo.ButtonStyle, emoji string) discordgo.Button {
	b := discordgo.Button{CustomID: id, Label: label, Style: style}
	if emoji != "" {
		b.Emoji = &discordgo.ComponentEmoji{Name: emoji}
	}
	return b
}
func row(parts ...discordgo.MessageComponent) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: parts}
}
func mentionID(s string) string                 { r := regexp.MustCompile(`\d{17,20}`).FindString(s); return r }
func hasPerm(m *discordgo.Member, p int64) bool { return m != nil && (m.Permissions&p) != 0 }
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "0" {
		return 0, nil
	}
	re := regexp.MustCompile(`^(\d+)(s|m|h|d|w)$`)
	x := re.FindStringSubmatch(s)
	if x == nil {
		return 0, fmt.Errorf("süre 10s, 5m, 2h veya 3d biçiminde olmalı")
	}
	n, _ := strconv.ParseInt(x[1], 10, 64)
	unit := map[string]time.Duration{"s": time.Second, "m": time.Minute, "h": time.Hour, "d": 24 * time.Hour, "w": 7 * 24 * time.Hour}[x[2]]
	return time.Duration(n) * unit, nil
}
func formatDuration(d time.Duration) string {
	if d%(24*time.Hour) == 0 {
		return fmt.Sprintf("%d gün", d/(24*time.Hour))
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%d saat", d/time.Hour)
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%d dakika", d/time.Minute)
	}
	return fmt.Sprintf("%d saniye", d/time.Second)
}
func boolIcon(v bool) string {
	if v {
		return "🟢 Açık"
	}
	return "🔴 Kapalı"
}
func str(v bool, a, b string) string {
	if v {
		return a
	}
	return b
}
func safeText(s string) string {
	s = strings.ReplaceAll(s, "@everyone", "@\u200beveryone")
	s = strings.ReplaceAll(s, "@here", "@\u200bhere")
	s = strings.ReplaceAll(s, "<@&", "<@\u200b&")
	return s
}
