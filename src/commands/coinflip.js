module.exports = {
  name: 'coinflip',
  aliases: ['yazıtura', 'yazitura'],
  category: 'eglence',
  description: 'yazı tura atar.',
  cooldown: 2,
  async execute(message) {
    const isHeads = Math.random() < 0.5;
    const result = isHeads ? 'Yazı' : 'Tura';
    const emoji = isHeads ? '🪙' : '🪙';
    const embed = new (require('discord.js').EmbedBuilder)()
      .setColor(0x57F287)
      .setTitle(`${emoji} Yazı Tura Atıldı`)
      .setDescription(`Para havaya fırlatıldı ve düştü:\n\n### **${result}!**`)
      .setFooter({ text: `Atan: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
      .setTimestamp();
    const row = new (require('discord.js').ActionRowBuilder)().addComponents(
      new (require('discord.js').ButtonBuilder)().setCustomId('coinflip_retry').setLabel('Tekrar Fırlat').setEmoji('🪙').setStyle(require('discord.js').ButtonStyle.Success)
    );
    await message.reply({ embeds: [embed], components: [row] });
  }
};
