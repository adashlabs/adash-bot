const {
  PermissionFlagsBits, ChannelType, EmbedBuilder, ActionRowBuilder,
  ChannelSelectMenuBuilder, RoleSelectMenuBuilder, ButtonBuilder, ButtonStyle
} = require('discord.js');
const database = require('../database');
const { buildTicketPanel } = require('../tickets');
const { COLORS } = require('../utils/ui');

function buildTicketSetupWizard(guild) {
  const settings = database.getSettings(guild.id);
  const supportRoleId = database.getConfig(guild.id, 'ticket_support_role_id');
  const panelTitle = database.getConfig(guild.id, 'ticket_panel_title', '🎫 Destek Talebi');
  const panelChannelId = database.getConfig(guild.id, 'ticket_panel_channel_id');

  const channelStr = (id) => id ? `<#${id}>` : '`Ayarlanmadı`';
  const roleStr = (id) => id ? `<@&${id}>` : '`Ayarlanmadı`';

  const embed = new EmbedBuilder()
    .setColor(COLORS.primary)
    .setTitle('🎫 Etkileşimli Ticket Kurulum Paneli')
    .setDescription([
      'Aşağıdaki açılır menüler ve butonları kullanarak ticket sisteminizi kolayca yapılandırın.',
      '',
      `📁 **Kategori:** ${channelStr(settings.ticket_category_id)}`,
      `📌 **Panel Kanalı:** ${channelStr(panelChannelId)}`,
      `📜 **Log Kanalı:** ${channelStr(settings.ticket_log_channel_id)}`,
      `🛡️ **Destek Rolü:** ${roleStr(supportRoleId)}`,
      `📝 **Panel Başlığı:** ${panelTitle}`
    ].join('\n'))
    .setFooter({ text: 'Menülerden seçim yaptıktan sonra "🚀 Paneli Gönder" butonuna basın.' })
    .setTimestamp();

  const rows = [
    new ActionRowBuilder().addComponents(
      new ChannelSelectMenuBuilder()
        .setCustomId(`setup_channel:ticketcategory:${guild.id}`)
        .setPlaceholder('1. Ticket Kategorisini Seç (📁 Kategori)')
        .setChannelTypes(ChannelType.GuildCategory)
    ),
    new ActionRowBuilder().addComponents(
      new ChannelSelectMenuBuilder()
        .setCustomId(`ticketsetup_panelchan:${guild.id}`)
        .setPlaceholder('2. Ticket Panelinin Gönderileceği Kanalı Seç (📌 Metin Kanalı)')
        .setChannelTypes(ChannelType.GuildText)
    ),
    new ActionRowBuilder().addComponents(
      new ChannelSelectMenuBuilder()
        .setCustomId(`setup_channel:ticketlog:${guild.id}`)
        .setPlaceholder('3. Ticket Log Kanalını Seç (📜 Metin Kanalı)')
        .setChannelTypes(ChannelType.GuildText)
    ),
    new ActionRowBuilder().addComponents(
      new RoleSelectMenuBuilder()
        .setCustomId(`setup_role:ticketsupport:${guild.id}`)
        .setPlaceholder('4. Ticket Destek Rolünü Seç (🛡️ Rol)')
    ),
    new ActionRowBuilder().addComponents(
      new ButtonBuilder().setCustomId(`setup_ticket_modal:${guild.id}`).setLabel('Metinleri Düzenle').setEmoji('📝').setStyle(ButtonStyle.Primary),
      new ButtonBuilder().setCustomId(`ticketsetup_deploy:${guild.id}`).setLabel('Paneli Gönder').setEmoji('🚀').setStyle(ButtonStyle.Success)
    )
  ];

  return { embeds: [embed], components: rows };
}
module.exports = {
  name: 'ticketsetup',
  aliases: ['ticketkurulum', 'destekkur'],
  category: 'ayarlar',
  description: 'etkileşimli menüler, butonlar ve formlarla ticket sistemini kurar.',
  cooldown: 5,
  buildTicketSetupWizard,
  async execute(message, args) {
    if (!message.member.permissions.has(PermissionFlagsBits.ManageGuild)) {
      return message.reply('bu kurulum için `Sunucuyu Yönet` yetkisi gerekiyor.');
    }

    const categoryInput = String(args[0] || '').trim();
    const allCategories = message.guild.channels.cache.filter((c) => c.type === ChannelType.GuildCategory);
    
    // Resolve category strictly by ID, name, or mention
    const category = allCategories.get(categoryInput) ||
      allCategories.find((c) => c.name.toLocaleLowerCase('tr-TR') === categoryInput.toLocaleLowerCase('tr-TR')) ||
      allCategories.find((c) => c.name.toLocaleLowerCase('tr-TR').includes(categoryInput.toLocaleLowerCase('tr-TR'))) ||
      message.mentions.channels.find((c) => c.type === ChannelType.GuildCategory);

    const textChannels = message.mentions.channels.filter((c) => c.isTextBased() && c.type !== ChannelType.GuildCategory);
    const panelChannel = textChannels.first() ||
      message.guild.channels.cache.find((c) => (c.type === ChannelType.GuildText || c.isTextBased()) && c.id === args[1]);
    const logChannel = textChannels.size > 1 ? textChannels.at(1) :
      message.guild.channels.cache.find((c) => (c.type === ChannelType.GuildText || c.isTextBased()) && c.id === args[2]);
    const supportRole = message.mentions.roles.first() ||
      message.guild.roles.cache.get(args[3]);

    if (!categoryInput && args.length === 0) {
      return message.reply(buildTicketSetupWizard(message.guild));
    }

    if (!category || !panelChannel) {
      return message.reply(buildTicketSetupWizard(message.guild));
    }

    const botPerms = panelChannel.permissionsFor(message.guild.members.me);
    if (!botPerms?.has([PermissionFlagsBits.ViewChannel, PermissionFlagsBits.SendMessages, PermissionFlagsBits.EmbedLinks])) {
      return message.reply(`botun ${panelChannel} kanalında mesaj ve embed gönderme yetkisi yok.`);
    }

    // Save configurations
    database.setSetting(message.guild.id, 'ticket_category_id', category.id);
    if (logChannel) database.setSetting(message.guild.id, 'ticket_log_channel_id', logChannel.id);
    if (supportRole) database.setConfig(message.guild.id, 'ticket_support_role_id', supportRole.id);

    // Send Ticket Panel to designated panel channel
    const panelMsg = await panelChannel.send(buildTicketPanel(message.guild)).catch(() => null);
    if (!panelMsg) {
      return message.reply(`${panelChannel} kanalına panel gönderilemedi. Kanal izinlerini kontrol et.`);
    }

    const embed = new EmbedBuilder()
      .setColor(COLORS.success)
      .setTitle('🎫 Ticket Sistemi Başarıyla Kuruldu')
      .setDescription('Ticket sistemi tüm kanalları ve ayarları ile tek adımda yapılandırıldı.')
      .addFields(
        { name: 'Kategori', value: `${category.name} (\`${category.id}\`)`, inline: true },
        { name: 'Panel Kanalı', value: `${panelChannel} ([Paneli Gör](${panelMsg.url}))`, inline: true },
        { name: 'Log Kanalı', value: logChannel ? `${logChannel}` : '`Ayarlanmadı`', inline: true },
        { name: 'Destek Rolü', value: supportRole ? `${supportRole}` : '`Yalnızca Yetkililer`', inline: true }
      )
      .setFooter({ text: `Kurulumu yapan: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
      .setTimestamp();
    await message.reply({ embeds: [embed] });
  }
};
