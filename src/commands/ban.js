const { PermissionFlagsBits, EmbedBuilder } = require('discord.js');
const { resolveTarget, moderationError } = require('../utils/security');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');
const { COLORS, truncate } = require('../utils/ui');
const database = require('../database');

module.exports = {
  name: 'ban', aliases: ['yasakla'], category: 'moderasyon',
  description: 'kullanıcıyı rol kontrolleri ve düğmeli onay sonrasında yasaklar.', cooldown: 3,
  async execute(message, args) {
    const target = await resolveTarget(message, args[0]);
    if (!target) return message.reply('kullanıcı belirt. kullanım: `a!ban @kullanıcı [sebep]`');
    const options = { userPermission: PermissionFlagsBits.BanMembers, action: 'bannable' };
    const error = moderationError(message, target.member, options);
    if (error) return message.reply(error);
    const days = Math.min(7, Math.max(0, Number(args.find((value) => value.startsWith('--days='))?.split('=')[1] || 0)));
    const reason = truncate(args.slice(1).filter((value) => !value.startsWith('--days=')).join(' ') || 'Sebep belirtilmedi', 500);

    await requestConfirmation(message, {
      title: 'Kullanıcıyı Yasakla', target: `${target.user.tag} (\`${target.user.id}\`)`, reason,
      details: `Kullanıcı sunucudan çıkarılacak. Son ${days} günlük mesaj geçmişi silinecek.`
    }, async (interaction) => {
      const member = await message.guild.members.fetch(target.user.id).catch(() => null);
      const freshError = moderationError({ ...message, member: interaction.member }, member, options);
      if (freshError) return interaction.followUp({ content: freshError, flags: 64 });
      await message.guild.members.ban(target.user.id, { reason, deleteMessageSeconds: days * 86400 });
      database.logModAction(message.guild.id, target.user.id, interaction.user.id, 'ban', reason);
      await sendModLog(message.guild, { action: 'ban', targetId: target.user.id, moderatorId: interaction.user.id, reason });
      const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('🔨 Kullanıcı Yasaklandı')
        .addFields(
          { name: 'Kullanıcı', value: `${target.user.tag} (\`${target.user.id}\`)`, inline: true },
          { name: 'Yetkili', value: `<@${interaction.user.id}>`, inline: true },
          { name: 'Sebep', value: reason }
        ).setFooter({ text: days > 0 ? `Son ${days} günlük mesajlar silindi` : 'Mesajlar silinmedi' }).setTimestamp();
      await interaction.followUp({ embeds: [embed] });
    });
  }
};
