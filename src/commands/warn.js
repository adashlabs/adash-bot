const { PermissionFlagsBits } = require('discord.js');
const { resolveUser } = require('../utils/resolvers');
const database = require('../database');

module.exports = {
  name: 'warn',
  category: 'moderasyon',
  description: 'kullanıcıya uyarı verir. kullanım: a!warn <@kullanıcı|id> [sebep]',
  async execute(message, args, client) {
    if (!message.member.permissions.has(PermissionFlagsBits.ModerateMembers)) {
      return message.reply('bu komut için `üyeleri zaman aşımı` yetkisi gerekiyor.');
    }

    const target = await resolveUser(client, args[0]);
    if (!target) {
      return message.reply('geçerli bir kullanıcı belirt. örnek: `a!warn @user kural ihlali`');
    }

    if (target.id === message.author.id) {
      return message.reply('kendine uyarı veremezsin.');
    }
    if (target.id === client.user.id) {
      return message.reply('bana uyarı veremezsin.');
    }

    const reason = args.slice(1).join(' ') || 'sebep belirtilmedi';

    database.logModAction(message.guild.id, target.id, message.author.id, 'warn', reason);
    const warnCount = database.getUserWarnCount(message.guild.id, target.id, 'warn');
    const total = warnCount;

    await message.reply(`⚠️ **${target.tag}** uyarıldı. sebep: ${reason}\n\`bu kullanıcının toplam uyarı sayısı: ${total}\``);

    if (total >= 3) {
      await message.channel.send(`\uD83D\uDD12 **${target.tag}** ${total} uyarıya ulaştı. otomatik susturuluyor (10 dakika)...`);
      const member = await message.guild.members.fetch(target.id).catch(() => null);
      if (member && member.moderatable) {
        try {
          await member.timeout(10 * 60 * 1000, `${total} uyarı sonrası otomatik susturma`);
          database.logModAction(message.guild.id, target.id, client.user.id, 'mute', `${total} uyarı sonrası otomatik susturma`, 10 * 60 * 1000);
        } catch (_) {}
      }
    }
  }
};