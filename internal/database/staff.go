package database

import (
	"database/sql"
)

type StaffApplication struct {
	ID           int64
	GuildID      string
	UserID       string
	NameAge      string
	Activity     string
	Experience   string
	Reason       string
	About        string
	Status       string
	ReviewerID   sql.NullString
	ReviewReason sql.NullString
	CreatedAt    int64
	ReviewedAt   sql.NullInt64
}

const staffAppCols = `id,guild_id,user_id,name_age,activity,experience,reason,about,status,reviewer_id,review_reason,created_at,reviewed_at`

func scanStaffApplication(row interface{ Scan(...any) error }) (StaffApplication, error) {
	var a StaffApplication
	err := row.Scan(&a.ID, &a.GuildID, &a.UserID, &a.NameAge, &a.Activity, &a.Experience, &a.Reason, &a.About, &a.Status, &a.ReviewerID, &a.ReviewReason, &a.CreatedAt, &a.ReviewedAt)
	return a, err
}

func (d *DB) CreateStaffApplication(guildID, userID, nameAge, activity, experience, reason, about string) (int64, error) {
	r, err := d.sql.Exec(`INSERT INTO staff_applications(guild_id,user_id,name_age,activity,experience,reason,about,status,created_at) VALUES(?,?,?,?,?,?,?,'pending',?)`,
		guildID, userID, nameAge, activity, experience, reason, about, now())
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d *DB) HasPendingStaffApplication(guildID, userID string) (bool, error) {
	var id int64
	err := d.sql.QueryRow(`SELECT id FROM staff_applications WHERE guild_id=? AND user_id=? AND status='pending' LIMIT 1`, guildID, userID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *DB) StaffApplicationByID(id int64) (StaffApplication, error) {
	return scanStaffApplication(d.sql.QueryRow(`SELECT `+staffAppCols+` FROM staff_applications WHERE id=?`, id))
}

func (d *DB) ReviewStaffApplication(id int64, status, reviewerID, reason string) (bool, error) {
	r, err := d.sql.Exec(`UPDATE staff_applications SET status=?,reviewer_id=?,review_reason=?,reviewed_at=? WHERE id=? AND status='pending'`,
		status, reviewerID, null(reason), now(), id)
	if err != nil {
		return false, err
	}
	n, _ := r.RowsAffected()
	return n == 1, nil
}

func (d *DB) GuildStaffApplications(guildID string, limit int) ([]StaffApplication, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := d.sql.Query(`SELECT `+staffAppCols+` FROM staff_applications WHERE guild_id=? ORDER BY id DESC LIMIT ?`, guildID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StaffApplication
	for rows.Next() {
		a, err := scanStaffApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
