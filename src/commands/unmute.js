const { PermissionFlagsBits } = require('discord.js');
const { resolveTarget, moderationError } = require('../utils/security');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');
const { truncate } = require('../utils/ui');
const database = require('../database');

module.exports = {
  name: 'unmute', aliases: ['susturmaaç'], category: 'moderasyon',
  description: 'kullanıcının susturmasını düğmeli onay sonrasında kaldırır.', cooldown: 3,
  async execute(message, args) {
    const target = await resolveTarget(message, args[0]);
    if (!target?.member) return message.reply('sunucudaki bir kullanıcıyı belirt.');
    const options = { userPermission: PermissionFlagsBits.ModerateMembers, action: 'moderatable' };
    const error = moderationError(message, target.member, options);
    if (error) return message.reply(error);
    if (!target.member.isCommunicationDisabled()) return message.reply('bu kullanıcı susturulmuş değil.');
    const reason = truncate(args.slice(1).join(' ') || 'Sebep belirtilmedi', 500);
    await requestConfirmation(message, { title: 'Susturmayı Kaldır', target: `${target.user.tag} (\`${target.user.id}\`)`, reason }, async (interaction) => {
      const member = await message.guild.members.fetch(target.user.id).catch(() => null);
      if (!member) return interaction.followUp({ content: 'kullanıcı artık sunucuda değil.', flags: 64 });
      const freshError = moderationError({ ...message, member: interaction.member }, member, options);
      if (freshError) return interaction.followUp({ content: freshError, flags: 64 });
      await member.timeout(null, reason);
      database.logModAction(message.guild.id, target.user.id, interaction.user.id, 'unmute', reason);
      await sendModLog(message.guild, { action: 'unmute', targetId: target.user.id, moderatorId: interaction.user.id, reason, color: COLORS.success });
      const embed = new EmbedBuilder().setColor(COLORS.success).setTitle('🔊 Susturma Kaldırıldı')
        .addFields(
          { name: 'Kullanıcı', value: `${target.user.tag} (\`${target.user.id}\`)`, inline: true },
          { name: 'Yetkili', value: `<@${interaction.user.id}>`, inline: true },
          { name: 'Sebep', value: reason }
        ).setTimestamp();
      await interaction.followUp({ embeds: [embed] });
    });
  }
};
