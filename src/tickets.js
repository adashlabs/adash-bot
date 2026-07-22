const {
  ChannelType, PermissionFlagsBits, EmbedBuilder, ActionRowBuilder, ButtonBuilder,
  ButtonStyle, AttachmentBuilder, ModalBuilder, TextInputBuilder, TextInputStyle,
  StringSelectMenuBuilder
} = require('discord.js');
const database = require('./database');
const { COLORS, truncate } = require('./utils/ui');
const { sendModLog } = require('./utils/modLog');

function ticketConfig(guildId) {
  return {
    supportRoleId: database.getConfig(guildId, 'ticket_support_role_id'),
    panelTitle: database.getConfig(guildId, 'ticket_panel_title', '🎫 Destek Talebi'),
    panelDescription: database.getConfig(guildId, 'ticket_panel_description', 'Destek ekibimize ulaşmak için aşağıdaki düğmeye bas. Aynı anda yalnızca bir açık talebin olabilir.'),
    welcomeMessage: database.getConfig(guildId, 'ticket_welcome_message', '{user}, talebin oluşturuldu. Sorununu ayrıntılı biçimde anlat; ekibimiz kısa sürede ilgilenecek.'),
    buttonLabel: database.getConfig(guildId, 'ticket_panel_button_label', 'Destek Talebi Aç'),
    buttonEmoji: database.getConfig(guildId, 'ticket_panel_button_emoji', '🎫')
  };
}

function buildTicketPanel(guild) {
  const config = guild ? ticketConfig(guild.id) : ticketConfig('0');
  return {
    embeds: [new EmbedBuilder().setColor(COLORS.primary).setTitle(truncate(config.panelTitle, 256)).setDescription(truncate(config.panelDescription, 4000))
      .setFooter({ text: 'Bir kullanıcı aynı anda yalnızca bir açık talep oluşturabilir.' })],
    components: [new ActionRowBuilder().addComponents(
      new ButtonBuilder().setCustomId('ticket_open').setLabel(truncate(config.buttonLabel, 80)).setEmoji(config.buttonEmoji || '🎫').setStyle(ButtonStyle.Primary)
    )]
  };
}

function ticketOpenModal() {
  const modal = new ModalBuilder().setCustomId('ticket_open_modal').setTitle('Destek Talebi Oluştur');
  modal.addComponents(
    new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('subject').setLabel('Konu').setStyle(TextInputStyle.Short).setRequired(true).setMaxLength(100)),
    new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('description').setLabel('Sorunun / talebin').setStyle(TextInputStyle.Paragraph).setRequired(true).setMinLength(10).setMaxLength(1500)),
    new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('priority').setLabel('Öncelik: düşük / normal / yüksek / acil').setStyle(TextInputStyle.Short).setRequired(true).setValue('normal').setMaxLength(10))
  );
  return modal;
}

function ticketAddModal() {
  const modal = new ModalBuilder().setCustomId('ticket_add_modal').setTitle('Kanalına Üye Ekle');
  modal.addComponents(
    new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('user_input').setLabel('Eklenecek Üye ID veya Tag').setStyle(TextInputStyle.Short).setRequired(true).setPlaceholder('Örn: 123456789012345678'))
  );
  return modal;
}

function ticketRenameModal() {
  const modal = new ModalBuilder().setCustomId('ticket_rename_modal').setTitle('Ticket Kanalını Adlandır');
  modal.addComponents(
    new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('name_input').setLabel('Yeni Kanal Adı').setStyle(TextInputStyle.Short).setRequired(true).setMaxLength(80))
  );
  return modal;
}

function ticketCloseModal() {
  const modal = new ModalBuilder().setCustomId('ticket_close_modal').setTitle('Ticket Talebini Kapat');
  modal.addComponents(
    new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('reason_input').setLabel('Kapanış Sebebi / Notu').setStyle(TextInputStyle.Paragraph).setRequired(false).setValue('Çözüldü / Talebiniz tamamlandı.').setMaxLength(500))
  );
  return modal;
}

function ticketPermissions(guild, ownerId, supportRoleId) {
  const permissions = [
    { id: guild.id, deny: [PermissionFlagsBits.ViewChannel] },
    { id: ownerId, allow: [PermissionFlagsBits.ViewChannel, PermissionFlagsBits.SendMessages, PermissionFlagsBits.ReadMessageHistory, PermissionFlagsBits.AttachFiles] },
    { id: guild.members.me.id, allow: [PermissionFlagsBits.ViewChannel, PermissionFlagsBits.SendMessages, PermissionFlagsBits.ReadMessageHistory, PermissionFlagsBits.ManageChannels, PermissionFlagsBits.ManageMessages] }
  ];
  if (supportRoleId) permissions.push({ id: supportRoleId, allow: [PermissionFlagsBits.ViewChannel, PermissionFlagsBits.SendMessages, PermissionFlagsBits.ReadMessageHistory] });
  return permissions;
}

