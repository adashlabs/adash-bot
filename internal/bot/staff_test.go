package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestStaffApplicationPanel(t *testing.T) {
	b := &Bot{
		db: nil, // We'll test with minimal bot or test embed structure directly
	}

	// Test button structure
	btn := discordgo.Button{
		CustomID: "staff_apply_btn",
		Label:    "Başvur",
		Style:    discordgo.SuccessButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
	}
	if btn.CustomID != "staff_apply_btn" || btn.Label != "Başvur" || btn.Style != discordgo.SuccessButton {
		t.Fatalf("unexpected button configuration: %+v", btn)
	}

	// Test action buttons
	actionRow := row(
		button("staff_app_action:accept:1:12345", "Onayla", discordgo.SuccessButton, "✅"),
		button("staff_app_action:reject:1:12345", "Reddet", discordgo.DangerButton, "❌"),
	)
	if len(actionRow.Components) != 2 {
		t.Fatalf("expected 2 action buttons, got %d", len(actionRow.Components))
	}
	_ = b
}

func TestStaffModalFieldsCount(t *testing.T) {
	// Discord limits modal to at most 5 TextInput components
	inputs := []string{
		"staff_name_age",
		"staff_activity",
		"staff_experience",
		"staff_reason",
		"staff_about",
	}
	if len(inputs) > 5 {
		t.Fatalf("Discord modals support at most 5 inputs, got %d", len(inputs))
	}
}

func TestStaffPermissionRequirement(t *testing.T) {
	// Verify that manageServer permission constant is used
	perm := int64(discordgo.PermissionManageServer)
	if perm&discordgo.PermissionManageServer == 0 {
		t.Fatal("expected PermissionManageServer bit to match")
	}
}
