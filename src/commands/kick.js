const { PermissionFlagsBits, EmbedBuilder } = require('discord.js');
const { resolveTarget, moderationError } = require('../utils/security');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');
const { COLORS, truncate } = require('../utils/ui');
const database = require('../database');

module.exports = {
  name: 'kick', aliases: ['at'], category: 'moderasyon',
  description: 'kullanıcıyı rol kontrolleri ve düğmeli onay sonrasında sunucudan atar.', cooldown: 3,
  async execute(message, args) {
    const target = await resolveTarget(message, args[0]);
    if (!target?.member) return message.reply('sunucudaki bir kullanıcıyı belirt.');
    const options = { userPermission: PermissionFlagsBits.KickMembers, action: 'kickable' };
    const error = moderationError(message, target.member, options);
    if (error) return message.reply(error);
    const reason = truncate(args.slice(1).join(' ') || 'Sebep belirtilmedi', 500);
    await requestConfirmation(message, { title: 'Kullanıcıyı Sunucudan At', target: `${target.user.tag} (\`${target.user.id}\`)`, reason }, async (interaction) => {
      const member = await message.guild.members.fetch(target.user.id).catch(() => null);
      if (!member) return interaction.followUp({ content: 'kullanıcı artık sunucuda değil.', flags: 64 });
      const freshError = moderationError({ ...message, member: interaction.member }, member, options);
      if (freshError) return interaction.followUp({ content: freshError, flags: 64 });
      await member.kick(reason);
      database.logModAction(message.guild.id, target.user.id, interaction.user.id, 'kick', reason);
      await sendModLog(message.guild, { action: 'kick', targetId: target.user.id, moderatorId: interaction.user.id, reason });
      const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('👢 Kullanıcı Sunucudan Atıldı')
        .addFields(
          { name: 'Kullanıcı', value: `${target.user.tag} (\`${target.user.id}\`)`, inline: true },
          { name: 'Yetkili', value: `<@${interaction.user.id}>`, inline: true },
          { name: 'Sebep', value: reason }
        ).setTimestamp();
      await interaction.followUp({ embeds: [embed] });
    });
  }
};
