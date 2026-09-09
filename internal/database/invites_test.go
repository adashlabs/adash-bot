//go:build cgo

package database

import (
	"path/filepath"
	"testing"
)

func TestInviteTrackingDatabase(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "invites_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	guildID := "1001"
	inviterID := "2001"
	member1 := "3001"
	member2 := "3002"

	// Check initial
	stats, err := db.GetInviteStats(guildID, inviterID)
	if err != nil {
		t.Fatalf("GetInviteStats error: %v", err)
	}
	if stats.Net() != 0 {
		t.Fatalf("expected 0 net invites, got %d", stats.Net())
	}

	// Member 1 joins via regular invite
	err = db.RecordMemberJoin(guildID, member1, inviterID, "CODE1", false)
	if err != nil {
		t.Fatalf("RecordMemberJoin error: %v", err)
	}

	stats, _ = db.GetInviteStats(guildID, inviterID)
	if stats.Regular != 1 || stats.Net() != 1 {
		t.Fatalf("expected regular=1, net=1, got regular=%d, net=%d", stats.Regular, stats.Net())
	}

	// Member 2 joins as fake (<3 days old account)
	err = db.RecordMemberJoin(guildID, member2, inviterID, "CODE1", true)
	if err != nil {
		t.Fatalf("RecordMemberJoin fake error: %v", err)
	}

	stats, _ = db.GetInviteStats(guildID, inviterID)
	if stats.Fake != 1 || stats.Regular != 1 || stats.Net() != 1 {
		t.Fatalf("expected fake=1, regular=1, net=1, got %+v", stats)
	}

	// Member 1 leaves
	inviter, err := db.RecordMemberLeave(guildID, member1)
	if err != nil || inviter != inviterID {
		t.Fatalf("RecordMemberLeave error: %v, inviter: %s", err, inviter)
	}

	stats, _ = db.GetInviteStats(guildID, inviterID)
	if stats.Left != 1 || stats.Net() != 0 {
		t.Fatalf("expected left=1, net=0, got regular=%d, left=%d, net=%d", stats.Regular, stats.Left, stats.Net())
	}

	// Add bonus invites
	net, err := db.AddBonusInvites(guildID, inviterID, 3)
	if err != nil {
		t.Fatalf("AddBonusInvites error: %v", err)
	}
	if net != 3 {
		t.Fatalf("expected net=3 after bonus, got %d", net)
	}

	// Top inviters
	top, err := db.TopInviters(guildID, 5)
	if err != nil {
		t.Fatalf("TopInviters error: %v", err)
	}
	if len(top) != 1 || top[0].UserID != inviterID {
		t.Fatalf("unexpected top inviters: %+v", top)
	}
}
