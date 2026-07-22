const { PermissionFlagsBits, EmbedBuilder } = require('discord.js');
const database = require('../database');
const { COLORS } = require('../utils/ui');

module.exports = {
  name: 'prefix',
  category: 'ayarlar',
  description: 'sunucunun komut ön ekini değiştirir. kullanım: a!prefix !',
  cooldown: 3,
  async execute(message, args) {
    if (!message.member.permissions.has(PermissionFlagsBits.ManageGuild)) {
      const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('❌ Yetersiz Yetki')
        .setDescription('Bu ayarı değiştirmek için `Sunucuyu Yönet` yetkisine sahip olmalısın.');
      return message.reply({ embeds: [embed] });
    }
    const prefix = args[0];
    if (!prefix || prefix.length > 5 || /\s|[`<>{}]/.test(prefix)) {
      const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle('⚠️ Geçersiz Prefix')
        .setDescription('Boşluk ve özel mention karakterleri içermeyen, en fazla 5 karakterlik bir prefix seç. Örnek: `a!prefix !`');
      return message.reply({ embeds: [embed] });
    }
    database.setPrefix(message.guild.id, prefix);
    const embed = new EmbedBuilder().setColor(COLORS.success).setTitle('✅ Prefix Değiştirildi')
      .addFields(
        { name: 'Yeni Prefix', value: `\`${prefix}\``, inline: true },
        { name: 'Yardım Menüsü', value: `\`${prefix}help\``, inline: true }
      )
      .setFooter({ text: `Değiştiren: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
      .setTimestamp();
    await message.reply({ embeds: [embed] });
  }
};
