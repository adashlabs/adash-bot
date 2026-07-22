const { PermissionFlagsBits, EmbedBuilder } = require('discord.js');
const { sendModLog } = require('../utils/modLog');
const { transient, COLORS } = require('../utils/ui');
const database = require('../database');

module.exports = {
  name: 'clear', aliases: ['sil', 'temizle'], category: 'moderasyon',
  description: '1–100 mesajı onay beklemeden anında temizler.', cooldown: 3,
  async execute(message, args) {
    if (!message.channel.permissionsFor(message.member).has(PermissionFlagsBits.ManageMessages)) {
      const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('❌ Yetersiz Yetki')
        .setDescription('Mesaj silmek için `Mesajları Yönet` yetkisine sahip olmalısın.');
      return message.reply({ embeds: [embed] });
    }
    const botPermissions = message.channel.permissionsFor(message.guild.members.me);
    if (!botPermissions.has([PermissionFlagsBits.ManageMessages, PermissionFlagsBits.ReadMessageHistory])) {
      const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('❌ Bot Yetkisi Eksik')
        .setDescription('Botun bu kanalda mesaj yönetme ve geçmiş okuma yetkileri olmalı.');
      return message.reply({ embeds: [embed] });
    }
    const count = Number(args[0]);
    if (!Number.isInteger(count) || count < 1 || count > 100) {
      const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle('⚠️ Geçersiz Sayı')
        .setDescription('Lütfen 1 ile 100 arasında silinecek mesaj sayısı belirt. Örnek: `a!sil 20` veya `/temizle sayi:20`');
      return message.reply({ embeds: [embed] });
    }
    const target = message.mentions.users.first();
    const messages = await message.channel.messages.fetch({ limit: 100 });
    const candidates = messages.filter((entry) => entry.id !== message.id && (!target || entry.author.id === target.id)).first(count);
    const countToDelete = Array.isArray(candidates) ? candidates.length : candidates?.size || 0;
    if (!countToDelete) {
      const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle('⚠️ Silinecek Mesaj Bulunamadı')
        .setDescription('Silinecek uygun mesaj bulunamadı (14 günden eski mesajlar toplu silinemez).');
      return message.reply({ embeds: [embed] });
    }

    let deleted;
    try {
      deleted = await message.channel.bulkDelete(candidates, true);
    } catch (err) {
      const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('❌ Silme Hatası')
        .setDescription('Mesajlar silinirken hata oluştu (14 günden eski mesajlar toplu silinemez).');
      return message.reply({ embeds: [embed] });
    }

    const deletedCount = deleted.size ?? deleted.length ?? 0;
    await message.delete().catch(() => null);
    const reason = target ? `${target.tag} kullanıcısının mesajları` : 'Kanal mesajları';
    database.logModAction(message.guild.id, target?.id || message.channel.id, message.author.id, 'clear', reason, deletedCount);
    await sendModLog(message.guild, { action: 'clear', targetId: target?.id, moderatorId: message.author.id, reason, extra: `${deletedCount} mesaj · <#${message.channel.id}>`, color: 0x5865F2 });

    const embed = new EmbedBuilder().setColor(COLORS.success).setTitle('🗑️ Mesajlar Silindi')
      .setDescription(`**${deletedCount}** mesaj başarıyla temizlendi${target ? ` · <@${target.id}>` : ''}.`)
      .setFooter({ text: 'Bu mesaj 4 saniye içinde otomatik silinecektir.' });
    await transient(message.channel, { embeds: [embed] }, 4000);
  }
};
