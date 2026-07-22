const os = require('node:os');
const process = require('node:process');
const fs = require('node:fs');
const path = require('node:path');
const { ActionRowBuilder, ButtonBuilder, ButtonStyle, EmbedBuilder } = require('discord.js');

let cachedDjsVersion = null;
function getDjsVersion() {
  if (cachedDjsVersion !== null) return cachedDjsVersion;
  try {
    const pkgPath = path.join(__dirname, '..', '..', 'node_modules', 'discord.js', 'package.json');
    cachedDjsVersion = JSON.parse(fs.readFileSync(pkgPath, 'utf8')).version;
  } catch (e) {
    cachedDjsVersion = '?';
  }
  return cachedDjsVersion;
}

function loadBar(ratio, scale = 1) {
  const clamped = Math.max(0, Math.min(1, ratio / scale));
  const green = 10 - Math.round(clamped * 10);
  const red = 10 - green;
  return '\uD83D\uDFE9'.repeat(green) + '\uD83D\uDFE5'.repeat(red);
}

function formatUptime(seconds) {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor(seconds / 3600) % 24;
  const m = Math.floor(seconds / 60) % 60;
  const s = Math.floor(seconds % 60);
  return `${d}g ${h}s ${m}d ${s}sn`;
}

function buildOutput(client, guildId, roundtripOverride, user) {
  const ws = Math.round(client.ws.ping);
  const uptime = process.uptime();
  const loadAvg = os.loadavg()[0];
  const totalMem = os.totalmem();
  const freeMem = os.freemem();
  const usedMem = totalMem - freeMem;
  const memRatio = totalMem > 0 ? usedMem / totalMem : 0;

  const shardId = client.shard?.ids?.[0] ?? 0;
  const shardCount = client.shard?.count ?? 1;
  const roundtrip = roundtripOverride !== undefined ? roundtripOverride : (ws >= 0 ? Math.max(0, ws) : 0);
  const statusEmoji = ws < 150 ? '🟢' : ws < 300 ? '🟡' : '🔴';

  const embed = new EmbedBuilder()
    .setColor(ws < 150 ? 0x57F287 : ws < 300 ? 0xFEE75C : 0xED4245)
    .setTitle(`${statusEmoji} Adash · Sistem & Bağlantı Durumu`)
    .addFields(
      { name: 'Gecikme (Ping)', value: `⚡ Mesaj: **${roundtrip}ms**\n🌐 WebSocket: **${ws}ms**`, inline: true },
      { name: 'Çalışma Süresi', value: `⏱️ **${formatUptime(uptime)}**`, inline: true },
      { name: 'Shard', value: `🧩 **${shardId}/${shardCount}**`, inline: true },
      { name: 'CPU Yükü', value: `${loadBar(loadAvg, 2)} **${loadAvg.toFixed(2)}**`, inline: true },
      { name: 'RAM Kullanımı', value: `${loadBar(memRatio, 1)} **${(memRatio * 100).toFixed(1)}%**`, inline: true },
      { name: 'Kitaplık / Node', value: `📦 discord.js v${getDjsVersion()}\n🟢 Node.js ${process.version}`, inline: true }
    )
    .setTimestamp();
  if (user) embed.setFooter({ text: `İsteyen: ${user.tag}`, iconURL: user.displayAvatarURL() });

  const row = new ActionRowBuilder().addComponents(
    new ButtonBuilder().setCustomId('ping_refresh').setLabel('Yenile').setEmoji('🔄').setStyle(ButtonStyle.Primary)
  );

  return { embeds: [embed], components: [row] };
}

module.exports = {
  name: 'ping',
  category: 'genel',
  description: 'botun durumunu g\u00F6sterir (gecikme, y\u00FCk, shard).',
  buildOutput,

  async execute(message, args, client) {
    const placeholder = await message.reply('hesaplanıyor...');
    const roundtrip = placeholder.createdTimestamp - message.createdTimestamp;
    const payload = buildOutput(client, message.guild.id, roundtrip, message.author);
    await placeholder.edit({ content: null, ...payload });
  }
};