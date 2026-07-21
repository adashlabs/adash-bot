const { PermissionFlagsBits } = require('discord.js');
const { resolveUser } = require('../utils/resolvers');
const database = require('../database');

module.exports = {
  name: 'kick',
  category: 'moderasyon',
  description: 'kullan\u0131c\u0131y\u0131 sunucudan atar. kullan\u0131m: a!kick <@kullan\u0131c\u0131|id> [sebep]',
  async execute(message, args, client) {
    if (!message.member.permissions.has(PermissionFlagsBits.KickMembers)) {
      return message.reply('bu komut i\u00E7in `\u00FCyeleri at` yetkisi gerekiyor.');
    }
    if (!message.guild.members.me.permissions.has(PermissionFlagsBits.KickMembers)) {
      return message.reply('benim `\u00FCyeleri at` yetkim yok.');
    }

    const target = await resolveUser(client, args[0]);
    if (!target) {
      return message.reply('ge\u00E7erli bir kullan\u0131c\u0131 belirt. \u00F6rnek: `a!kick @user sebep`');
    }

    if (target.id === message.author.id) {
      return message.reply('kendini atamazs\u0131n.');
    }
    if (target.id === client.user.id) {
      return message.reply('kendimi atamam.');
    }

    const member = await message.guild.members.fetch(target.id).catch(() => null);
    if (!member) {
      return message.reply('bu kullan\u0131c\u0131 sunucuda bulunamad\u0131.');
    }

    if (!member.kickable) {
      return message.reply('bu kullan\u0131c\u0131y\u0131 atam\u0131yorum (yetki hiyerar\u015Fisi).');
    }

    const reason = args.slice(1).join(' ') || 'sebep belirtilmedi';

    try {
      await member.kick(reason);
      database.logModAction(message.guild.id, target.id, message.author.id, 'kick', reason);
      await message.reply(`\uD83D\uDC62 **${target.tag}** sunucudan at\u0131ld\u0131. sebep: ${reason}`);
    } catch (error) {
      console.error('[HATA] kick:', error);
      await message.reply('atma i\u015Flemi ba\u015Far\u0131s\u0131z oldu.');
    }
  }
};