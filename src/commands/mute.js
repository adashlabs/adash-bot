const { PermissionFlagsBits } = require('discord.js');
const { resolveUser, parseDuration, formatDuration } = require('../utils/resolvers');
const database = require('../database');

module.exports = {
  name: 'mute',
  category: 'moderasyon',
  description: 'kullan\u0131c\u0131y\u0131 susturur (timeout). kullan\u0131m: a!mute <@kullan\u0131c\u0131|id> <s\u00FCre> [sebep]',
  async execute(message, args, client) {
    if (!message.member.permissions.has(PermissionFlagsBits.ModerateMembers)) {
      return message.reply('bu komut i\u00E7in `\u00FCyeleri zaman a\u015F\u0131m\u0131` yetkisi gerekiyor.');
    }
    if (!message.guild.members.me.permissions.has(PermissionFlagsBits.ModerateMembers)) {
      return message.reply('benim `\u00FCyeleri zaman a\u015F\u0131m\u0131` yetkim yok.');
    }

    const target = await resolveUser(client, args[0]);
    if (!target) {
      return message.reply('ge\u00E7erli bir kullan\u0131c\u0131 belirt. \u00F6rnek: `a!mute @user 10m sebep`');
    }

    if (target.id === message.author.id) {
      return message.reply('kendini susturamazs\u0131n.');
    }
    if (target.id === client.user.id) {
      return message.reply('kendimi susturamam.');
    }

    const member = await message.guild.members.fetch(target.id).catch(() => null);
    if (!member) {
      return message.reply('bu kullan\u0131c\u0131 sunucuda bulunamad\u0131.');
    }

    if (!member.moderatable) {
      return message.reply('bu kullan\u0131c\u0131y\u0131 susturam\u0131yorum (yetki hiyerar\u015Fisi).');
    }

    if (member.isCommunicationDisabled()) {
      return message.reply(`**${target.tag}** zaten susturulmu\u015F durumda.`);
    }

    const durationInput = args[1];
    if (!durationInput) {
      return message.reply('bir s\u00FCre belirt. \u00F6rnek: `a!mute @user 10m spam`');
    }

    const durationMs = parseDuration(durationInput);
    if (!durationMs || durationMs < 1000) {
      return message.reply('ge\u00E7ersiz s\u00FCre. \u00F6rnek: `10s`, `5m`, `1h`, `1d`');
    }

    const maxDuration = 28 * 24 * 60 * 60 * 1000;
    if (durationMs > maxDuration) {
      return message.reply('s\u00FCre en fazla 28 g\u00FCn olabilir.');
    }

    const reason = args.slice(2).join(' ') || 'sebep belirtilmedi';

    try {
      await member.timeout(durationMs, reason);
      database.logModAction(message.guild.id, target.id, message.author.id, 'mute', reason, durationMs);
      await message.reply(`\uD83D\uDD07 **${target.tag}** ${formatDuration(durationMs)} boyunca susturuldu. sebep: ${reason}`);
    } catch (error) {
      console.error('[HATA] mute:', error);
      await message.reply('susturma i\u015Flemi ba\u015Far\u0131s\u0131z oldu.');
    }
  }
};