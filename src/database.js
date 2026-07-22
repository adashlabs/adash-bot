const { DatabaseSync } = require('node:sqlite');
const path = require('node:path');
const fs = require('node:fs');

const dataDir = path.join(__dirname, '..', 'data');
fs.mkdirSync(dataDir, { recursive: true });

const dbPath = process.env.ADASH_DB_PATH || path.join(dataDir, 'adash.db');
const db = new DatabaseSync(dbPath);
db.exec('PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;');

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

  CREATE TABLE IF NOT EXISTS guild_settings (
    guild_id TEXT PRIMARY KEY,
    welcome_channel_id TEXT,
    farewell_channel_id TEXT,
    autorole_id TEXT,
    mod_log_channel_id TEXT,
    ticket_category_id TEXT,
    ticket_log_channel_id TEXT,
    counting_channel_id TEXT,
    word_chain_channel_id TEXT,
    welcome_enabled INTEGER NOT NULL DEFAULT 0,
    farewell_enabled INTEGER NOT NULL DEFAULT 0,
    counting_enabled INTEGER NOT NULL DEFAULT 0,
    word_chain_enabled INTEGER NOT NULL DEFAULT 0,
    welcome_message TEXT NOT NULL DEFAULT '{user}, {server} sunucusuna hoş geldin!',
    farewell_message TEXT NOT NULL DEFAULT '{user} aramızdan ayrıldı. Görüşmek üzere!',
    updated_at INTEGER NOT NULL
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

  CREATE TABLE IF NOT EXISTS warnings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    moderator_id TEXT NOT NULL,
    reason TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    removed_at INTEGER
  );

  CREATE TABLE IF NOT EXISTS game_states (
    guild_id TEXT PRIMARY KEY,
    counting_value INTEGER NOT NULL DEFAULT 0,
    counting_user_id TEXT,
    last_word TEXT,
    word_user_id TEXT,
    updated_at INTEGER NOT NULL
  );

  CREATE TABLE IF NOT EXISTS word_game_used (
    guild_id TEXT NOT NULL,
    word TEXT NOT NULL,
    PRIMARY KEY (guild_id, word)
  );

  CREATE TABLE IF NOT EXISTS guild_config (
    guild_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (guild_id, key)
  );

  CREATE TABLE IF NOT EXISTS tickets (
    channel_id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    category_id TEXT,
    claimed_by_id TEXT,
    ticket_type TEXT NOT NULL DEFAULT 'destek',
    priority TEXT NOT NULL DEFAULT 'normal',
    subject TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',
    opened_at INTEGER NOT NULL,
    closed_at INTEGER,
    closed_by_id TEXT,
    close_reason TEXT
  );

  CREATE TABLE IF NOT EXISTS giveaways (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL UNIQUE,
    host_id TEXT NOT NULL,
    prize TEXT NOT NULL,
    winner_count INTEGER NOT NULL,
    required_role_id TEXT,
    min_account_age_days INTEGER NOT NULL DEFAULT 0,
    ends_at INTEGER NOT NULL,
    ended_at INTEGER
  );

  CREATE TABLE IF NOT EXISTS giveaway_entries (
    giveaway_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    PRIMARY KEY (giveaway_id, user_id),
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE
  );

  CREATE INDEX IF NOT EXISTS idx_tickets_guild_owner ON tickets(guild_id, owner_id, closed_at);
  CREATE INDEX IF NOT EXISTS idx_giveaways_due ON giveaways(ended_at, ends_at);
  CREATE INDEX IF NOT EXISTS idx_cmdlogs_guild ON command_logs(guild_id);
  CREATE INDEX IF NOT EXISTS idx_cmdlogs_user ON command_logs(user_id);
  CREATE INDEX IF NOT EXISTS idx_cmdlogs_time ON command_logs(timestamp);
  CREATE INDEX IF NOT EXISTS idx_modlogs_guild ON mod_logs(guild_id);
  CREATE INDEX IF NOT EXISTS idx_modlogs_user ON mod_logs(user_id);
  CREATE INDEX IF NOT EXISTS idx_modlogs_time ON mod_logs(timestamp);
  CREATE INDEX IF NOT EXISTS idx_warnings_user ON warnings(guild_id, user_id, active);
