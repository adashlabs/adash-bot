package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct{ sql *sql.DB }
type Settings struct {
	GuildID                                                                                  string
	WelcomeChannelID, FarewellChannelID, AutoroleID, ModLogChannelID                         sql.NullString
	TicketCategoryID, TicketLogChannelID, CountingChannelID, WordChainChannelID, AIChannelID sql.NullString
	WelcomeEnabled, FarewellEnabled, CountingEnabled, WordChainEnabled                       bool
	WelcomeMessage, FarewellMessage                                                          string
}
type GameState struct {
	CountingValue                        int64
	CountingUserID, LastWord, WordUserID sql.NullString
}
type Warning struct {
	ID                                   int64
	GuildID, UserID, ModeratorID, Reason string
	Active                               bool
	CreatedAt                            int64
	RemovedAt                            sql.NullInt64
}
type ModLog struct {
	ID                                   int64
	GuildID, UserID, ModeratorID, Action string
	Reason                               sql.NullString
	Duration                             sql.NullInt64
	Timestamp                            int64
}
type Ticket struct {
	ChannelID, GuildID, OwnerID                  string
	CategoryID, ClaimedByID                      sql.NullString
	Type, Priority, Subject, Description, Status string
	OpenedAt                                     int64
	ClosedAt                                     sql.NullInt64
	ClosedByID, CloseReason                      sql.NullString
}
type Giveaway struct {
	ID                                           int64
	GuildID, ChannelID, MessageID, HostID, Prize string
	WinnerCount                                  int
	RequiredRoleID                               sql.NullString
	MinAccountAgeDays                            int
	EndsAt                                       int64
	EndedAt                                      sql.NullInt64
}

