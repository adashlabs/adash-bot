const { PermissionFlagsBits } = require('discord.js');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');

module.exports = {
  name: 'lock', aliases: ['kilit'], category: 'moderasyon',
  description: 'kanalı düğmeli onay sonrasında mesaj yazmaya kapatır veya açar.', cooldown: 3,
  async execute(message, args) {
    if (!message.channel.permissionsFor(message.member).has(PermissionFlagsBits.ManageChannels)) return message.reply('bu işlem için `Kanalları Yönet` yetkisi gerekiyor.');
    if (!message.channel.permissionsFor(message.guild.members.me).has(PermissionFlagsBits.ManageChannels)) return message.reply('botun bu kanalda `Kanalları Yönet` yetkisi yok.');
    const unlock = ['aç', 'ac', 'unlock'].includes(args[0]?.toLocaleLowerCase('tr-TR'));
    await requestConfirmation(message, {
      title: unlock ? 'Kanal Kilidini Aç' : 'Kanalı Kilitle', target: `<#${message.channel.id}>`,
      reason: unlock ? 'Üyeler tekrar mesaj gönderebilecek.' : 'Varsayılan üyeler mesaj gönderemeyecek.'
    }, async (interaction) => {
      if (!message.channel.permissionsFor(interaction.member).has(PermissionFlagsBits.ManageChannels)) return interaction.followUp({ content: 'kanal yönetme yetkin artık yok.', flags: 64 });
      await message.channel.permissionOverwrites.edit(message.guild.roles.everyone, { SendMessages: unlock ? null : false }, { reason: `${interaction.user.tag} kanal kilidi` });
      await sendModLog(message.guild, { action: unlock ? 'unlock' : 'lock', moderatorId: interaction.user.id, reason: `<#${message.channel.id}>`, color: unlock ? 0x57F287 : 0xED4245 });
      await interaction.followUp(unlock ? '🔓 Kanal kilidi açıldı.' : '🔒 Kanal mesaj göndermeye kapatıldı.');
    });
  }
};
