//go:build cgo

package database

import (
	"path/filepath"
	"testing"
)

func TestStaffApplicationLifecycle(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "staff_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	guildID := "123456789012345678"
	userID := "987654321098765432"

	// Initially no pending application
	hasPending, err := db.HasPendingStaffApplication(guildID, userID)
	if err != nil {
		t.Fatalf("HasPendingStaffApplication error: %v", err)
	}
	if hasPending {
		t.Fatal("expected no pending application initially")
	}

	// Create application
	appID, err := db.CreateStaffApplication(guildID, userID, "Can, 20", "Günde 5 saat", "Adash Sunucusu Mod", "Sunucuya katkı sağlamak", "Eklemek istediğim bir şey yok")
	if err != nil {
		t.Fatalf("CreateStaffApplication error: %v", err)
	}
	if appID <= 0 {
		t.Fatalf("expected positive appID, got %d", appID)
	}

	// Now pending should be true
	hasPending, err = db.HasPendingStaffApplication(guildID, userID)
	if err != nil || !hasPending {
		t.Fatalf("expected pending application, got %v, err: %v", hasPending, err)
	}

	// Fetch application by ID
	app, err := db.StaffApplicationByID(appID)
	if err != nil {
		t.Fatalf("StaffApplicationByID error: %v", err)
	}
	if app.NameAge != "Can, 20" || app.Status != "pending" {
		t.Fatalf("unexpected app data: %+v", app)
	}

	// Review application - Accept
	reviewerID := "111222333444555666"
	ok, err := db.ReviewStaffApplication(appID, "accepted", reviewerID, "")
	if err != nil {
		t.Fatalf("ReviewStaffApplication error: %v", err)
	}
	if !ok {
		t.Fatal("expected review to succeed")
	}

	// Now hasPending should be false again
	hasPending, err = db.HasPendingStaffApplication(guildID, userID)
	if err != nil || hasPending {
		t.Fatalf("expected no pending application after review, got %v", hasPending)
	}

	// Verify updated app status
	app, err = db.StaffApplicationByID(appID)
	if err != nil {
		t.Fatalf("StaffApplicationByID error: %v", err)
	}
	if app.Status != "accepted" || !app.ReviewerID.Valid || app.ReviewerID.String != reviewerID {
		t.Fatalf("app review fields not set properly: %+v", app)
	}

	// Reviewing already reviewed app should return false
	ok, err = db.ReviewStaffApplication(appID, "rejected", reviewerID, "geçersiz")
	if err != nil || ok {
		t.Fatalf("expected already reviewed application update to return false, got ok=%v, err=%v", ok, err)
	}

	// List applications
	list, err := db.GuildStaffApplications(guildID, 10)
	if err != nil {
		t.Fatalf("GuildStaffApplications error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 application in list, got %d", len(list))
	}
}
