const { EmbedBuilder } = require('discord.js');
const { COLORS, truncate } = require('../utils/ui');
const database = require('../database');

module.exports = {
  name: 'appeal', aliases: ['itiraz'], category: 'moderasyon',
  description: 'yetkililere gizli bir moderasyon itirazı gönderir.', cooldown: 30,
  async execute(message, args) {
    const text = truncate(args.join(' ').trim(), 1500);
    if (text.length < 20) return message.reply('itirazını en az 20 karakterle ayrıntılı yaz.');
    const channelId = database.getConfig(message.guild.id, 'appeal_channel_id');
    if (!channelId) return message.reply('bu sunucuda itiraz kanalı ayarlanmamış. Bir yetkiliyle iletişime geç.');
    const channel = await message.guild.channels.fetch(channelId).catch(() => null);
    if (!channel?.isTextBased()) return message.reply('itiraz kanalı artık erişilebilir değil; yöneticiler kurulumu güncellemeli.');
    await channel.send({ embeds: [new EmbedBuilder().setColor(COLORS.warning).setTitle('📨 Moderasyon İtirazı')
      .setDescription(text).addFields({ name: 'Gönderen', value: `${message.author.tag} (\`${message.author.id}\`)` }, { name: 'Gönderildi', value: `<t:${Math.floor(Date.now() / 1000)}:F>` })
      .setFooter({ text: 'İtirazı herkese açık kanallarda paylaşmayın.' })], allowedMentions: { parse: [] } });
    await message.reply('itirazın yetkili ekibe iletildi.');
  }
};
