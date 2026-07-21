const Database = require('better-sqlite3');
const path = require('node:path');
const fs = require('node:fs');

const dataDir = path.join(__dirname, '..', 'data');
if (!fs.existsSync(dataDir)) {
  fs.mkdirSync(dataDir, { recursive: true });
}

const dbPath = path.join(dataDir, 'adash.db');
const db = new Database(dbPath);

db.pragma('journal_mode = WAL');
db.pragma('foreign_keys = ON');

db.exec(`
  CREATE TABLE IF NOT EXISTS guilds (
    guild_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    joined_at INTEGER NOT NULL,
    prefix TEXT NOT NULL DEFAULT 'a!'
  );

  CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    discriminator TEXT,
    first_seen INTEGER NOT NULL,
    last_seen INTEGER NOT NULL
  );

  CREATE TABLE IF NOT EXISTS command_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT,
    user_id TEXT NOT NULL,
    command TEXT NOT NULL,
    args TEXT,
    timestamp INTEGER NOT NULL
  );

  CREATE TABLE IF NOT EXISTS mod_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    moderator_id TEXT NOT NULL,
    action TEXT NOT NULL,
    reason TEXT,
    duration INTEGER,
    timestamp INTEGER NOT NULL
  );

  CREATE INDEX IF NOT EXISTS idx_cmdlogs_guild ON command_logs(guild_id);
  CREATE INDEX IF NOT EXISTS idx_cmdlogs_user ON command_logs(user_id);
  CREATE INDEX IF NOT EXISTS idx_cmdlogs_time ON command_logs(timestamp);
  CREATE INDEX IF NOT EXISTS idx_modlogs_guild ON mod_logs(guild_id);
  CREATE INDEX IF NOT EXISTS idx_modlogs_user ON mod_logs(user_id);
  CREATE INDEX IF NOT EXISTS idx_modlogs_time ON mod_logs(timestamp);
`);

const stmts = {
  upsertGuild: db.prepare(`
    INSERT INTO guilds (guild_id, name, joined_at)
    VALUES (?, ?, ?)
    ON CONFLICT(guild_id) DO UPDATE SET name = excluded.name
  `),
  getGuild: db.prepare('SELECT * FROM guilds WHERE guild_id = ?'),
  getGuildPrefix: db.prepare('SELECT prefix FROM guilds WHERE guild_id = ?'),
  setPrefix: db.prepare('UPDATE guilds SET prefix = ? WHERE guild_id = ?'),
  getGuildCount: db.prepare('SELECT COUNT(*) AS count FROM guilds'),

  upsertUser: db.prepare(`
    INSERT INTO users (user_id, username, discriminator, first_seen, last_seen)
    VALUES (?, ?, ?, ?, ?)
    ON CONFLICT(user_id) DO UPDATE SET
      username = excluded.username,
      discriminator = excluded.discriminator,
      last_seen = excluded.last_seen
  `),
  getUser: db.prepare('SELECT * FROM users WHERE user_id = ?'),
  getUserCount: db.prepare('SELECT COUNT(*) AS count FROM users'),

  logCommand: db.prepare(`
    INSERT INTO command_logs (guild_id, user_id, command, args, timestamp)
    VALUES (?, ?, ?, ?, ?)
  `),
  getTotalCommandCount: db.prepare('SELECT COUNT(*) AS count FROM command_logs'),
  getCommandCountByGuild: db.prepare('SELECT COUNT(*) AS count FROM command_logs WHERE guild_id = ?'),

  logModAction: db.prepare(`
    INSERT INTO mod_logs (guild_id, user_id, moderator_id, action, reason, duration, timestamp)
    VALUES (?, ?, ?, ?, ?, ?, ?)
  `),
  getModCount: db.prepare('SELECT COUNT(*) AS count FROM mod_logs'),
  getModCountForUser: db.prepare('SELECT COUNT(*) AS count FROM mod_logs WHERE guild_id = ? AND user_id = ? AND action = ?')
};

module.exports = {
  raw: db,
  path: dbPath,

  registerGuild(guild) {
    if (!guild?.id) return null;
    stmts.upsertGuild.run(guild.id, guild.name || 'unknown', Date.now());
    return stmts.getGuild.get(guild.id);
  },

  registerUser(user) {
    if (!user?.id) return null;
    const now = Date.now();
    const discriminator = user.discriminator && user.discriminator !== '0' ? user.discriminator : null;
    stmts.upsertUser.run(user.id, user.username || 'unknown', discriminator, now, now);
    return stmts.getUser.get(user.id);
  },

  getPrefix(guildId) {
    if (!guildId) return 'a!';
    const row = stmts.getGuildPrefix.get(guildId);
    return row?.prefix || 'a!';
  },

  setPrefix(guildId, prefix) {
    if (!guildId || !prefix) return null;
    stmts.setPrefix.run(prefix, guildId);
    return this.getPrefix(guildId);
  },

  getGuild(guildId) {
    return stmts.getGuild.get(guildId);
  },

  logCommand(guildId, userId, command, args) {
    stmts.logCommand.run(guildId || null, userId, command, args || '', Date.now());
  },

  getStats() {
    return {
      guilds: stmts.getGuildCount.get().count,
      users: stmts.getUserCount.get().count,
      commands: stmts.getTotalCommandCount.get().count
    };
  },

  getGuildStats(guildId) {
    return {
      commands: stmts.getCommandCountByGuild.get(guildId).count
    };
  },

  logModAction(guildId, userId, moderatorId, action, reason, duration = null) {
    stmts.logModAction.run(
      guildId,
      userId,
      moderatorId,
      action,
      reason || null,
      duration,
      Date.now()
    );
  },

  getModStats() {
    return { actions: stmts.getModCount.get().count };
  },

  getUserWarnCount(guildId, userId, action = 'warn') {
    return stmts.getModCountForUser.get(guildId, userId, action).count;
  },

  close() {
    db.close();
  }
};
