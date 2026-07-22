const { PermissionFlagsBits } = require('discord.js');
const { resolveTarget, moderationError } = require('../utils/security');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');
const database = require('../database');

module.exports = {
  name: 'clearwarns', aliases: ['uyarıtemizle', 'uyaritemizle'], category: 'moderasyon',
  description: 'kullanıcının aktif uyarılarını düğmeli onay sonrasında temizler.', cooldown: 3,
  async execute(message, args) {
    const target = await resolveTarget(message, args[0]);
    if (!target?.member) return message.reply('sunucudaki bir kullanıcıyı belirt.');
    const options = { userPermission: PermissionFlagsBits.ModerateMembers, action: 'moderatable' };
    const error = moderationError(message, target.member, options);
    if (error) return message.reply(error);
    const count = database.getUserWarnCount(message.guild.id, target.user.id);
    if (!count) return message.reply('bu kullanıcının aktif uyarısı yok.');
    await requestConfirmation(message, { title: 'Uyarıları Temizle', target: `${target.user.tag} (\`${target.user.id}\`)`, reason: `${count} aktif uyarı temizlenecek` }, async (interaction) => {
      const member = await message.guild.members.fetch(target.user.id).catch(() => null);
      const freshError = moderationError({ ...message, member: interaction.member }, member, options);
      if (freshError) return interaction.followUp({ content: freshError, flags: 64 });
      const cleared = database.clearWarnings(message.guild.id, target.user.id);
      const reason = `${cleared} aktif uyarı temizlendi`;
      database.logModAction(message.guild.id, target.user.id, interaction.user.id, 'clearwarns', reason);
      await sendModLog(message.guild, { action: 'clearwarns', targetId: target.user.id, moderatorId: interaction.user.id, reason, color: 0x57F287 });
      await interaction.followUp(`✅ **${target.user.tag}** kullanıcısının **${cleared}** aktif uyarısı temizlendi.`);
    });
  }
};