function isSupport(member, guildId) {
  const supportRoleId = database.getConfig(guildId, 'ticket_support_role_id');
  return member.permissions.has(PermissionFlagsBits.ManageChannels) || Boolean(supportRoleId && member.roles.cache.has(supportRoleId));
}

async function ticketLogChannel(guild) {
  const channelId = database.getSettings(guild.id).ticket_log_channel_id;
  return channelId ? guild.channels.fetch(channelId).catch(() => null) : null;
}

async function sendTicketLog(guild, payload) {
  const channel = await ticketLogChannel(guild);
  if (channel?.isTextBased()) await channel.send(payload).catch(() => null);
}

async function createTranscript(channel) {
  const messages = await channel.messages.fetch({ limit: 100 }).catch(() => null);
  if (!messages) return null;
  const lines = [...messages.values()].reverse().map((message) => {
    const date = new Date(message.createdTimestamp).toISOString();
    const attachments = [...message.attachments.values()].map((item) => item.url).join(' ');
    return `[${date}] ${message.author.tag} (${message.author.id}): ${message.content || '[embed/boş mesaj]'} ${attachments}`.trim();
  });
  return new AttachmentBuilder(Buffer.from(lines.join('\n'), 'utf8'), { name: `ticket-${channel.id}.txt` });
}

async function openTicket(interaction) {
  const guild = interaction.guild;
  const settings = database.getSettings(guild.id);
  const config = ticketConfig(guild.id);
  const existing = database.getOpenTicket(guild.id, interaction.user.id);
  if (existing) return interaction.reply({ content: `zaten açık bir talebin var: <#${existing.channel_id}>`, flags: 64 });
  if (!guild.members.me.permissions.has(PermissionFlagsBits.ManageChannels)) return interaction.reply({ content: 'botta `Kanalları Yönet` yetkisi yok.', flags: 64 });
  const category = settings.ticket_category_id ? await guild.channels.fetch(settings.ticket_category_id).catch(() => null) : null;
  if (settings.ticket_category_id && category?.type !== ChannelType.GuildCategory && category?.type !== 4) {
    const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('❌ Geçersiz Ticket Kategorisi')
      .setDescription('Sunucuda ayarlı olan Ticket Kategorisi bir metin kanalı olarak kaydedilmiş. Yetkililer `/ticketsetup` veya `/kurulum` ile bir **Discord Kategori Başlığı (📁 GuildCategory)** seçmelidir.');
    return interaction.reply({ embeds: [embed], flags: 64 });
  }
  const details = {
    type: 'destek',
    priority: interaction.fields.getTextInputValue('priority').toLocaleLowerCase('tr-TR'),
    subject: interaction.fields.getTextInputValue('subject'),
    description: interaction.fields.getTextInputValue('description')
  };
  if (!['düşük', 'dusuk', 'normal', 'yüksek', 'yuksek', 'acil'].includes(details.priority)) {
    return interaction.reply({ content: 'öncelik yalnızca düşük, normal, yüksek veya acil olabilir.', flags: 64 });
  }
  const priority = details.priority === 'dusuk' ? 'düşük' : details.priority === 'yuksek' ? 'yüksek' : details.priority;
  const safeName = `${priority}-${interaction.user.username}`.toLocaleLowerCase('tr-TR').replace(/[^a-z0-9çğıöşü-]/g, '-').slice(0, 70) || interaction.user.id;
  const ticket = await guild.channels.create({
    name: `talep-${safeName}`,
    type: ChannelType.GuildText,
    parent: category?.id,
    permissionOverwrites: ticketPermissions(guild, interaction.user.id, config.supportRoleId),
    topic: `Ticket sahibi: ${interaction.user.tag} (${interaction.user.id}) · ${details.subject}`,
    reason: `Ticket: ${interaction.user.tag}`
  });
  database.createTicket(ticket.id, guild.id, interaction.user.id, category?.id, { ...details, priority });
  await sendTicketLog(guild, { content: `🎫 Ticket açıldı: ${ticket} · sahibi: <@${interaction.user.id}> · öncelik: **${priority}** · konu: **${truncate(details.subject, 100)}**` });
  const welcome = config.welcomeMessage.replaceAll('{user}', `<@${interaction.user.id}>`).replaceAll('{username}', interaction.user.username);
  const embed = new EmbedBuilder().setColor(priority === 'acil' ? COLORS.danger : priority === 'yüksek' ? COLORS.warning : COLORS.success)
    .setTitle(`🎫 ${details.subject}`).setDescription(`${truncate(welcome, 2500)}\n\n**Açıklama**\n${truncate(details.description, 1200)}`)
    .addFields(
      { name: 'Talep sahibi', value: `<@${interaction.user.id}>`, inline: true },
      { name: 'Öncelik', value: priority, inline: true },
      { name: 'Durum', value: '🟢 Açık', inline: true }
    ).setTimestamp();
  const row1 = new ActionRowBuilder().addComponents(
    new ButtonBuilder().setCustomId('ticket_claim').setLabel('Talebi Üstlen').setEmoji('🙋').setStyle(ButtonStyle.Success),
    new ButtonBuilder().setCustomId('ticket_add_btn').setLabel('Üye Ekle').setEmoji('➕').setStyle(ButtonStyle.Primary),
    new ButtonBuilder().setCustomId('ticket_rename_btn').setLabel('Adlandır').setEmoji('✏️').setStyle(ButtonStyle.Secondary)
  );
  const row2 = new ActionRowBuilder().addComponents(
    new ButtonBuilder().setCustomId('ticket_status').setLabel('Durum Değiştir').setEmoji('📌').setStyle(ButtonStyle.Primary),
    new ButtonBuilder().setCustomId('ticket_close_btn').setLabel('Talebi Kapat').setEmoji('🔒').setStyle(ButtonStyle.Danger)
  );
  await ticket.send({
    content: config.supportRoleId ? `<@&${config.supportRoleId}>` : undefined,
    embeds: [embed],
    components: [row1, row2],
    allowedMentions: { roles: config.supportRoleId ? [config.supportRoleId] : [] }
  });
  await interaction.reply({ content: `talebin oluşturuldu: ${ticket}`, flags: 64 });
}

