const { PermissionFlagsBits } = require('discord.js');
const { parseDuration, formatDuration } = require('../utils/resolvers');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');

module.exports = {
  name: 'slowmode', aliases: ['yavaşmod', 'yavasmod'], category: 'moderasyon',
  description: 'kanal yavaş modunu 0 saniye–6 saat arasında onayla ayarlar.', cooldown: 3,
  async execute(message, args) {
    if (!message.channel.permissionsFor(message.member).has(PermissionFlagsBits.ManageChannels)) return message.reply('bu işlem için `Kanalları Yönet` yetkisi gerekiyor.');
    if (!message.channel.permissionsFor(message.guild.members.me).has(PermissionFlagsBits.ManageChannels)) return message.reply('botun bu kanalda `Kanalları Yönet` yetkisi yok.');
    const durationMs = args[0] === '0' ? 0 : parseDuration(args[0]);
    if (durationMs === null || durationMs < 0 || durationMs > 6 * 60 * 60 * 1000) return message.reply('0 veya 1s–6h arasında süre belirt. Örnek: `a!slowmode 10s`');
    const seconds = Math.floor(durationMs / 1000);
    await requestConfirmation(message, { title: 'Yavaş Modu Ayarla', target: `<#${message.channel.id}>`, reason: seconds ? formatDuration(durationMs) : 'Kapalı' }, async (interaction) => {
      if (!message.channel.permissionsFor(interaction.member).has(PermissionFlagsBits.ManageChannels)) return interaction.followUp({ content: 'kanal yönetme yetkin artık yok.', flags: 64 });
      await message.channel.setRateLimitPerUser(seconds, `${interaction.user.tag} yavaş mod`);
      await sendModLog(message.guild, { action: 'slowmode', moderatorId: interaction.user.id, reason: seconds ? `${seconds} saniye` : 'kapalı', color: 0x5865F2 });
      await interaction.followUp(seconds ? `⏱️ Yavaş mod **${formatDuration(durationMs)}** olarak ayarlandı.` : '⏱️ Yavaş mod kapatıldı.');
    });
  }
};
