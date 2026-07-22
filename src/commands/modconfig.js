const { PermissionFlagsBits, ChannelType, EmbedBuilder } = require('discord.js');
const { parseDuration, formatDuration } = require('../utils/resolvers');
const { COLORS } = require('../utils/ui');
const database = require('../database');

const usage = 'Kullanım: `a!modconfig warn <1-10> <10m-28d>` veya `a!modconfig appeal <#kanal|kapalı>`';

module.exports = {
  name: 'modconfig', aliases: ['modayar'], category: 'moderasyon',
  description: 'uyarı otomasyonu ve itiraz kanalını ayarlar.', cooldown: 3,
  async execute(message, args) {
    if (!message.member.permissions.has(PermissionFlagsBits.ManageGuild)) return message.reply('bu komut için `Sunucuyu Yönet` yetkisi gerekiyor.');
    const mode = String(args.shift() || '').toLocaleLowerCase('tr-TR');
    if (mode === 'warn') {
      const threshold = Number(args.shift());
      const duration = parseDuration(args.shift());
      if (!Number.isInteger(threshold) || threshold < 1 || threshold > 10 || !duration || duration < 10_000 || duration > 28 * 86400_000) return message.reply(usage);
      database.setConfig(message.guild.id, 'warn_auto_threshold', threshold);
      database.setConfig(message.guild.id, 'warn_auto_timeout_ms', duration);
      return message.reply({ embeds: [new EmbedBuilder().setColor(COLORS.success).setTitle('⚙️ Uyarı politikası kaydedildi')
        .setDescription(`${threshold}. aktif uyarıda kullanıcı **${formatDuration(duration)}** susturulacak.`)] });
    }
    if (mode === 'appeal') {
      const input = String(args[0] || '');
      if (['kapalı', 'kapali', 'off'].includes(input.toLocaleLowerCase('tr-TR'))) {
        database.setConfig(message.guild.id, 'appeal_channel_id', '');
        return message.reply('itiraz kanalı kapatıldı.');
      }
      const id = input.match(/^<#(\d{17,20})>$/)?.[1] || (input.match(/^\d{17,20}$/)?.[0]);
      const channel = id ? await message.guild.channels.fetch(id).catch(() => null) : null;
      if (!channel?.isTextBased() || channel.type === ChannelType.GuildCategory) return message.reply('geçerli bir metin kanalı belirt.');
      database.setConfig(message.guild.id, 'appeal_channel_id', channel.id);
      return message.reply(`itiraz kanalı ${channel} olarak ayarlandı.`);
    }
    return message.reply(usage);
  }
};
