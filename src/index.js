require('dotenv').config();
const { Client, Collection, GatewayIntentBits, Partials } = require('discord.js');
const fs = require('node:fs');
const path = require('node:path');
const database = require('./database');

if (!process.env.DISCORD_TOKEN) {
  console.error('HATA: DISCORD_TOKEN .env dosyasında bulunamadı.');
  process.exit(1);
}

const client = new Client({
  intents: [
    GatewayIntentBits.Guilds,
    GatewayIntentBits.GuildMessages,
    GatewayIntentBits.MessageContent,
    GatewayIntentBits.GuildMembers
  ],
  partials: [Partials.Channel, Partials.Message, Partials.GuildMember],
  allowedMentions: { parse: ['users', 'roles'], repliedUser: false }
});

client.commands = new Collection();
client.aliases = new Collection();
client.cooldowns = new Collection();
client.defaultPrefix = 'a!';

const commandsPath = path.join(__dirname, 'commands');
const commandFiles = fs.readdirSync(commandsPath).filter((file) => file.endsWith('.js'));
for (const file of commandFiles) {
  const filePath = path.join(commandsPath, file);
  const command = require(filePath);
  if (!command.name || typeof command.execute !== 'function') {
    console.warn(`[UYARI] ${filePath}: "name" veya "execute" eksik.`);
    continue;
  }
  if (client.commands.has(command.name)) throw new Error(`Yinelenen komut: ${command.name}`);
  client.commands.set(command.name, command);
  for (const alias of command.aliases || []) {
    if (client.aliases.has(alias) || client.commands.has(alias)) throw new Error(`Yinelenen alias: ${alias}`);
    client.aliases.set(alias, command.name);
  }
}

const eventsPath = path.join(__dirname, 'events');
const eventFiles = fs.readdirSync(eventsPath).filter((file) => file.endsWith('.js'));
for (const file of eventFiles) {
  const filePath = path.join(eventsPath, file);
  const event = require(filePath);
  const listener = (...args) => Promise.resolve(event.execute(...args, client)).catch((error) => {
    console.error(`[HATA] ${event.name}:`, error);
  });
  if (event.once) client.once(event.name, listener);
  else client.on(event.name, listener);
}

process.on('unhandledRejection', (error) => console.error('[YAKALANMAMIŞ HATA]', error));
process.on('uncaughtException', (error) => console.error('[BEKLENMEYEN HATA]', error));

let shuttingDown = false;
async function shutdown(signal) {
  if (shuttingDown) return;
  shuttingDown = true;
  console.log(`\n${signal}: kapatılıyor...`);
  client.destroy();
  database.close();
}
process.once('SIGINT', () => shutdown('SIGINT'));
process.once('SIGTERM', () => shutdown('SIGTERM'));

client.login(process.env.DISCORD_TOKEN);
