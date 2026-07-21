const { PermissionFlagsBits } = require('discord.js');
const { resolveUser } = require('../utils/resolvers');
const database = require('../database');

module.exports = {
  name: 'unmute',
  category: 'moderasyon',
  description: 'kullanıcının susturulmasını kaldırır. kullanım: a!unmute <@kullanıcı|id> [sebep]',
  async execute(message, args, client) {
    if (!message.member.permissions.has(PermissionFlagsBits.ModerateMembers)) {
      return message.reply('bu komut için `üyeleri zaman aşımı` yetkisi gerekiyor.');
    }
    if (!message.guild.members.me.permissions.has(PermissionFlagsBits.ModerateMembers)) {
      return message.reply('benim `üyeleri zaman aşımı` yetkim yok.');
    }

    const target = await resolveUser(client, args[0]);
    if (!target) {
      return message.reply('geçerli bir kullanıcı belirt. örnek: `a!unmute @user`');
    }

    const member = await message.guild.members.fetch(target.id).catch(() => null);
    if (!member) {
      return message.reply('bu kullanıcı sunucuda bulunamadı.');
    }

    if (!member.isCommunicationDisabled()) {
      return message.reply(`**${target.tag}** zaten susturulmuş değil.`);
    }

    const reason = args.slice(1).join(' ') || 'sebep belirtilmedi';

    try {
      await member.timeout(null, reason);
      database.logModAction(message.guild.id, target.id, message.author.id, 'unmute', reason);
      await message.reply(`🔊 **${target.tag}**ın susturulması kaldırıldı. sebep: ${reason}`);
    } catch (error) {
      console.error('[HATA] unmute:', error);
      await message.reply('susturma kaldırma işlemi başarısız oldu.');
    }
  }
};