const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const database = require('../database');
const { COLORS } = require('../utils/ui');

module.exports = {
  name: 'games',
  aliases: ['oyunlar', 'oyundurumu'],
  category: 'oyun',
  description: 'sayı saymaca ve kelime türetmece durumunu ve kurallarını gösterir.',
  async execute(message) {
    const settings = database.getSettings(message.guild.id);
    const state = database.getGameState(message.guild.id);
    const embed = new EmbedBuilder().setColor(COLORS.primary).setTitle('🎮 Sunucu Kanal Oyunları')
      .addFields(
        {
          name: '🔢 Sayı Saymaca Oyunu',
          value: settings.counting_enabled
            ? `✅ **Açık** · Kanal: <#${settings.counting_channel_id}>\n- Mevcut Sayı: **${state.counting_value}**\n- Sıradaki Sayı: **${state.counting_value + 1}**\n- Kural: Bir üye üst üste 2 kez yazamaz.`
            : '⛔ **Kapalı** · Yetkililer `a!setup` veya `/kurulum` ile aktifleştirebilir.'
        },
        {
          name: '🔤 Kelime Türetmece Oyunu',
          value: settings.word_chain_enabled
            ? `✅ **Açık** · Kanal: <#${settings.word_chain_channel_id}>\n- Son Kelime: **${state.last_word || 'Henüz yok'}**\n- Kural: Kelimeler TDK/Türkçe sözlükte olmalı, son harfle başlamalı ve tekrar edilmemeli.`
            : '⛔ **Kapalı** · Yetkililer `a!setup` veya `/kurulum` ile aktifleştirebilir.'
        }
      )
      .setFooter({ text: `İsteyen: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
      .setTimestamp();
    const components = [];
    const row = new ActionRowBuilder();
    if (settings.counting_enabled && settings.counting_channel_id) {
      row.addComponents(new ButtonBuilder().setStyle(ButtonStyle.Link).setURL(`https://discord.com/channels/${message.guild.id}/${settings.counting_channel_id}`).setLabel('Sayı Saymaca').setEmoji('🔢'));
    }
    if (settings.word_chain_enabled && settings.word_chain_channel_id) {
      row.addComponents(new ButtonBuilder().setStyle(ButtonStyle.Link).setURL(`https://discord.com/channels/${message.guild.id}/${settings.word_chain_channel_id}`).setLabel('Kelime Türetmece').setEmoji('🔤'));
    }
    if (row.components.length > 0) components.push(row);

    await message.reply({ embeds: [embed], components });
  }
};