async function claimTicket(interaction) {
  const ticket = database.getTicket(interaction.channelId);
  if (!ticket || ticket.closed_at) return interaction.followUp({ content: 'bu kanal açık bir ticket değil.', flags: 64 });
  if (!isSupport(interaction.member, interaction.guildId)) return interaction.followUp({ content: 'ticket destek rolüne veya `Kanalları Yönet` yetkisine sahip değilsin.', flags: 64 });
  if (!database.claimTicket(interaction.channelId, interaction.user.id)) return interaction.followUp({ content: ticket.claimed_by_id ? `talep zaten <@${ticket.claimed_by_id}> tarafından üstlenildi.` : 'talep üstlenilemedi.', flags: 64 });
  await interaction.channel.send(`🙋 Bu talebi <@${interaction.user.id}> üstlendi.`);
  await sendTicketLog(interaction.guild, { content: `🙋 Ticket üstlenildi: <#${ticket.channel_id}> · yetkili: <@${interaction.user.id}>` });
}

async function closeTicket(interaction, customReason) {
  const ticket = database.getTicket(interaction.channelId);
  const reason = customReason || (interaction.fields ? interaction.fields.getTextInputValue('reason_input') : 'Sebep belirtilmedi');
  const respond = interaction.replied || interaction.deferred
    ? (payload) => interaction.followUp(payload)
    : (payload) => interaction.reply(payload);

  if (!ticket || ticket.closed_at) return respond({ content: 'bu kanal açık bir ticket değil.', flags: 64 });
  if (ticket.owner_id !== interaction.user.id && !isSupport(interaction.member, interaction.guildId)) return respond({ content: 'bu talebi kapatma yetkin yok.', flags: 64 });
  if (!database.closeTicket(interaction.channelId, interaction.user.id, reason)) return respond({ content: 'ticket zaten kapatılmış.', flags: 64 });

  const transcript = await createTranscript(interaction.channel);
  await sendModLog(interaction.guild, { action: 'ticket', targetId: ticket.owner_id, moderatorId: interaction.user.id, reason, color: COLORS.neutral });
  await sendTicketLog(interaction.guild, {
    content: `🔒 Ticket kapatıldı: **${interaction.channel.name}** · sahibi: <@${ticket.owner_id}> · kapatan: <@${interaction.user.id}> · sebep: **${truncate(reason, 200)}**`,
    files: transcript ? [transcript] : []
  });

  const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('🔒 Ticket Kapatıldı')
    .setDescription(`Bu destek talebi **<@${interaction.user.id}>** tarafından kapatıldı.\n\n**Kapanış Sebebi:**\n${truncate(reason, 500)}\n\n*Transcript log kanalına gönderildi. Kanal 5 saniye içinde silinecektir.*`)
    .setTimestamp();

  await interaction.channel.send({ embeds: [embed] }).catch(() => null);
  setTimeout(() => interaction.channel.delete(`Ticket kapatıldı: ${interaction.user.tag}`).catch(() => null), 5000).unref();
}

module.exports = {
  buildTicketPanel,
  ticketOpenModal,
  ticketAddModal,
  ticketRenameModal,
  ticketCloseModal,
  openTicket,
  claimTicket,
  closeTicket,
  isSupport,
  ticketConfig
};
