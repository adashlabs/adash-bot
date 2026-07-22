const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { resolveTarget } = require('../utils/security');
const { COLORS } = require('../utils/ui');

module.exports = {
  name: 'avatar',
  aliases: ['pp'],
  category: 'genel',
  description: 'kullanıcının avatarını büyük boyutta ve indirme düğmesiyle gösterir.',
  async execute(message, args) {
    const resolved = args[0] ? await resolveTarget(message, args[0]) : { user: message.author };
    if (!resolved) return message.reply('geçerli bir kullanıcı belirt.');
    const url = resolved.user.displayAvatarURL({ extension: 'png', size: 4096 });
    const embed = new EmbedBuilder().setColor(COLORS.primary).setTitle(`🖼️ ${resolved.user.tag}`)
      .setImage(url).setFooter({ text: '4096×4096 PNG' });
    const row = new ActionRowBuilder().addComponents(new ButtonBuilder().setStyle(ButtonStyle.Link).setURL(url).setLabel('Avatarı Aç'));
    await message.reply({ embeds: [embed], components: [row] });
  }
};
