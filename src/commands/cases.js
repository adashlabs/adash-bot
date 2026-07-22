const { EmbedBuilder, PermissionFlagsBits } = require('discord.js');
const { resolveTarget } = require('../utils/security');
const { COLORS, truncate } = require('../utils/ui');
const database = require('../database');

const labels = { ban: 'Yasaklama', unban: 'Yasak kaldırma', kick: 'Atma', mute: 'Susturma', unmute: 'Susturma kaldırma', warn: 'Uyarı', clearwarns: 'Uyarı temizleme', clear: 'Mesaj temizleme' };

module.exports = {
  name: 'cases', aliases: ['vakalar', 'modlog'], category: 'moderasyon',
  description: 'bir kullanıcının veya sunucunun son moderasyon vakalarını gösterir.', cooldown: 3,
  async execute(message, args) {
    if (!message.member.permissions.has(PermissionFlagsBits.ModerateMembers)) return message.reply('bu komut için moderasyon yetkisi gerekiyor.');
    const target = args[0] ? await resolveTarget(message, args[0]) : null;
    const rows = target ? database.getModLogsForUser(message.guild.id, target.user.id, 10) : database.getRecentModLogs(message.guild.id, 10);
    const embed = new EmbedBuilder().setColor(COLORS.primary).setTitle(target ? `🛡️ ${target.user.tag} · Moderasyon Geçmişi` : '🛡️ Son Moderasyon Vakaları')
      .setDescription(rows.length ? rows.map((row) => `**#${row.id} · ${labels[row.action] || row.action}** · <t:${Math.floor(row.timestamp / 1000)}:R>\nHedef: <@${row.user_id}> · Yetkili: <@${row.moderator_id}>\n${truncate(row.reason || 'Sebep belirtilmedi', 180)}`).join('\n\n') : 'Kayıt bulunamadı.')
      .setFooter({ text: 'En fazla son 10 kayıt gösterilir.' });
    await message.reply({ embeds: [embed] });
  }
};
