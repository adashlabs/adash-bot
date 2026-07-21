const { PermissionFlagsBits } = require('discord.js');

module.exports = {
  name: 'clear',
  category: 'moderasyon',
  description: 'belirtilen sayıda mesajı siler. kullanım: a!clear <sayı> [@kullanıcı]',
  async execute(message, args, client) {
    if (!message.member.permissions.has(PermissionFlagsBits.ManageMessages)) {
      return message.reply('bu komut için `mesajları yönet` yetkisi gerekiyor.');
    }
    if (!message.guild.members.me.permissions.has(PermissionFlagsBits.ManageMessages)) {
      return message.reply('benim `mesajları yönet` yetkim yok.');
    }

    const countInput = parseInt(args[0], 10);
    if (!Number.isFinite(countInput) || countInput < 1 || countInput > 100) {
      return message.reply('1 ile 100 arası bir sayı belirt. örnek: `a!clear 50`');
    }

    const targetUser = message.mentions.users.first();
    const deleteCount = Math.min(countInput, 100);

    try {
      await message.delete();
    } catch (_) {}

    let collected;
    try {
      collected = await message.channel.messages.fetch({ limit: 100 });
    } catch (error) {
      console.error('[HATA] clear fetch:', error);
      return message.reply('mesajlar getirilirken hata oluştu.');
    }

    if (targetUser) {
      collected = collected.filter((m) => m.author.id === targetUser.id);
    }

    const toDelete = collected.first(deleteCount);
    if (toDelete.size === 0) {
      return message.reply(targetUser ? 'bu kullanıcıya ait silinecek mesaj bulunamadı.' : 'silinecek mesaj bulunamadı.');
    }

    try {
      const deleted = await message.channel.bulkDelete(toDelete, true);
      const suffix = targetUser ? ` (${targetUser.tag} ait)` : '';
      await message.channel.send(`🗑️ **${deleted.size}** mesaj silindi${suffix}.`).then((m) => {
        setTimeout(() => m.delete().catch(() => {}), 4000);
      });
    } catch (error) {
      console.error('[HATA] clear:', error);
      await message.reply('mesajlar silinirken hata oluştu (14 günden eski mesajlar tek tek silinemez).');
    }
  }
};