const schema = `
CREATE TABLE IF NOT EXISTS guilds (guild_id TEXT PRIMARY KEY,name TEXT NOT NULL,joined_at INTEGER NOT NULL,prefix TEXT NOT NULL DEFAULT 'a!');
CREATE TABLE IF NOT EXISTS users (user_id TEXT PRIMARY KEY,username TEXT NOT NULL,discriminator TEXT,first_seen INTEGER NOT NULL,last_seen INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS guild_settings (guild_id TEXT PRIMARY KEY,welcome_channel_id TEXT,farewell_channel_id TEXT,autorole_id TEXT,mod_log_channel_id TEXT,ticket_category_id TEXT,ticket_log_channel_id TEXT,counting_channel_id TEXT,word_chain_channel_id TEXT,ai_channel_id TEXT,welcome_enabled INTEGER NOT NULL DEFAULT 0,farewell_enabled INTEGER NOT NULL DEFAULT 0,counting_enabled INTEGER NOT NULL DEFAULT 0,word_chain_enabled INTEGER NOT NULL DEFAULT 0,welcome_message TEXT NOT NULL DEFAULT '{user}, {server} sunucusuna hoş geldin!',farewell_message TEXT NOT NULL DEFAULT '{user} aramızdan ayrıldı. Görüşmek üzere!',updated_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS command_logs (id INTEGER PRIMARY KEY AUTOINCREMENT,guild_id TEXT,user_id TEXT NOT NULL,command TEXT NOT NULL,args TEXT,timestamp INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS mod_logs (id INTEGER PRIMARY KEY AUTOINCREMENT,guild_id TEXT NOT NULL,user_id TEXT NOT NULL,moderator_id TEXT NOT NULL,action TEXT NOT NULL,reason TEXT,duration INTEGER,timestamp INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS warnings (id INTEGER PRIMARY KEY AUTOINCREMENT,guild_id TEXT NOT NULL,user_id TEXT NOT NULL,moderator_id TEXT NOT NULL,reason TEXT NOT NULL,active INTEGER NOT NULL DEFAULT 1,created_at INTEGER NOT NULL,removed_at INTEGER);
CREATE TABLE IF NOT EXISTS game_states (guild_id TEXT PRIMARY KEY,counting_value INTEGER NOT NULL DEFAULT 0,counting_user_id TEXT,last_word TEXT,word_user_id TEXT,updated_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS word_game_used (guild_id TEXT NOT NULL,word TEXT NOT NULL,PRIMARY KEY(guild_id,word));
CREATE TABLE IF NOT EXISTS guild_config (guild_id TEXT NOT NULL,key TEXT NOT NULL,value TEXT NOT NULL,updated_at INTEGER NOT NULL,PRIMARY KEY(guild_id,key));
CREATE TABLE IF NOT EXISTS tickets (channel_id TEXT PRIMARY KEY,guild_id TEXT NOT NULL,owner_id TEXT NOT NULL,category_id TEXT,claimed_by_id TEXT,ticket_type TEXT NOT NULL DEFAULT 'destek',priority TEXT NOT NULL DEFAULT 'normal',subject TEXT NOT NULL DEFAULT '',description TEXT NOT NULL DEFAULT '',status TEXT NOT NULL DEFAULT 'open',opened_at INTEGER NOT NULL,closed_at INTEGER,closed_by_id TEXT,close_reason TEXT);
CREATE TABLE IF NOT EXISTS giveaways (id INTEGER PRIMARY KEY AUTOINCREMENT,guild_id TEXT NOT NULL,channel_id TEXT NOT NULL,message_id TEXT NOT NULL UNIQUE,host_id TEXT NOT NULL,prize TEXT NOT NULL,winner_count INTEGER NOT NULL,required_role_id TEXT,min_account_age_days INTEGER NOT NULL DEFAULT 0,ends_at INTEGER NOT NULL,ended_at INTEGER);
CREATE TABLE IF NOT EXISTS giveaway_entries (giveaway_id INTEGER NOT NULL,user_id TEXT NOT NULL,PRIMARY KEY(giveaway_id,user_id),FOREIGN KEY(giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE);
CREATE INDEX IF NOT EXISTS idx_tickets_guild_owner ON tickets(guild_id,owner_id,closed_at); CREATE INDEX IF NOT EXISTS idx_giveaways_due ON giveaways(ended_at,ends_at); CREATE INDEX IF NOT EXISTS idx_cmdlogs_guild ON command_logs(guild_id); CREATE INDEX IF NOT EXISTS idx_cmdlogs_user ON command_logs(user_id); CREATE INDEX IF NOT EXISTS idx_cmdlogs_time ON command_logs(timestamp); CREATE INDEX IF NOT EXISTS idx_modlogs_guild ON mod_logs(guild_id); CREATE INDEX IF NOT EXISTS idx_modlogs_user ON mod_logs(user_id); CREATE INDEX IF NOT EXISTS idx_modlogs_time ON mod_logs(timestamp); CREATE INDEX IF NOT EXISTS idx_warnings_user ON warnings(guild_id,user_id,active);`

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	dsn := path + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-2000"
	s, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	s.SetMaxOpenConns(1)
	s.SetMaxIdleConns(1)
	s.SetConnMaxLifetime(0)
	if _, err = s.Exec(schema); err != nil {
		s.Close()
		return nil, err
	}
	d := &DB{sql: s}
	for table, columns := range map[string]map[string]string{
		"guild_settings": {"ticket_category_id": "TEXT", "ticket_log_channel_id": "TEXT", "ai_channel_id": "TEXT"},
		"tickets":        {"claimed_by_id": "TEXT", "ticket_type": "TEXT NOT NULL DEFAULT 'destek'", "priority": "TEXT NOT NULL DEFAULT 'normal'", "subject": "TEXT NOT NULL DEFAULT ''", "description": "TEXT NOT NULL DEFAULT ''", "status": "TEXT NOT NULL DEFAULT 'open'", "close_reason": "TEXT"},
		"giveaways":      {"required_role_id": "TEXT", "min_account_age_days": "INTEGER NOT NULL DEFAULT 0"},
	} {
		for col, typ := range columns {
			if err := d.ensureColumn(table, col, typ); err != nil {
				s.Close()
				return nil, err
			}
		}
	}
	return d, nil
}
func (d *DB) ensureColumn(table, col, typ string) error {
	rows, e := d.sql.Query("PRAGMA table_info(" + table + ")")
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var n, t string
		var nn, pk int
		var def any
		if e = rows.Scan(&cid, &n, &t, &nn, &def, &pk); e != nil {
			return e
		}
		if n == col {
			return nil
		}
	}
	if e = rows.Err(); e != nil {
		return e
	}
	if e = rows.Close(); e != nil {
		return e
	}
	_, e = d.sql.Exec("ALTER TABLE " + table + " ADD COLUMN " + col + " " + typ)
	return e
}
func now() int64           { return time.Now().UnixMilli() }
func (d *DB) Close() error { return d.sql.Close() }
func (d *DB) Integrity() error {
	var result string
	if e := d.sql.QueryRow("PRAGMA integrity_check").Scan(&result); e != nil {
		return e
	}
	if result != "ok" {
		return fmt.Errorf("sqlite integrity_check: %s", result)
	}
	return nil
}
func (d *DB) RegisterGuild(id, name string) error {
	_, e := d.sql.Exec(`INSERT INTO guilds(guild_id,name,joined_at) VALUES(?,?,?) ON CONFLICT(guild_id) DO UPDATE SET name=excluded.name`, id, name, now())
	if e == nil {
		e = d.ensureGuild(id)
	}
	return e
}
func (d *DB) RegisterUser(id, name, disc string) error {
	var v any = disc
	if disc == "" || disc == "0" {
		v = nil
	}
	n := now()
	_, e := d.sql.Exec(`INSERT INTO users(user_id,username,discriminator,first_seen,last_seen) VALUES(?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET username=excluded.username,discriminator=excluded.discriminator,last_seen=excluded.last_seen`, id, name, v, n, n)
	return e
}
func (d *DB) ensureGuild(id string) error {
	n := now()
	if _, e := d.sql.Exec(`INSERT OR IGNORE INTO guild_settings(guild_id,updated_at) VALUES(?,?)`, id, n); e != nil {
		return e
	}
	_, e := d.sql.Exec(`INSERT OR IGNORE INTO game_states(guild_id,updated_at) VALUES(?,?)`, id, n)
	return e
}
func (d *DB) Prefix(id string) string {
	var p string
	if d.sql.QueryRow(`SELECT prefix FROM guilds WHERE guild_id=?`, id).Scan(&p) != nil {
		return "a!"
	}
	return p
}
func (d *DB) SetPrefix(id, p string) error {
	_, e := d.sql.Exec(`UPDATE guilds SET prefix=? WHERE guild_id=?`, p, id)
	return e
}
func (d *DB) Settings(id string) (Settings, error) {
	if e := d.ensureGuild(id); e != nil {
		return Settings{}, e
	}
	var s Settings
	var we, fe, ce, wce int
	e := d.sql.QueryRow(`SELECT guild_id,welcome_channel_id,farewell_channel_id,autorole_id,mod_log_channel_id,ticket_category_id,ticket_log_channel_id,counting_channel_id,word_chain_channel_id,ai_channel_id,welcome_enabled,farewell_enabled,counting_enabled,word_chain_enabled,welcome_message,farewell_message FROM guild_settings WHERE guild_id=?`, id).Scan(&s.GuildID, &s.WelcomeChannelID, &s.FarewellChannelID, &s.AutoroleID, &s.ModLogChannelID, &s.TicketCategoryID, &s.TicketLogChannelID, &s.CountingChannelID, &s.WordChainChannelID, &s.AIChannelID, &we, &fe, &ce, &wce, &s.WelcomeMessage, &s.FarewellMessage)
	s.WelcomeEnabled = we != 0
	s.FarewellEnabled = fe != 0
	s.CountingEnabled = ce != 0
	s.WordChainEnabled = wce != 0
	return s, e
}

