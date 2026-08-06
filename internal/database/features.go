package database

import "database/sql"

func scanTicket(row interface{ Scan(...any) error }) (Ticket, error) {
	var t Ticket
	e := row.Scan(&t.ChannelID, &t.GuildID, &t.OwnerID, &t.CategoryID, &t.ClaimedByID, &t.Type, &t.Priority, &t.Subject, &t.Description, &t.Status, &t.OpenedAt, &t.ClosedAt, &t.ClosedByID, &t.CloseReason)
	return t, e
}

const ticketCols = `channel_id,guild_id,owner_id,category_id,claimed_by_id,ticket_type,priority,subject,description,status,opened_at,closed_at,closed_by_id,close_reason`

func (d *DB) OpenTicket(guild, owner string) (Ticket, error) {
	return scanTicket(d.sql.QueryRow(`SELECT `+ticketCols+` FROM tickets WHERE guild_id=? AND owner_id=? AND closed_at IS NULL LIMIT 1`, guild, owner))
}
func (d *DB) Ticket(channel string) (Ticket, error) {
	return scanTicket(d.sql.QueryRow(`SELECT `+ticketCols+` FROM tickets WHERE channel_id=?`, channel))
}
func (d *DB) CreateTicket(channel, guild, owner, category, typ, priority, subject, description string) error {
	_, e := d.sql.Exec(`INSERT INTO tickets(channel_id,guild_id,owner_id,category_id,ticket_type,priority,subject,description,opened_at) VALUES(?,?,?,?,?,?,?,?,?)`, channel, guild, owner, null(category), value(typ, "destek"), value(priority, "normal"), subject, description, now())
	return e
}
func (d *DB) ClaimTicket(channel, user string) (bool, error) {
	r, e := d.sql.Exec(`UPDATE tickets SET claimed_by_id=?,status='claimed' WHERE channel_id=? AND closed_at IS NULL AND claimed_by_id IS NULL`, user, channel)
	if e != nil {
		return false, e
	}
	n, _ := r.RowsAffected()
	return n == 1, nil
}
func (d *DB) SetTicketStatus(channel, status string) (bool, error) {
	r, e := d.sql.Exec(`UPDATE tickets SET status=? WHERE channel_id=? AND closed_at IS NULL`, status, channel)
	if e != nil {
		return false, e
	}
	n, _ := r.RowsAffected()
	return n == 1, nil
}
func (d *DB) CloseTicket(channel, user, reason string) (bool, error) {
	r, e := d.sql.Exec(`UPDATE tickets SET closed_at=?,closed_by_id=?,close_reason=?,status='closed' WHERE channel_id=? AND closed_at IS NULL`, now(), user, value(reason, "Sebep belirtilmedi"), channel)
	if e != nil {
		return false, e
	}
	n, _ := r.RowsAffected()
	return n == 1, nil
}

func scanGiveaway(row interface{ Scan(...any) error }) (Giveaway, error) {
	var g Giveaway
	e := row.Scan(&g.ID, &g.GuildID, &g.ChannelID, &g.MessageID, &g.HostID, &g.Prize, &g.WinnerCount, &g.RequiredRoleID, &g.MinAccountAgeDays, &g.EndsAt, &g.EndedAt)
	return g, e
}

const giveawayCols = `id,guild_id,channel_id,message_id,host_id,prize,winner_count,required_role_id,min_account_age_days,ends_at,ended_at`

func (d *DB) CreateGiveaway(guild, channel, message, host, prize string, winners int, role string, minDays int, ends int64) (int64, error) {
	r, e := d.sql.Exec(`INSERT INTO giveaways(guild_id,channel_id,message_id,host_id,prize,winner_count,required_role_id,min_account_age_days,ends_at) VALUES(?,?,?,?,?,?,?,?,?)`, guild, channel, message, host, prize, winners, null(role), minDays, ends)
	if e != nil {
		return 0, e
	}
	return r.LastInsertId()
}
func (d *DB) GiveawayByMessage(message string) (Giveaway, error) {
	return scanGiveaway(d.sql.QueryRow(`SELECT `+giveawayCols+` FROM giveaways WHERE message_id=?`, message))
}
func (d *DB) GiveawayByID(id int64) (Giveaway, error) {
	return scanGiveaway(d.sql.QueryRow(`SELECT `+giveawayCols+` FROM giveaways WHERE id=?`, id))
}
func (d *DB) ActiveGiveaways() (out []Giveaway, e error) {
	rows, e := d.sql.Query(`SELECT ` + giveawayCols + ` FROM giveaways WHERE ended_at IS NULL`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		g, x := scanGiveaway(rows)
		if x != nil {
			return nil, x
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
func (d *DB) EndGiveaway(id int64) (bool, error) {
	r, e := d.sql.Exec(`UPDATE giveaways SET ended_at=? WHERE id=? AND ended_at IS NULL`, now(), id)
	if e != nil {
		return false, e
	}
	n, _ := r.RowsAffected()
	return n == 1, nil
}
func (d *DB) JoinGiveaway(id int64, user string) (bool, error) {
	r, e := d.sql.Exec(`INSERT OR IGNORE INTO giveaway_entries(giveaway_id,user_id) VALUES(?,?)`, id, user)
	if e != nil {
		return false, e
	}
	n, _ := r.RowsAffected()
	return n == 1, nil
}
func (d *DB) LeaveGiveaway(id int64, user string) error {
	_, e := d.sql.Exec(`DELETE FROM giveaway_entries WHERE giveaway_id=? AND user_id=?`, id, user)
	return e
}
func (d *DB) GiveawayEntries(id int64) ([]string, error) {
	rows, e := d.sql.Query(`SELECT user_id FROM giveaway_entries WHERE giveaway_id=?`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if e = rows.Scan(&s); e != nil {
			return nil, e
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (d *DB) TableCounts() (map[string]int64, error) {
	out := map[string]int64{}
	for _, t := range []string{"guilds", "users", "guild_settings", "command_logs", "mod_logs", "warnings", "game_states", "word_game_used", "guild_config", "tickets", "giveaways", "giveaway_entries"} {
		var n int64
		if e := d.sql.QueryRow(`SELECT COUNT(*) FROM ` + t).Scan(&n); e != nil {
			return nil, e
		}
		out[t] = n
	}
	return out, nil
}
func IsNotFound(err error) bool { return err == sql.ErrNoRows }
func value(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
