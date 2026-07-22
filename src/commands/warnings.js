const { EmbedBuilder, PermissionFlagsBits } = require('discord.js');
const { resolveTarget } = require('../utils/security');
const { COLORS, truncate } = require('../utils/ui');
const database = require('../database');

module.exports = {
  name: 'warnings',
  aliases: ['uyarılar', 'uyarilar'],
  category: 'moderasyon',
  description: 'kendinin veya yetkiliysen başka bir üyenin aktif uyarılarını gösterir.',
  async execute(message, args) {
    let user = message.author;
    if (args[0]) {
      if (!message.member.permissions.has(PermissionFlagsBits.ModerateMembers)) {
        return message.reply('başka birinin uyarılarını görmek için moderasyon yetkisi gerekiyor.');
      }
      const target = await resolveTarget(message, args[0]);
      if (!target) return message.reply('geçerli bir kullanıcı belirt.');
      user = target.user;
    }

    const warnings = database.getWarnings(message.guild.id, user.id);
    const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle(`⚠️ ${user.tag} · Aktif Uyarılar`)
      .setDescription(warnings.length ? warnings.slice(0, 10).map((warning) =>
        `**#${warning.id}** · <t:${Math.floor(warning.created_at / 1000)}:R> · ${truncate(warning.reason, 180)}`
      ).join('\n') : 'Aktif uyarı bulunmuyor.')
      .setFooter({ text: `Toplam aktif uyarı: ${warnings.length}` });
    await message.reply({ embeds: [embed] });
  }
};
