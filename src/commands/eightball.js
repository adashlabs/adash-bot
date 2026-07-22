const answers = [
  'Kesinlikle evet.', 'Büyük ihtimalle.', 'İşaretler olumlu.', 'Şimdilik evet.',
  'Buna güvenme.', 'Pek sanmıyorum.', 'Kesinlikle hayır.', 'Bunu sonra tekrar sor.',
  'Şu an söylemem daha iyi olmaz.', 'Sonuç belirsiz.'
];

module.exports = {
  name: '8ball',
  aliases: ['sihirliküre', 'sihirlikure'],
  category: 'eglence',
  description: 'sorduğun evet/hayır sorusuna sihirli küre cevap verir.',
  cooldown: 2,
  async execute(message, args) {
    const question = args.join(' ').trim();
    if (!question) return message.reply('bir soru sor. örnek: `a!8ball bugün şanslı mıyım?`');
    const answer = answers[Math.floor(Math.random() * answers.length)];
    const embed = new (require('discord.js').EmbedBuilder)()
      .setColor(0x5865F2)
      .setTitle('🎱 Sihirli 8Ball Küresi')
      .addFields(
        { name: 'Soru', value: question.slice(0, 500) },
        { name: 'Cevap', value: `✨ **${answer}**` }
      )
      .setFooter({ text: `Soran: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
      .setTimestamp();
    const row = new (require('discord.js').ActionRowBuilder)().addComponents(
      new (require('discord.js').ButtonBuilder)().setCustomId('8ball_retry').setLabel('Yeniden Sor').setEmoji('🎱').setStyle(require('discord.js').ButtonStyle.Primary)
    );
    await message.reply({ embeds: [embed], components: [row] });
  }
};
