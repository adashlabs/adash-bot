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

	// 1. Discord timestamp formatı: <t:1725900000:R>, <t:1725900000:F> veya <t:1725900000>
	tsRegex := regexp.MustCompile(`^<t:(\d{10,13})(?::[a-zA-Z])?>$`)
	if m := tsRegex.FindStringSubmatch(s); m != nil {
		rawTs, _ := strconv.ParseInt(m[1], 10, 64)
		if len(m[1]) == 13 {
			rawTs /= 1000
		}
		target := time.Unix(rawTs, 0)
		diff := time.Until(target)
		if diff <= 0 {
			return 0, fmt.Errorf("belirtilen unix zamanı geçmişte kalmış")
		}
		return diff, nil
	}

	// 2. Ham Unix Timestamp: 10 haneli saniye (1725900000) veya 13 haneli milisaniye
	if rawNumRegex := regexp.MustCompile(`^\d{10,13}$`); rawNumRegex.MatchString(s) {
		rawTs, _ := strconv.ParseInt(s, 10, 64)
		if len(s) == 13 {
			rawTs /= 1000
		}
		if rawTs >= 1577836800 && rawTs <= 2524608000 {
			target := time.Unix(rawTs, 0)
			diff := time.Until(target)
			if diff <= 0 {
				return 0, fmt.Errorf("belirtilen unix zamanı geçmişte kalmış")
			}
			return diff, nil
		}
	}

	// 3. Klasik ve bileşik süreler: 10s, 5m, 2h, 3d, 1w, 1d12h, 2h30m vb.
	clean := strings.ReplaceAll(s, " ", "")
	re := regexp.MustCompile(`(\d+)(s|m|h|d|w)`)
	matches := re.FindAllStringSubmatch(clean, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("süre 10m, 2h, 3d veya unix timestamp (örn: <t:1725900000:R>) biçiminde olmalı")
	}

	reconstruct := ""
	var total time.Duration
	unitMap := map[string]time.Duration{
		"s": time.Second,
		"m": time.Minute,
		"h": time.Hour,
		"d": 24 * time.Hour,
		"w": 7 * 24 * time.Hour,
	}
	for _, match := range matches {
		reconstruct += match[0]
		n, _ := strconv.ParseInt(match[1], 10, 64)
		total += time.Duration(n) * unitMap[match[2]]
	}
	if reconstruct != clean {
		return 0, fmt.Errorf("süre 10m, 2h, 3d veya unix timestamp biçiminde olmalı")
	}

	return total, nil
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
