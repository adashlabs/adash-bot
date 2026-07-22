const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { COLORS } = require('../utils/ui');

module.exports = {
  name: 'serverinfo',
  aliases: ['sunucu', 'sunucubilgi'],
  category: 'genel',
  description: 'sunucunun sahibi, üye, kanal, rol ve kuruluş bilgilerini gösterir.',
  async execute(message) {
    const guild = message.guild;
    const owner = await guild.fetchOwner().catch(() => null);
    const textCount = guild.channels.cache.filter((c) => c.type === 0).size;
    const voiceCount = guild.channels.cache.filter((c) => c.type === 2).size;
    const categoryCount = guild.channels.cache.filter((c) => c.type === 4).size;
    const emojiCount = guild.emojis.cache.size;

    const embed = new EmbedBuilder()
      .setColor(COLORS.primary)
      .setTitle(`🏠 ${guild.name}`)
      .setThumbnail(guild.iconURL({ size: 512, extension: 'png' }))
      .addFields(
        { name: 'Sahip', value: owner ? `${owner.user.tag} (<@${owner.id}>)` : 'Bilinmiyor', inline: true },
        { name: 'Sunucu ID', value: `\`${guild.id}\``, inline: true },
        { name: 'Kuruluş Tarihi', value: `<t:${Math.floor(guild.createdTimestamp / 1000)}:F> (<t:${Math.floor(guild.createdTimestamp / 1000)}:R>)` },
        { name: 'Toplam Üye', value: `👥 **${guild.memberCount}** üye`, inline: true },
        { name: 'Kanallar', value: `💬 ${textCount} Metin · 🔊 ${voiceCount} Ses · 📁 ${categoryCount} Kategori`, inline: true },
        { name: 'Rol & Emojiler', value: `🎭 ${guild.roles.cache.size} Rol · 😀 ${emojiCount} Emoji`, inline: true },
        { name: 'Takviye (Boost)', value: `💎 Seviye **${guild.premiumTier}** (${guild.premiumSubscriptionCount || 0} Takviye)` }
      )
      .setFooter({ text: `İsteyen: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
      .setTimestamp();

    const iconUrl = guild.iconURL({ size: 4096, extension: 'png' });
    const bannerUrl = guild.bannerURL({ size: 4096, extension: 'png' });
    const components = [];
    if (iconUrl || bannerUrl) {
      const row = new ActionRowBuilder();
      if (iconUrl) row.addComponents(new ButtonBuilder().setStyle(ButtonStyle.Link).setURL(iconUrl).setLabel('Sunucu İkonu').setEmoji('🖼️'));
      if (bannerUrl) row.addComponents(new ButtonBuilder().setStyle(ButtonStyle.Link).setURL(bannerUrl).setLabel('Sunucu Afişi').setEmoji('🎨'));
      components.push(row);
    }

    await message.reply({ embeds: [embed], components });
  }
};
