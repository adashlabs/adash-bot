package database

import (
	"database/sql"
)

type InviteStats struct {
	GuildID string
	UserID  string
	Regular int
	Left    int
	Fake    int
	Bonus   int
}

func (s InviteStats) Net() int {
	n := s.Regular - s.Left - s.Fake + s.Bonus
	if n < 0 {
		return 0
	}
	return n
}

func (d *DB) GetInviteStats(guildID, userID string) (InviteStats, error) {
	var s InviteStats
	s.GuildID = guildID
	s.UserID = userID
	err := d.sql.QueryRow(`SELECT regular, "left", fake, bonus FROM invites WHERE guild_id=? AND user_id=?`, guildID, userID).
		Scan(&s.Regular, &s.Left, &s.Fake, &s.Bonus)
	if err == sql.ErrNoRows {
		return s, nil
	}
	return s, err
}

func (d *DB) UserNetInvites(guildID, userID string) int {
	stats, err := d.GetInviteStats(guildID, userID)
	if err != nil {
		return 0
	}
	return stats.Net()
}

func (d *DB) RecordMemberJoin(guildID, memberID, inviterID, code string, isFake bool) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	fakeVal := 0
	if isFake {
		fakeVal = 1
	}

	_, err = tx.Exec(`INSERT INTO invited_members(guild_id, member_id, inviter_id, code, joined_at, is_fake)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT(guild_id, member_id) DO UPDATE SET inviter_id=excluded.inviter_id, code=excluded.code, joined_at=excluded.joined_at, is_fake=excluded.is_fake`,
		guildID, memberID, inviterID, code, now(), fakeVal)
	if err != nil {
		return err
	}

	if isFake {
		_, err = tx.Exec(`INSERT INTO invites(guild_id, user_id, regular, "left", fake, bonus)
			VALUES(?,?,0,0,1,0)
			ON CONFLICT(guild_id, user_id) DO UPDATE SET fake=fake+1`, guildID, inviterID)
	} else {
		_, err = tx.Exec(`INSERT INTO invites(guild_id, user_id, regular, "left", fake, bonus)
			VALUES(?,?,1,0,0,0)
			ON CONFLICT(guild_id, user_id) DO UPDATE SET regular=regular+1`, guildID, inviterID)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *DB) RecordMemberLeave(guildID, memberID string) (string, error) {
	var inviterID string
	err := d.sql.QueryRow(`SELECT inviter_id FROM invited_members WHERE guild_id=? AND member_id=?`, guildID, memberID).Scan(&inviterID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	if inviterID != "" {
		_, err = d.sql.Exec(`INSERT INTO invites(guild_id, user_id, regular, "left", fake, bonus)
			VALUES(?,?,0,1,0,0)
			ON CONFLICT(guild_id, user_id) DO UPDATE SET "left"="left"+1`, guildID, inviterID)
		if err != nil {
			return inviterID, err
		}
	}

	return inviterID, nil
}

func (d *DB) AddBonusInvites(guildID, userID string, amount int) (int, error) {
	_, err := d.sql.Exec(`INSERT INTO invites(guild_id, user_id, regular, "left", fake, bonus)
		VALUES(?,?,0,0,0,?)
		ON CONFLICT(guild_id, user_id) DO UPDATE SET bonus=bonus+?`, guildID, userID, amount, amount)
	if err != nil {
		return 0, err
	}
	return d.UserNetInvites(guildID, userID), nil
}

func (d *DB) TopInviters(guildID string, limit int) ([]InviteStats, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := d.sql.Query(`SELECT guild_id, user_id, regular, "left", fake, bonus, (regular - "left" - fake + bonus) as net
		FROM invites
		WHERE guild_id=? AND (regular > 0 OR bonus > 0)
		ORDER BY net DESC, regular DESC
		LIMIT ?`, guildID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []InviteStats
	for rows.Next() {
		var s InviteStats
		var net int
		if err := rows.Scan(&s.GuildID, &s.UserID, &s.Regular, &s.Left, &s.Fake, &s.Bonus, &net); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