var settingsColumns = map[string]bool{"welcome_channel_id": true, "farewell_channel_id": true, "autorole_id": true, "mod_log_channel_id": true, "counting_channel_id": true, "word_chain_channel_id": true, "ticket_category_id": true, "ticket_log_channel_id": true, "welcome_enabled": true, "farewell_enabled": true, "counting_enabled": true, "word_chain_enabled": true, "welcome_message": true, "farewell_message": true, "ai_channel_id": true}

func (d *DB) SetSetting(id, key string, value any) error {
	if !settingsColumns[key] {
		return fmt.Errorf("geçersiz ayar: %s", key)
	}
	if e := d.ensureGuild(id); e != nil {
		return e
	}
	_, e := d.sql.Exec(`UPDATE guild_settings SET `+key+`=?,updated_at=? WHERE guild_id=?`, value, now(), id)
	return e
}
func (d *DB) GetConfig(id, key string, fallback any) any {
	var raw string
	if d.sql.QueryRow(`SELECT value FROM guild_config WHERE guild_id=? AND key=?`, id, key).Scan(&raw) != nil {
		return fallback
	}
	var v any
	if json.Unmarshal([]byte(raw), &v) != nil {
		return fallback
	}
	return v
}
func (d *DB) ConfigString(id, key, fallback string) string {
	v := d.GetConfig(id, key, fallback)
	if s, ok := v.(string); ok {
		return s
	}
	return fallback
}
func (d *DB) ConfigBool(id, key string, fallback bool) bool {
	v := d.GetConfig(id, key, fallback)
	if b, ok := v.(bool); ok {
		return b
	}
	return fallback
}
func (d *DB) ConfigInt(id, key string, fallback int) int {
	v := d.GetConfig(id, key, float64(fallback))
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return fallback
}
func (d *DB) SetConfig(id, key string, value any) error {
	raw, e := json.Marshal(value)
	if e != nil {
		return e
	}
	_, e = d.sql.Exec(`INSERT INTO guild_config(guild_id,key,value,updated_at) VALUES(?,?,?,?) ON CONFLICT(guild_id,key) DO UPDATE SET value=excluded.value,updated_at=excluded.updated_at`, id, key, string(raw), now())
	return e
}
func (d *DB) LogCommand(guild, user, command, args string) {
	var g any = guild
	if guild == "" {
		g = nil
	}
	_, _ = d.sql.Exec(`INSERT INTO command_logs(guild_id,user_id,command,args,timestamp) VALUES(?,?,?,?,?)`, g, user, command, args, now())
}
func (d *DB) Stats() (guilds, users, commands int64) {
	_ = d.sql.QueryRow(`SELECT COUNT(*) FROM guilds`).Scan(&guilds)
	_ = d.sql.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
	_ = d.sql.QueryRow(`SELECT COUNT(*) FROM command_logs`).Scan(&commands)
	return
}
func (d *DB) LogMod(guild, user, moderator, action, reason string, duration *int64) error {
	_, e := d.sql.Exec(`INSERT INTO mod_logs(guild_id,user_id,moderator_id,action,reason,duration,timestamp) VALUES(?,?,?,?,?,?,?)`, guild, user, moderator, action, null(reason), duration, now())
	return e
}
func (d *DB) RecentModLogs(guild, user string, limit int) ([]ModLog, error) {
	q := `SELECT id,guild_id,user_id,moderator_id,action,reason,duration,timestamp FROM mod_logs WHERE guild_id=?`
	args := []any{guild}
	if user != "" {
		q += ` AND user_id=?`
		args = append(args, user)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, e := d.sql.Query(q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []ModLog
	for rows.Next() {
		var x ModLog
		if e = rows.Scan(&x.ID, &x.GuildID, &x.UserID, &x.ModeratorID, &x.Action, &x.Reason, &x.Duration, &x.Timestamp); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (d *DB) AddWarning(guild, user, moderator, reason string) (int64, int, error) {
	r, e := d.sql.Exec(`INSERT INTO warnings(guild_id,user_id,moderator_id,reason,created_at) VALUES(?,?,?,?,?)`, guild, user, moderator, reason, now())
	if e != nil {
		return 0, 0, e
	}
	id, _ := r.LastInsertId()
	c, _ := d.WarningCount(guild, user)
	return id, c, nil
}
func (d *DB) Warnings(guild, user string) ([]Warning, error) {
	rows, e := d.sql.Query(`SELECT id,guild_id,user_id,moderator_id,reason,active,created_at,removed_at FROM warnings WHERE guild_id=? AND user_id=? AND active=1 ORDER BY id DESC`, guild, user)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []Warning
	for rows.Next() {
		var x Warning
		if e = rows.Scan(&x.ID, &x.GuildID, &x.UserID, &x.ModeratorID, &x.Reason, &x.Active, &x.CreatedAt, &x.RemovedAt); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (d *DB) WarningCount(guild, user string) (int, error) {
	var c int
	e := d.sql.QueryRow(`SELECT COUNT(*) FROM warnings WHERE guild_id=? AND user_id=? AND active=1`, guild, user).Scan(&c)
	return c, e
}
func (d *DB) ClearWarnings(guild, user string) (int64, error) {
	r, e := d.sql.Exec(`UPDATE warnings SET active=0,removed_at=? WHERE guild_id=? AND user_id=? AND active=1`, now(), guild, user)
	if e != nil {
		return 0, e
	}
	return r.RowsAffected()
}
func (d *DB) Game(id string) (GameState, error) {
	if e := d.ensureGuild(id); e != nil {
		return GameState{}, e
	}
	var g GameState
	e := d.sql.QueryRow(`SELECT counting_value,counting_user_id,last_word,word_user_id FROM game_states WHERE guild_id=?`, id).Scan(&g.CountingValue, &g.CountingUserID, &g.LastWord, &g.WordUserID)
	return g, e
}
func (d *DB) SetCount(id string, value int64, user string) error {
	_, e := d.sql.Exec(`UPDATE game_states SET counting_value=?,counting_user_id=?,updated_at=? WHERE guild_id=?`, value, null(user), now(), id)
	return e
}
func (d *DB) SetWord(id, word, user string) error {
	tx, e := d.sql.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`UPDATE game_states SET last_word=?,word_user_id=?,updated_at=? WHERE guild_id=?`, null(word), null(user), now(), id); e != nil {
		return e
	}
	if word != "" {
		if _, e = tx.Exec(`INSERT INTO word_game_used(guild_id,word) VALUES(?,?)`, id, word); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (d *DB) UsedWord(id, word string) bool {
	var x int
	return d.sql.QueryRow(`SELECT 1 FROM word_game_used WHERE guild_id=? AND word=?`, id, word).Scan(&x) == nil
}
func (d *DB) ResetGame(id, game string) error {
	if game == "counting" {
		_, e := d.sql.Exec(`UPDATE game_states SET counting_value=0,counting_user_id=NULL,updated_at=? WHERE guild_id=?`, now(), id)
		return e
	}
	tx, e := d.sql.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`UPDATE game_states SET last_word=NULL,word_user_id=NULL,updated_at=? WHERE guild_id=?`, now(), id); e != nil {
		return e
	}
	if _, e = tx.Exec(`DELETE FROM word_game_used WHERE guild_id=?`, id); e != nil {
		return e
	}
	return tx.Commit()
}
func null(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
