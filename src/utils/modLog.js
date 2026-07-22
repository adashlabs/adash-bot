const { EmbedBuilder } = require('discord.js');
const database = require('../database');

const ACTION_LABELS = {
  ban: 'Yasaklama', kick: 'Sunucudan atma', mute: 'Susturma', unmute: 'Susturma kaldırma',
  warn: 'Uyarı', unwarn: 'Uyarı kaldırma', clearwarns: 'Uyarıları temizleme', clear: 'Mesaj temizleme',
  ticket: 'Ticket kapatma', lock: 'Kanal kilitleme', unlock: 'Kanal kilidi açma', slowmode: 'Yavaş mod'
};

async function sendModLog(guild, data) {
  const settings = database.getSettings(guild.id);
  if (!settings.mod_log_channel_id) return false;

  const channel = await guild.channels.fetch(settings.mod_log_channel_id).catch(() => null);
  if (!channel?.isTextBased()) return false;

  const embed = new EmbedBuilder()
    .setColor(data.color || 0xED4245)
    .setTitle(`🛡️ ${ACTION_LABELS[data.action] || data.action}`)
    .addFields(
      { name: 'Hedef', value: data.targetId ? `<@${data.targetId}> (\`${data.targetId}\`)` : '—', inline: true },
      { name: 'Yetkili', value: `<@${data.moderatorId}>`, inline: true },
      { name: 'Sebep', value: String(data.reason || 'Sebep belirtilmedi').slice(0, 1024) }
    )
    .setTimestamp();

  if (data.duration) embed.addFields({ name: 'Süre', value: data.duration, inline: true });
  if (data.extra) embed.addFields({ name: 'Detay', value: String(data.extra).slice(0, 1024) });

  await channel.send({ embeds: [embed] }).catch(() => null);
  return true;
}

module.exports = { sendModLog };