`);

function ensureColumn(table, column, type) {
  const columns = new Set(db.prepare(`PRAGMA table_info(${table})`).all().map((item) => item.name));
  if (!columns.has(column)) db.exec(`ALTER TABLE ${table} ADD COLUMN ${column} ${type}`);
}
ensureColumn('guild_settings', 'ticket_category_id', 'TEXT');
ensureColumn('guild_settings', 'ticket_log_channel_id', 'TEXT');
ensureColumn('guild_settings', 'ai_channel_id', 'TEXT');
ensureColumn('tickets', 'claimed_by_id', 'TEXT');
ensureColumn('tickets', 'ticket_type', "TEXT NOT NULL DEFAULT 'destek'");
ensureColumn('tickets', 'priority', "TEXT NOT NULL DEFAULT 'normal'");
ensureColumn('tickets', 'subject', "TEXT NOT NULL DEFAULT ''");
ensureColumn('tickets', 'description', "TEXT NOT NULL DEFAULT ''");
ensureColumn('tickets', 'status', "TEXT NOT NULL DEFAULT 'open'");
ensureColumn('tickets', 'close_reason', 'TEXT');
ensureColumn('giveaways', 'required_role_id', 'TEXT');
ensureColumn('giveaways', 'min_account_age_days', 'INTEGER NOT NULL DEFAULT 0');

const SETTINGS = new Set([
  'welcome_channel_id', 'farewell_channel_id', 'autorole_id', 'mod_log_channel_id',
  'counting_channel_id', 'word_chain_channel_id', 'ticket_category_id', 'ticket_log_channel_id',
  'welcome_enabled', 'farewell_enabled', 'counting_enabled', 'word_chain_enabled',
  'welcome_message', 'farewell_message', 'ai_channel_id'
]);

const stmts = {
  upsertGuild: db.prepare(`INSERT INTO guilds (guild_id, name, joined_at) VALUES (?, ?, ?)
    ON CONFLICT(guild_id) DO UPDATE SET name = excluded.name`),
  getGuild: db.prepare('SELECT * FROM guilds WHERE guild_id = ?'),
  getGuildPrefix: db.prepare('SELECT prefix FROM guilds WHERE guild_id = ?'),
  setPrefix: db.prepare('UPDATE guilds SET prefix = ? WHERE guild_id = ?'),
  getGuildCount: db.prepare('SELECT COUNT(*) AS count FROM guilds'),
  ensureSettings: db.prepare('INSERT OR IGNORE INTO guild_settings (guild_id, updated_at) VALUES (?, ?)'),
  getSettings: db.prepare('SELECT * FROM guild_settings WHERE guild_id = ?'),
  getConfig: db.prepare('SELECT value FROM guild_config WHERE guild_id = ? AND key = ?'),
  setConfig: db.prepare(`INSERT INTO guild_config (guild_id, key, value, updated_at) VALUES (?, ?, ?, ?)
    ON CONFLICT(guild_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`),
  upsertUser: db.prepare(`INSERT INTO users (user_id, username, discriminator, first_seen, last_seen)
    VALUES (?, ?, ?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET username = excluded.username,
    discriminator = excluded.discriminator, last_seen = excluded.last_seen`),
  getUser: db.prepare('SELECT * FROM users WHERE user_id = ?'),
  getUserCount: db.prepare('SELECT COUNT(*) AS count FROM users'),
  getModLogsForUser: db.prepare('SELECT * FROM mod_logs WHERE guild_id = ? AND user_id = ? ORDER BY id DESC LIMIT ?'),
  getRecentModLogs: db.prepare('SELECT * FROM mod_logs WHERE guild_id = ? ORDER BY id DESC LIMIT ?'),
  logCommand: db.prepare('INSERT INTO command_logs (guild_id, user_id, command, args, timestamp) VALUES (?, ?, ?, ?, ?)'),
  getTotalCommandCount: db.prepare('SELECT COUNT(*) AS count FROM command_logs'),
  getCommandCountByGuild: db.prepare('SELECT COUNT(*) AS count FROM command_logs WHERE guild_id = ?'),
  logModAction: db.prepare(`INSERT INTO mod_logs
    (guild_id, user_id, moderator_id, action, reason, duration, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?)`),
  getModCount: db.prepare('SELECT COUNT(*) AS count FROM mod_logs'),
  addWarning: db.prepare(`INSERT INTO warnings
    (guild_id, user_id, moderator_id, reason, created_at) VALUES (?, ?, ?, ?, ?)`),
  getWarnings: db.prepare('SELECT * FROM warnings WHERE guild_id = ? AND user_id = ? AND active = 1 ORDER BY id DESC'),
  getWarningCount: db.prepare('SELECT COUNT(*) AS count FROM warnings WHERE guild_id = ? AND user_id = ? AND active = 1'),
  getLatestWarning: db.prepare('SELECT id FROM warnings WHERE guild_id = ? AND user_id = ? AND active = 1 ORDER BY id DESC LIMIT 1'),
  removeWarning: db.prepare('UPDATE warnings SET active = 0, removed_at = ? WHERE id = ?'),
  clearWarnings: db.prepare('UPDATE warnings SET active = 0, removed_at = ? WHERE guild_id = ? AND user_id = ? AND active = 1'),
  ensureGame: db.prepare('INSERT OR IGNORE INTO game_states (guild_id, updated_at) VALUES (?, ?)'),
  getGame: db.prepare('SELECT * FROM game_states WHERE guild_id = ?'),
  setCount: db.prepare('UPDATE game_states SET counting_value = ?, counting_user_id = ?, updated_at = ? WHERE guild_id = ?'),
  setWord: db.prepare('UPDATE game_states SET last_word = ?, word_user_id = ?, updated_at = ? WHERE guild_id = ?'),
  hasWord: db.prepare('SELECT 1 AS found FROM word_game_used WHERE guild_id = ? AND word = ?'),
  addWord: db.prepare('INSERT INTO word_game_used (guild_id, word) VALUES (?, ?)'),
  clearWords: db.prepare('DELETE FROM word_game_used WHERE guild_id = ?'),
  getOpenTicket: db.prepare('SELECT * FROM tickets WHERE guild_id = ? AND owner_id = ? AND closed_at IS NULL LIMIT 1'),
  createTicket: db.prepare(`INSERT INTO tickets
    (channel_id, guild_id, owner_id, category_id, ticket_type, priority, subject, description, opened_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
  getTicket: db.prepare('SELECT * FROM tickets WHERE channel_id = ?'),
  closeTicket: db.prepare('UPDATE tickets SET closed_at = ?, closed_by_id = ?, close_reason = ?, status = ? WHERE channel_id = ? AND closed_at IS NULL'),
  claimTicket: db.prepare('UPDATE tickets SET claimed_by_id = ?, status = ? WHERE channel_id = ? AND closed_at IS NULL AND claimed_by_id IS NULL'),
  setTicketStatus: db.prepare('UPDATE tickets SET status = ? WHERE channel_id = ? AND closed_at IS NULL'),
  createGiveaway: db.prepare(`INSERT INTO giveaways
    (guild_id, channel_id, message_id, host_id, prize, winner_count, required_role_id, min_account_age_days, ends_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
  getGiveawayByMessage: db.prepare('SELECT * FROM giveaways WHERE message_id = ?'),
  getGiveawayById: db.prepare('SELECT * FROM giveaways WHERE id = ?'),
  getDueGiveaways: db.prepare('SELECT * FROM giveaways WHERE ended_at IS NULL AND ends_at <= ?'),
  endGiveaway: db.prepare('UPDATE giveaways SET ended_at = ? WHERE id = ? AND ended_at IS NULL'),
  addGiveawayEntry: db.prepare('INSERT OR IGNORE INTO giveaway_entries (giveaway_id, user_id) VALUES (?, ?)'),
  removeGiveawayEntry: db.prepare('DELETE FROM giveaway_entries WHERE giveaway_id = ? AND user_id = ?'),
  getGiveawayEntries: db.prepare('SELECT user_id FROM giveaway_entries WHERE giveaway_id = ?')
};

function ensureGuildData(guildId) {
  const now = Date.now();
  stmts.ensureSettings.run(guildId, now);
  stmts.ensureGame.run(guildId, now);
}

module.exports = {
  raw: db,
  path: dbPath,

  registerGuild(guild) {
    if (!guild?.id) return null;
    stmts.upsertGuild.run(guild.id, guild.name || 'unknown', Date.now());
    ensureGuildData(guild.id);
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
    return stmts.getGuildPrefix.get(guildId)?.prefix || 'a!';
  },

  setPrefix(guildId, prefix) {
    if (!guildId || !prefix) return null;
    stmts.setPrefix.run(prefix, guildId);
    return this.getPrefix(guildId);
  },

  getGuild(guildId) {
    return stmts.getGuild.get(guildId);
  },

  getSettings(guildId) {
    ensureGuildData(guildId);
    return stmts.getSettings.get(guildId);
  },

  setSetting(guildId, key, value) {
    if (!SETTINGS.has(key)) throw new Error(`Geçersiz ayar: ${key}`);
    ensureGuildData(guildId);
    db.prepare(`UPDATE guild_settings SET ${key} = ?, updated_at = ? WHERE guild_id = ?`).run(value, Date.now(), guildId);
    return this.getSettings(guildId);
  },

  getConfig(guildId, key, fallback = null) {
    const row = stmts.getConfig.get(guildId, key);
    if (!row) return fallback;
    try {
      return JSON.parse(row.value);
    } catch {
      return fallback;
    }
  },

  setConfig(guildId, key, value) {
    stmts.setConfig.run(guildId, key, JSON.stringify(value), Date.now());
    return value;
  },

  logCommand(guildId, userId, command, args) {
    stmts.logCommand.run(guildId || null, userId, command, args || '', Date.now());
  },

  getModLogsForUser(guildId, userId, limit = 10) {
    return stmts.getModLogsForUser.all(guildId, userId, limit);
  },

  getRecentModLogs(guildId, limit = 10) {
    return stmts.getRecentModLogs.all(guildId, limit);
  },

  getStats() {
    return {
      guilds: stmts.getGuildCount.get().count,
      users: stmts.getUserCount.get().count,
      commands: stmts.getTotalCommandCount.get().count
    };
  },

  getGuildStats(guildId) {
    return { commands: stmts.getCommandCountByGuild.get(guildId).count };
  },

  logModAction(guildId, userId, moderatorId, action, reason, duration = null) {
    stmts.logModAction.run(guildId, userId, moderatorId, action, reason || null, duration, Date.now());
  },

  getModStats() {
    return { actions: stmts.getModCount.get().count };
  },

  addWarning(guildId, userId, moderatorId, reason) {
    const result = stmts.addWarning.run(guildId, userId, moderatorId, reason, Date.now());
    return { id: Number(result.lastInsertRowid), count: this.getUserWarnCount(guildId, userId) };
  },

  getWarnings(guildId, userId) {
    return stmts.getWarnings.all(guildId, userId);
  },

  getUserWarnCount(guildId, userId) {
    return stmts.getWarningCount.get(guildId, userId).count;
  },

  removeLatestWarning(guildId, userId) {
    const row = stmts.getLatestWarning.get(guildId, userId);
    if (!row) return false;
    stmts.removeWarning.run(Date.now(), row.id);
    return true;
  },

  clearWarnings(guildId, userId) {
    return Number(stmts.clearWarnings.run(Date.now(), guildId, userId).changes);
  },

  getGameState(guildId) {
    ensureGuildData(guildId);
    return stmts.getGame.get(guildId);
  },

  setCountingState(guildId, value, userId) {
    ensureGuildData(guildId);
    stmts.setCount.run(value, userId, Date.now(), guildId);
  },

  setWordState(guildId, word, userId) {
    ensureGuildData(guildId);
    stmts.setWord.run(word, userId, Date.now(), guildId);
    stmts.addWord.run(guildId, word);
  },

  hasUsedWord(guildId, word) {
    return Boolean(stmts.hasWord.get(guildId, word));
  },

  resetGame(guildId, game) {
    ensureGuildData(guildId);
    if (game === 'counting') {
      stmts.setCount.run(0, null, Date.now(), guildId);
    } else if (game === 'word') {
      stmts.setWord.run(null, null, Date.now(), guildId);
      stmts.clearWords.run(guildId);
    } else {
      throw new Error('Geçersiz oyun');
    }
    return this.getGameState(guildId);
  },

  getOpenTicket(guildId, ownerId) {
    return stmts.getOpenTicket.get(guildId, ownerId);
  },

  createTicket(channelId, guildId, ownerId, categoryId, data = {}) {
    stmts.createTicket.run(
      channelId, guildId, ownerId, categoryId || null, data.type || 'destek',
      data.priority || 'normal', data.subject || '', data.description || '', Date.now()
    );
  },

  getTicket(channelId) {
    return stmts.getTicket.get(channelId);
  },

  closeTicket(channelId, closedById, reason = 'Sebep belirtilmedi') {
    return Number(stmts.closeTicket.run(Date.now(), closedById, reason, 'closed', channelId).changes) === 1;
  },

  claimTicket(channelId, userId) {
    return Number(stmts.claimTicket.run(userId, 'claimed', channelId).changes) === 1;
  },

  setTicketStatus(channelId, status) {
    return Number(stmts.setTicketStatus.run(status, channelId).changes) === 1;
  },

  createGiveaway(guildId, channelId, messageId, hostId, prize, winnerCount, requiredRoleId, minAccountAgeDays, endsAt) {
    return Number(stmts.createGiveaway.run(
      guildId, channelId, messageId, hostId, prize, winnerCount,
      requiredRoleId || null, minAccountAgeDays || 0, endsAt
    ).lastInsertRowid);
  },

  getGiveawayByMessage(messageId) {
    return stmts.getGiveawayByMessage.get(messageId);
  },

  getGiveawayById(giveawayId) {
    return stmts.getGiveawayById.get(giveawayId);
  },

  joinGiveaway(giveawayId, userId) {
    return Number(stmts.addGiveawayEntry.run(giveawayId, userId).changes) === 1;
  },

  leaveGiveaway(giveawayId, userId) {
    return Number(stmts.removeGiveawayEntry.run(giveawayId, userId).changes) === 1;
  },

  getGiveawayEntries(giveawayId) {
    return stmts.getGiveawayEntries.all(giveawayId).map((row) => row.user_id);
  },

  getDueGiveaways(now = Date.now()) {
    return stmts.getDueGiveaways.all(now);
  },

  endGiveaway(giveawayId) {
    return Number(stmts.endGiveaway.run(Date.now(), giveawayId).changes) === 1;
  },

  close() {
    db.close();
  }
};
