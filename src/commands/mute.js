const { PermissionFlagsBits, EmbedBuilder } = require('discord.js');
const { resolveTarget, moderationError } = require('../utils/security');
const { parseDuration, formatDuration } = require('../utils/resolvers');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');
const { COLORS, truncate } = require('../utils/ui');
const database = require('../database');

module.exports = {
  name: 'mute', aliases: ['sustur', 'timeout'], category: 'moderasyon',
  description: 'kullanıcıyı 28 güne kadar, düğmeli onay sonrasında susturur.', cooldown: 3,
  async execute(message, args) {
    const target = await resolveTarget(message, args[0]);
    if (!target?.member) return message.reply('sunucudaki bir kullanıcıyı belirt.');
    const options = { userPermission: PermissionFlagsBits.ModerateMembers, action: 'moderatable' };
    const error = moderationError(message, target.member, options);
    if (error) return message.reply(error);
    if (target.member.isCommunicationDisabled()) return message.reply('bu kullanıcı zaten susturulmuş.');
    const durationMs = parseDuration(args[1]);
    if (!durationMs || durationMs < 1000 || durationMs > 28 * 24 * 60 * 60 * 1000) return message.reply('1 saniye ile 28 gün arasında süre belirt.');
    const duration = formatDuration(durationMs);
    const reason = truncate(args.slice(2).join(' ') || 'Sebep belirtilmedi', 500);
    await requestConfirmation(message, { title: 'Kullanıcıyı Sustur', target: `${target.user.tag} (\`${target.user.id}\`)`, reason, details: `Süre: ${duration}` }, async (interaction) => {
      const member = await message.guild.members.fetch(target.user.id).catch(() => null);
      if (!member) return interaction.followUp({ content: 'kullanıcı artık sunucuda değil.', flags: 64 });
      const freshError = moderationError({ ...message, member: interaction.member }, member, options);
      if (freshError) return interaction.followUp({ content: freshError, flags: 64 });
      await member.timeout(durationMs, reason);
      database.logModAction(message.guild.id, target.user.id, interaction.user.id, 'mute', reason, durationMs);
      await sendModLog(message.guild, { action: 'mute', targetId: target.user.id, moderatorId: interaction.user.id, reason, duration });
      const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle('🔇 Kullanıcı Susturuldu')
        .addFields(
          { name: 'Kullanıcı', value: `${target.user.tag} (\`${target.user.id}\`)`, inline: true },
          { name: 'Süre', value: duration, inline: true },
          { name: 'Yetkili', value: `<@${interaction.user.id}>`, inline: true },
          { name: 'Sebep', value: reason }
        ).setTimestamp();
      await interaction.followUp({ embeds: [embed] });
    });
  }
};
