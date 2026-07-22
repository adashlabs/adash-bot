const { EmbedBuilder, ActionRowBuilder, StringSelectMenuBuilder } = require('discord.js');
const { COLORS, truncate } = require('../utils/ui');

const CATEGORIES = {
  genel: ['📌', 'Genel'],
  moderasyon: ['🛡️', 'Moderasyon'],
  oyun: ['🎮', 'Oyun'],
  eglence: ['🎉', 'Eğlence'],
  ayarlar: ['⚙️', 'Ayarlar']
};

function buildHelp(client, prefix, category = 'genel', ownerId = '0') {
  const commands = [...client.commands.values()]
    .filter((command) => (command.category || 'genel') === category && !command.hidden)
    .sort((a, b) => a.name.localeCompare(b.name, 'tr'));
  const [emoji, label] = CATEGORIES[category] || CATEGORIES.genel;

  const embed = new EmbedBuilder()
    .setColor(COLORS.primary)
    .setTitle(`${emoji} ${label} Komutları`)
    .setDescription(commands.length
      ? commands.map((command) => `**${prefix}${command.name}** — ${truncate(command.description, 180)}`).join('\n')
      : 'Bu kategoride komut bulunmuyor.')
    .setFooter({ text: 'Aşağıdaki menüden kategori seçebilirsin.' });

  const menu = new StringSelectMenuBuilder()
    .setCustomId(`help_select:${ownerId}`)
    .setPlaceholder('Komut kategorisi seç')
    .addOptions(Object.entries(CATEGORIES).map(([value, [icon, name]]) => ({
      label: name,
      value,
      emoji: icon,
      default: value === category
    })));

  return { embeds: [embed], components: [new ActionRowBuilder().addComponents(menu)] };
}

module.exports = {
  name: 'help',
  aliases: ['yardim', 'yardım', 'komutlar'],
  category: 'genel',
  description: 'etkileşimli komut ve yardım menüsünü açar.',
  buildHelp,
  async execute(message, args, client) {
    const prefix = require('../database').getPrefix(message.guild.id);
    const query = args[0]?.toLocaleLowerCase('tr-TR');
    if (query) {
      const canonicalName = client.aliases?.get(query) || query;
      const command = client.commands.get(canonicalName);
      if (command) {
        const embed = new EmbedBuilder()
          .setColor(COLORS.primary)
          .setTitle(`📌 Komut: ${prefix}${command.name}`)
          .setDescription(command.description || 'Açıklama belirtilmemiş.')
          .addFields(
            { name: 'Kategori', value: command.category || 'genel', inline: true },
            { name: 'Bekleme Süresi', value: `${command.cooldown || 2} saniye`, inline: true },
            { name: 'Takma Adlar (Aliases)', value: Array.isArray(command.aliases) && command.aliases.length ? command.aliases.map((a) => `\`${prefix}${a}\``).join(', ') : 'Yok' }
          )
          .setFooter({ text: `İsteyen: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
          .setTimestamp();
        return message.reply({ embeds: [embed] });
      }
    }
    await message.reply(buildHelp(client, prefix, 'genel', message.author.id));
  }
};
