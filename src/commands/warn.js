const { PermissionFlagsBits, EmbedBuilder } = require('discord.js');
const { resolveTarget, moderationError } = require('../utils/security');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');
const { COLORS, truncate } = require('../utils/ui');
const { formatDuration } = require('../utils/resolvers');
const database = require('../database');

module.exports = {
  name: 'warn', aliases: ['uyar'], category: 'moderasyon',
  description: 'kullanıcıya onay sonrasında kayıtlı uyarı verir; sunucu uyarı politikasını uygular.', cooldown: 3,
  async execute(message, args, client) {
    const target = await resolveTarget(message, args[0]);
    if (!target?.member) return message.reply('sunucudaki bir kullanıcıyı belirt.');
    const options = { userPermission: PermissionFlagsBits.ModerateMembers, action: 'moderatable' };
    const error = moderationError(message, target.member, options);
    if (error) return message.reply(error);
    const reason = truncate(args.slice(1).join(' ') || 'Sebep belirtilmedi', 500);
    const currentCount = database.getUserWarnCount(message.guild.id, target.user.id);
    const threshold = Math.max(1, Math.min(10, Number(database.getConfig(message.guild.id, 'warn_auto_threshold', 3)) || 3));
    const timeoutMs = Math.max(10_000, Math.min(28 * 86400_000, Number(database.getConfig(message.guild.id, 'warn_auto_timeout_ms', 600_000)) || 600_000));
    await requestConfirmation(message, {
      title: 'Kullanıcıya Uyarı Ver', target: `${target.user.tag} (\`${target.user.id}\`)`, reason,
      details: `Mevcut aktif uyarı: ${currentCount} · işlem sonrası: ${currentCount + 1}${currentCount + 1 === threshold ? ` · ${formatDuration(timeoutMs)} otomatik timeout` : ''}`
    }, async (interaction) => {
      const member = await message.guild.members.fetch(target.user.id).catch(() => null);
      if (!member) return interaction.followUp({ content: 'kullanıcı artık sunucuda değil.', flags: 64 });
      const freshError = moderationError({ ...message, member: interaction.member }, member, options);
      if (freshError) return interaction.followUp({ content: freshError, flags: 64 });
      const warning = database.addWarning(message.guild.id, target.user.id, interaction.user.id, reason);
      database.logModAction(message.guild.id, target.user.id, interaction.user.id, 'warn', reason);
      await sendModLog(message.guild, { action: 'warn', targetId: target.user.id, moderatorId: interaction.user.id, reason, extra: `Aktif uyarı: ${warning.count}`, color: COLORS.warning });
      await interaction.followUp({ embeds: [new EmbedBuilder().setColor(COLORS.warning).setTitle('⚠️ Kullanıcı uyarıldı')
        .addFields({ name: 'Kullanıcı', value: target.user.tag, inline: true }, { name: 'Aktif uyarı', value: String(warning.count), inline: true }, { name: 'Sebep', value: reason }).setTimestamp()] });
      if (warning.count === threshold && member.moderatable) {
        const autoReason = `${threshold} aktif uyarı sonrası otomatik susturma`;
        await member.timeout(timeoutMs, autoReason);
        database.logModAction(message.guild.id, target.user.id, client.user.id, 'mute', autoReason, timeoutMs);
        await interaction.followUp(`🔇 **${target.user.tag}**, ${threshold} aktif uyarıya ulaştığı için ${formatDuration(timeoutMs)} susturuldu.`);
      }
    });
  }
};
