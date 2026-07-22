module.exports = {
  name: 'roll',
  aliases: ['zar'],
  category: 'eglence',
  description: 'zar atar. kullanım: a!roll veya a!roll 2d20',
  cooldown: 2,
  async execute(message, args) {
    const match = String(args[0] || '1d6').toLowerCase().match(/^(\d{1,2})d(\d{1,4})$/);
    if (!match) return message.reply('zar biçimi geçersiz. örnek: `a!roll 2d20`');
    const count = Number(match[1]);
    const sides = Number(match[2]);
    if (count < 1 || count > 20 || sides < 2 || sides > 1000) return message.reply('en fazla 20 zar ve 2–1000 yüz kullanabilirsin.');
    const rolls = Array.from({ length: count }, () => Math.floor(Math.random() * sides) + 1);
    const total = rolls.reduce((sum, value) => sum + value, 0);
    const embed = new (require('discord.js').EmbedBuilder)()
      .setColor(0x3498DB)
      .setTitle(`🎲 Zar Atıldı: ${count}d${sides}`)
      .addFields(
        { name: 'Gelen Zarlar', value: rolls.map((value) => `\`${value}\``).join(' ') },
        ...(count > 1 ? [{ name: 'Toplam', value: `**${total}**`, inline: true }] : [])
      )
      .setFooter({ text: `Atan: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
      .setTimestamp();
    const row = new (require('discord.js').ActionRowBuilder)().addComponents(
      new (require('discord.js').ButtonBuilder)().setCustomId(`roll_retry:${count}:${sides}`).setLabel('Tekrar Zar At').setEmoji('🎲').setStyle(require('discord.js').ButtonStyle.Primary)
    );
    await message.reply({ embeds: [embed], components: [row] });
  }
};
