const {
  ModalBuilder, ActionRowBuilder, TextInputBuilder, TextInputStyle, PermissionFlagsBits
} = require('discord.js');
const { canManageRole } = require('../utils/security');
const database = require('../database');
const ping = require('../commands/ping');
const help = require('../commands/help');
const setup = require('../commands/setup');
const { buildTicketSetupWizard } = require('../commands/ticketsetup');
const { buildTicketPanel } = require('../tickets');
const wsearch = require('../commands/wsearch');
const { toggleEntry, buildGiveawayEmbed, buttonRow, scheduleGiveaway } = require('../giveaways');
const { parseDuration } = require('../utils/resolvers');
const { truncate, errorEmbed } = require('../utils/ui');
const { ticketOpenModal, ticketAddModal, ticketRenameModal, ticketCloseModal, openTicket, claimTicket, closeTicket } = require('../tickets');
const { runSlash } = require('../slash');
const { handleConfirmation } = require('../utils/confirmations');
const { DEFAULT_SYSTEM_PROMPT } = require('../ai');

const EPHEMERAL = 64;
const ownerFromId = (id) => id.split(':').at(-1);

async function reject(interaction, titleOrContent, description) {
  const embed = description
    ? errorEmbed(titleOrContent, description, interaction.user)
    : errorEmbed('İşlem Başarısız', titleOrContent, interaction.user);
  if (interaction.deferred || interaction.replied) return interaction.followUp({ embeds: [embed], flags: EPHEMERAL });
  return interaction.reply({ embeds: [embed], flags: EPHEMERAL });
}

function hasSetupAccess(interaction) {
  if (!interaction.guild || !interaction.member) return false;
  if (interaction.user.id === interaction.guild.ownerId) return true;
  const hasPerm = interaction.member.permissions.has(PermissionFlagsBits.ManageGuild) || interaction.member.permissions.has(PermissionFlagsBits.Administrator);
  if (!hasPerm) return false;
  const parts = interaction.customId.split(':');
  const targetGuildId = parts.length > 1 ? parts.at(-1) : null;
  if (targetGuildId && /^\d{17,20}$/.test(targetGuildId) && targetGuildId !== interaction.guild.id) {
    return false;
  }
  return true;
}

async function editSetup(interaction, section) {
  await interaction.editReply(setup.buildSetupPanel(interaction.guild, section));
}

async function handleSetup(interaction) {
  if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
  const [, subject] = interaction.customId.split(':');
  if (interaction.customId.startsWith('setup_edit_messages:')) {
    const settings = database.getSettings(interaction.guild.id);
    const modal = new ModalBuilder().setCustomId(`setup_messages:${interaction.guild.id}`).setTitle('Karşılama Mesajları');
    modal.addComponents(
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('welcome_message').setLabel('Hoş geldin mesajı').setStyle(TextInputStyle.Paragraph).setRequired(true).setMaxLength(1000).setValue(settings.welcome_message)),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('farewell_message').setLabel('Görüşürüz mesajı').setStyle(TextInputStyle.Paragraph).setRequired(true).setMaxLength(1000).setValue(settings.farewell_message))
    );
    return interaction.showModal(modal);
  }
  if (interaction.customId.startsWith('setup_ticket_texts:') || (interaction.isButton() && interaction.customId.startsWith('setup_ticket_modal:'))) {
    const modal = new ModalBuilder().setCustomId(`setup_ticket_modal:${interaction.guild.id}`).setTitle('Ticket Metinleri & Düğme Ayarları');
    modal.addComponents(
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('ticket_panel_title').setLabel('Panel başlığı').setStyle(TextInputStyle.Short).setRequired(true).setMaxLength(256).setValue(database.getConfig(interaction.guild.id, 'ticket_panel_title', '🎫 Destek Talebi'))),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('ticket_panel_description').setLabel('Panel açıklaması').setStyle(TextInputStyle.Paragraph).setRequired(true).setMaxLength(1500).setValue(database.getConfig(interaction.guild.id, 'ticket_panel_description', 'Destek ekibimize ulaşmak için aşağıdaki düğmeye bas.'))),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('ticket_welcome_message').setLabel('Ticket karşılama mesajı').setStyle(TextInputStyle.Paragraph).setRequired(true).setMaxLength(1500).setValue(database.getConfig(interaction.guild.id, 'ticket_welcome_message', '{user}, talebin oluşturuldu. Sorununu ayrıntılı biçimde anlat.'))),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('ticket_panel_button_label').setLabel('Düğme yazısı').setStyle(TextInputStyle.Short).setRequired(true).setMaxLength(80).setValue(database.getConfig(interaction.guild.id, 'ticket_panel_button_label', 'Destek Talebi Aç'))),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('ticket_panel_button_emoji').setLabel('Düğme emojisi (Örn: 🎫, 📩, 💬)').setStyle(TextInputStyle.Short).setRequired(true).setMaxLength(10).setValue(database.getConfig(interaction.guild.id, 'ticket_panel_button_emoji', '🎫')))
    );
    return interaction.showModal(modal);
  }
  if (interaction.customId.startsWith('setup_ai_prompt:')) {
    const prompt = database.getConfig(interaction.guild.id, 'ai_system_prompt', DEFAULT_SYSTEM_PROMPT);
    const modal = new ModalBuilder().setCustomId(`setup_ai_prompt_modal:${interaction.guild.id}`).setTitle('Yapay Zekâ Sistem Promptu');
    modal.addComponents(new ActionRowBuilder().addComponents(new TextInputBuilder()
      .setCustomId('ai_system_prompt').setLabel('Yapay zekânın karakteri ve konuşma tarzı')
      .setStyle(TextInputStyle.Paragraph).setRequired(true).setMinLength(20).setMaxLength(4000)
      .setValue(truncate(prompt, 4000))));
    return interaction.showModal(modal);
  }
  if (interaction.customId.startsWith('setup_giveaway_rules:')) {
    const modal = new ModalBuilder().setCustomId(`setup_giveaway_modal:${interaction.guild.id}`).setTitle('Çekiliş Katılım Kuralları');
    modal.addComponents(new ActionRowBuilder().addComponents(new TextInputBuilder()
      .setCustomId('giveaway_min_account_days').setLabel('Minimum hesap yaşı (0-365 gün)').setStyle(TextInputStyle.Short)
      .setRequired(true).setMinLength(1).setMaxLength(3).setValue(String(database.getConfig(interaction.guild.id, 'giveaway_min_account_days', 0)))));
    return interaction.showModal(modal);
  }
  if (interaction.customId.startsWith('setup_game_rules:')) {
    const modal = new ModalBuilder().setCustomId(`setup_game_modal:${interaction.guild.id}`).setTitle('Oyun Kuralları');
    modal.addComponents(
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('word_min_length').setLabel('Minimum kelime uzunluğu (2-10)').setStyle(TextInputStyle.Short).setRequired(true).setValue(String(database.getConfig(interaction.guild.id, 'word_min_length', 2)))),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('counting_reset_on_error').setLabel('Yanlış sayıda sıfırla? (evet/hayır)').setStyle(TextInputStyle.Short).setRequired(true).setValue(database.getConfig(interaction.guild.id, 'counting_reset_on_error', true) ? 'evet' : 'hayır')),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('game_delete_invalid').setLabel('Geçersiz mesajı sil? (evet/hayır)').setStyle(TextInputStyle.Short).setRequired(true).setValue(database.getConfig(interaction.guild.id, 'game_delete_invalid', true) ? 'evet' : 'hayır'))
    );
    return interaction.showModal(modal);
  }
  if (interaction.customId.startsWith('setup_giveaway_create_btn:')) {
    const modal = new ModalBuilder().setCustomId('setup_giveaway_create_modal').setTitle('🎉 Çekiliş Oluştur');
    modal.addComponents(
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('duration').setLabel('Süre (Örn: 10m, 1h, 2d)').setStyle(TextInputStyle.Short).setRequired(true).setValue('1h').setMaxLength(20)),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('winners').setLabel('Kazanan Sayısı (1-20)').setStyle(TextInputStyle.Short).setRequired(true).setValue('1').setMaxLength(2)),
      new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('prize').setLabel('Çekiliş Ödülü').setStyle(TextInputStyle.Paragraph).setRequired(true).setMaxLength(500))
    );
    return interaction.showModal(modal);
  }

  await interaction.deferUpdate();
  if (interaction.customId.startsWith('setup_section:')) return editSetup(interaction, interaction.values[0]);
  if (interaction.customId.startsWith('setup_channel:')) {
    const keys = { welcome: 'welcome_channel_id', farewell: 'farewell_channel_id', counting: 'counting_channel_id', word: 'word_chain_channel_id', modlog: 'mod_log_channel_id', ticketcategory: 'ticket_category_id', ticketlog: 'ticket_log_channel_id', giveawaylog: 'giveaway_log_channel_id', ticketpanel: 'ticket_panel_channel_id', aichannel: 'ai_channel_id' };
    const selected = await interaction.guild.channels.fetch(interaction.values[0]).catch(() => null);
    if (!selected) return reject(interaction, 'seçilen kanal artık mevcut değil.');
    const permissions = selected.permissionsFor(interaction.guild.members.me);
    const required = [PermissionFlagsBits.ViewChannel];
    if (subject === 'ticketcategory') {
      if (selected.type !== 4 && selected.type !== ChannelType.GuildCategory) {
        return reject(interaction, 'seçilen kanal bir metin kanalı değil, Discord Kategori başlığı (GuildCategory) olmalıdır.');
      }
      required.push(PermissionFlagsBits.ManageChannels);
    }
    else required.push(PermissionFlagsBits.SendMessages);
    if (['welcome', 'farewell', 'modlog', 'ticketlog', 'giveawaylog'].includes(subject)) required.push(PermissionFlagsBits.EmbedLinks);
    if (['welcome', 'farewell'].includes(subject)) required.push(PermissionFlagsBits.AttachFiles);
    if (subject === 'ticketlog') required.push(PermissionFlagsBits.AttachFiles);
    if (['counting', 'word'].includes(subject)) required.push(PermissionFlagsBits.ManageMessages, PermissionFlagsBits.AddReactions, PermissionFlagsBits.ReadMessageHistory);
    if (!permissions?.has(required)) return reject(interaction, 'botun seçilen kanalda bu özellik için gerekli yetkileri eksik.');
    const section = ['welcome', 'farewell'].includes(subject) ? 'welcome' : (['counting', 'word'].includes(subject) ? 'games' : (subject.startsWith('ticket') ? 'tickets' : (subject.startsWith('giveaway') ? 'giveaways' : (subject === 'aichannel' ? 'ai' : 'modlog'))));
    if (['giveawaylog', 'ticketpanel'].includes(subject)) database.setConfig(interaction.guild.id, keys[subject], selected.id);
    else database.setSetting(interaction.guild.id, keys[subject], selected.id);
    return editSetup(interaction, section);
  }
  if (interaction.customId.startsWith('setup_clear:')) {
    if (subject === 'ticketsupport') {
      database.setConfig(interaction.guild.id, 'ticket_support_role_id', null);
      return editSetup(interaction, 'tickets');
    }
    if (subject === 'giveawayrole') {
      database.setConfig(interaction.guild.id, 'giveaway_required_role_id', null);
      return editSetup(interaction, 'giveaways');
    }
    if (subject === 'aichannel') {
      database.setSetting(interaction.guild.id, 'ai_channel_id', null);
      return editSetup(interaction, 'ai');
    }
  }
  if (interaction.customId.startsWith('setup_role:')) {
    const role = await interaction.guild.roles.fetch(interaction.values[0]).catch(() => null);
    if (!role || role.id === interaction.guild.id) return reject(interaction, 'geçerli bir rol seçmelisin.');
    if (subject === 'autorole') {
      const check = canManageRole(interaction.guild, role);
      if (!check.ok) return reject(interaction, check.reason);
    }
    if (subject === 'ticketsupport') {
      database.setConfig(interaction.guild.id, 'ticket_support_role_id', role.id);
      return editSetup(interaction, 'tickets');
    }
    if (subject === 'giveawayrole') {
      database.setConfig(interaction.guild.id, 'giveaway_required_role_id', role.id);
      return editSetup(interaction, 'giveaways');
    }
    database.setSetting(interaction.guild.id, 'autorole_id', role.id);
    return editSetup(interaction, 'welcome');
  }
  if (interaction.customId.startsWith('setup_toggle:')) {
    if (subject === 'ai') {
      database.setConfig(interaction.guild.id, 'ai_enabled', !database.getConfig(interaction.guild.id, 'ai_enabled', true));
      return editSetup(interaction, 'ai');
    }
    const keys = { welcome: ['welcome_enabled', 'welcome_channel_id', 'welcome'], farewell: ['farewell_enabled', 'farewell_channel_id', 'welcome'], counting: ['counting_enabled', 'counting_channel_id', 'games'], word: ['word_chain_enabled', 'word_chain_channel_id', 'games'] };
    const [enabledKey, channelKey, section] = keys[subject];
    const current = database.getSettings(interaction.guild.id);
    if (!current[enabledKey] && !current[channelKey]) return reject(interaction, 'önce bir kanal seçmelisin.');
    database.setSetting(interaction.guild.id, enabledKey, current[enabledKey] ? 0 : 1);
    return editSetup(interaction, section);
  }
  if (interaction.customId.startsWith('setup_reset:')) {
    database.resetGame(interaction.guild.id, subject);
    return editSetup(interaction, 'games');
  }
}

async function handleHelp(interaction) {
  const ownerId = ownerFromId(interaction.customId);
  if (ownerId !== '0' && ownerId !== interaction.user.id) return reject(interaction, 'bu yardım menüsü sana ait değil.');
  await interaction.deferUpdate();
  await interaction.editReply(help.buildHelp(interaction.client, database.getPrefix(interaction.guildId), interaction.values[0], ownerId));
}

async function handleSearch(interaction) {
  const [, action, sessionId] = interaction.customId.split(':');
  const session = wsearch.sessions.get(sessionId);
  if (!session) return reject(interaction, 'arama oturumunun süresi doldu.');
  if (session.userId !== interaction.user.id) return reject(interaction, 'bu arama sana ait değil.');
  const page = session.page + (action === 'next' ? 1 : -1);
  if (page < 1 || page > session.totalPages) return reject(interaction, 'geçersiz sayfa.');
  await interaction.deferUpdate();
  session.page = page;
  session.timestamp = Date.now();
  await interaction.editReply({ embeds: [wsearch.buildEmbed(session, page)], components: [wsearch.buildRow(sessionId, page, session.totalPages)] });
}


module.exports = {
  name: 'interactionCreate',
  async execute(interaction) {
    try {
      if (interaction.isChatInputCommand()) return runSlash(interaction);
      if (interaction.isModalSubmit() && interaction.customId.startsWith('setup_messages:')) {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        database.setSetting(interaction.guildId, 'welcome_message', interaction.fields.getTextInputValue('welcome_message'));
        database.setSetting(interaction.guildId, 'farewell_message', interaction.fields.getTextInputValue('farewell_message'));
        return interaction.reply({ content: 'karşılama mesajları kaydedildi.', flags: EPHEMERAL });
      }
      if (interaction.isModalSubmit() && interaction.customId.startsWith('setup_ai_prompt_modal:')) {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        const prompt = interaction.fields.getTextInputValue('ai_system_prompt').trim();
        if (prompt.length < 20) return reject(interaction, 'sistem promptu en az 20 karakter olmalıdır.');
        database.setConfig(interaction.guildId, 'ai_system_prompt', prompt);
        return interaction.reply({ content: 'yapay zekâ sistem promptu kaydedildi.', flags: EPHEMERAL });
      }
      if (interaction.isModalSubmit() && interaction.customId.startsWith('setup_ticket_modal:')) {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        database.setConfig(interaction.guildId, 'ticket_panel_title', interaction.fields.getTextInputValue('ticket_panel_title'));
        database.setConfig(interaction.guildId, 'ticket_panel_description', interaction.fields.getTextInputValue('ticket_panel_description'));
        database.setConfig(interaction.guildId, 'ticket_welcome_message', interaction.fields.getTextInputValue('ticket_welcome_message'));
        database.setConfig(interaction.guildId, 'ticket_panel_button_label', interaction.fields.getTextInputValue('ticket_panel_button_label'));
        database.setConfig(interaction.guildId, 'ticket_panel_button_emoji', interaction.fields.getTextInputValue('ticket_panel_button_emoji'));
        return interaction.reply({ content: 'ticket paneli, düğme ve karşılama metinleri başarıyla kaydedildi.', flags: EPHEMERAL });
      }
      if (interaction.isModalSubmit() && interaction.customId === 'ticket_open_modal') return openTicket(interaction);
      if (interaction.isStringSelectMenu() && interaction.customId.startsWith('help_select:')) return handleHelp(interaction);
      if ((interaction.isStringSelectMenu() || interaction.isChannelSelectMenu() || interaction.isRoleSelectMenu() || interaction.isButton()) && interaction.customId.startsWith('setup_')) return handleSetup(interaction);
      if (interaction.isChannelSelectMenu() && interaction.customId.startsWith('ticketsetup_panelchan:')) {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        const channelId = interaction.values[0];
        database.setConfig(interaction.guildId, 'ticket_panel_channel_id', channelId);
        await interaction.deferUpdate();
        return interaction.editReply(buildTicketSetupWizard(interaction.guild));
      }
      if (interaction.isButton() && interaction.customId.startsWith('ticketsetup_deploy:')) {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        const panelChannelId = database.getConfig(interaction.guildId, 'ticket_panel_channel_id');
        const channel = panelChannelId ? await interaction.guild.channels.fetch(panelChannelId).catch(() => null) : null;
        if (!channel?.isTextBased()) return reject(interaction, 'lütfen önce açılır menüden bir Ticket Panel Kanalı seçin.');
        const sent = await channel.send(buildTicketPanel(interaction.guild)).catch(() => null);
        if (!sent) return reject(interaction, `${channel} kanalına mesaj gönderme yetkisi yok.`);
        return interaction.reply({
          embeds: [
            new (require('discord.js').EmbedBuilder)().setColor(0x57F287).setTitle('🚀 Ticket Paneli Başarıyla Gönderildi')
              .setDescription(`Ticket açma paneli ${channel} kanalına başarıyla gönderildi: [Paneli Gör](${sent.url})`)
          ],
          flags: EPHEMERAL
        });
      }
      if (interaction.isButton() && interaction.customId === 'ping_refresh') {
        await interaction.deferUpdate();
        return interaction.editReply(ping.buildOutput(interaction.client, interaction.guildId));
      }
      if (interaction.isButton() && interaction.customId === 'coinflip_retry') {
        await interaction.deferUpdate();
        const isHeads = Math.random() < 0.5;
        const result = isHeads ? 'Yazı' : 'Tura';
        const embed = new (require('discord.js').EmbedBuilder)()
          .setColor(0x57F287)
          .setTitle('🪙 Yazı Tura Atıldı')
          .setDescription(`Para havaya fırlatıldı ve düştü:\n\n### **${result}!**`)
          .setFooter({ text: `Atan: ${interaction.user.tag}`, iconURL: interaction.user.displayAvatarURL() })
          .setTimestamp();
        return interaction.editReply({ embeds: [embed] });
      }
      if (interaction.isButton() && interaction.customId.startsWith('roll_retry:')) {
        await interaction.deferUpdate();
        const [, countStr, sidesStr] = interaction.customId.split(':');
        const count = Number(countStr) || 1;
        const sides = Number(sidesStr) || 6;
        const rolls = Array.from({ length: count }, () => Math.floor(Math.random() * sides) + 1);
        const total = rolls.reduce((sum, value) => sum + value, 0);
        const embed = new (require('discord.js').EmbedBuilder)()
          .setColor(0x3498DB)
          .setTitle(`🎲 Zar Atıldı: ${count}d${sides}`)
          .addFields(
            { name: 'Gelen Zarlar', value: rolls.map((value) => `\`${value}\``).join(' ') },
            ...(count > 1 ? [{ name: 'Toplam', value: `**${total}**`, inline: true }] : [])
          )
          .setFooter({ text: `Atan: ${interaction.user.tag}`, iconURL: interaction.user.displayAvatarURL() })
          .setTimestamp();
        return interaction.editReply({ embeds: [embed] });
      }
      if (interaction.isButton() && interaction.customId.startsWith('wsearch:')) return handleSearch(interaction);
      if (interaction.isButton() && interaction.customId.startsWith('setup_giveaway_create_btn:')) {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        const modal = new ModalBuilder().setCustomId('setup_giveaway_create_modal').setTitle('🎉 Çekiliş Oluştur');
        modal.addComponents(
          new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('duration').setLabel('Süre (Örn: 10m, 1h, 2d)').setStyle(TextInputStyle.Short).setRequired(true).setValue('1h').setMaxLength(20)),
          new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('winners').setLabel('Kazanan Sayısı (1-20)').setStyle(TextInputStyle.Short).setRequired(true).setValue('1').setMaxLength(2)),
          new ActionRowBuilder().addComponents(new TextInputBuilder().setCustomId('prize').setLabel('Çekiliş Ödülü').setStyle(TextInputStyle.Paragraph).setRequired(true).setMaxLength(500))
        );
        return interaction.showModal(modal);
      }
      if (interaction.isModalSubmit() && interaction.customId === 'setup_giveaway_create_modal') {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        const durationInput = interaction.fields.getTextInputValue('duration');
        const winnersInput = Number(interaction.fields.getTextInputValue('winners'));
        const prizeInput = truncate(interaction.fields.getTextInputValue('prize'), 500);
        const durationMs = parseDuration(durationInput);
        if (!durationMs || durationMs < 10000 || durationMs > 30 * 86400000) return reject(interaction, 'süre 10 saniye ile 30 gün arasında olmalı.');
        if (!Number.isInteger(winnersInput) || winnersInput < 1 || winnersInput > 20) return reject(interaction, 'kazanan sayısı 1 ile 20 arasında olmalı.');
        const requiredRoleId = database.getConfig(interaction.guildId, 'giveaway_required_role_id');
        const minAccountAgeDays = Number(database.getConfig(interaction.guildId, 'giveaway_min_account_days', 0)) || 0;
        const draft = {
          guild_id: interaction.guildId, channel_id: interaction.channelId, host_id: interaction.user.id,
          prize: prizeInput, winner_count: winnersInput, required_role_id: requiredRoleId,
          min_account_age_days: minAccountAgeDays, ends_at: Date.now() + durationMs
        };
        const sent = await interaction.channel.send({ embeds: [buildGiveawayEmbed(draft, 0)], components: buttonRow() });
        const id = database.createGiveaway(
          draft.guild_id, draft.channel_id, sent.id, draft.host_id, prizeInput, winnersInput,
          requiredRoleId, minAccountAgeDays, draft.ends_at
        );
        scheduleGiveaway(interaction.client, { ...draft, id, message_id: sent.id });
        return interaction.reply({ content: `✅ Çekiliş başarıyla başlatıldı ve gönderildi: ${sent.url}`, flags: EPHEMERAL });
      }
      if (interaction.isButton() && interaction.customId === 'ticket_open') return interaction.showModal(ticketOpenModal());
      if (interaction.isModalSubmit() && interaction.customId.startsWith('setup_giveaway_modal:')) {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        const days = Number(interaction.fields.getTextInputValue('giveaway_min_account_days'));
        if (!Number.isInteger(days) || days < 0 || days > 365) return reject(interaction, 'hesap yaşı 0 ile 365 gün arasında tam sayı olmalı.');
        database.setConfig(interaction.guildId, 'giveaway_min_account_days', days);
        return interaction.reply({ content: 'çekiliş katılım kuralları kaydedildi.', flags: EPHEMERAL });
      }
      if (interaction.isModalSubmit() && interaction.customId.startsWith('setup_game_modal:')) {
        if (!hasSetupAccess(interaction)) return reject(interaction, 'bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
        const minLength = Number(interaction.fields.getTextInputValue('word_min_length'));
        const reset = interaction.fields.getTextInputValue('counting_reset_on_error').toLocaleLowerCase('tr-TR');
        const remove = interaction.fields.getTextInputValue('game_delete_invalid').toLocaleLowerCase('tr-TR');
        if (!Number.isInteger(minLength) || minLength < 2 || minLength > 10 || !['evet', 'hayır', 'hayir'].includes(reset) || !['evet', 'hayır', 'hayir'].includes(remove)) {
          return reject(interaction, 'değerler geçersiz: uzunluk 2-10, seçenekler evet/hayır olmalı.');
        }
        database.setConfig(interaction.guildId, 'word_min_length', minLength);
        database.setConfig(interaction.guildId, 'counting_reset_on_error', reset === 'evet');
        database.setConfig(interaction.guildId, 'game_delete_invalid', remove === 'evet');
        return interaction.reply({ content: 'oyun kuralları kaydedildi.', flags: EPHEMERAL });
      }
      if (interaction.isButton() && interaction.customId === 'ticket_claim') { await interaction.deferUpdate(); return claimTicket(interaction); }
      if (interaction.isButton() && interaction.customId === 'ticket_status') {
        await interaction.deferUpdate();
        const ticket = database.getTicket(interaction.channelId);
        if (!ticket || ticket.closed_at) return interaction.followUp({ content: 'bu kanal açık bir ticket değil.', flags: EPHEMERAL });
        if (!interaction.member.permissions.has(PermissionFlagsBits.ManageChannels) && !interaction.member.roles.cache.has(database.getConfig(interaction.guildId, 'ticket_support_role_id'))) {
          return interaction.followUp({ content: 'ticket durumunu yalnızca destek ekibi değiştirebilir.', flags: EPHEMERAL });
        }
        const next = ticket.status === 'open' ? 'in_progress' : ticket.status === 'in_progress' ? 'waiting_user' : 'open';
        database.setTicketStatus(interaction.channelId, next);
        return interaction.channel.send(`📌 Ticket durumu: **${next === 'in_progress' ? 'İnceleniyor' : next === 'waiting_user' ? 'Kullanıcı yanıtı bekleniyor' : 'Açık'}**`);
      }
      if (interaction.isButton() && interaction.customId === 'ticket_close') { await interaction.deferUpdate(); return closeTicket(interaction); }
      if (interaction.isButton() && interaction.customId === 'ticket_add_btn') return interaction.showModal(ticketAddModal());
      if (interaction.isButton() && interaction.customId === 'ticket_rename_btn') return interaction.showModal(ticketRenameModal());
      if (interaction.isButton() && interaction.customId === 'ticket_close_btn') return interaction.showModal(ticketCloseModal());
      if (interaction.isModalSubmit() && interaction.customId === 'ticket_add_modal') {
        const input = interaction.fields.getTextInputValue('user_input').trim();
        const id = input.match(/\d{17,20}/)?.[0] || input;
        const member = await interaction.guild.members.fetch(id).catch(() => null);
        if (!member) return interaction.reply({ content: 'kullanıcı bulunamadı.', flags: EPHEMERAL });
        await interaction.channel.permissionOverwrites.edit(member, { ViewChannel: true, SendMessages: true, ReadMessageHistory: true });
        return interaction.reply({ content: `✅ <@${member.id}> bu talebe eklendi.` });
      }
      if (interaction.isModalSubmit() && interaction.customId === 'ticket_rename_modal') {
        const name = interaction.fields.getTextInputValue('name_input').toLocaleLowerCase('tr-TR').replace(/[^a-z0-9çğıöşü-]/g, '-').replace(/-+/g, '-').slice(0, 80);
        if (name.length < 2) return interaction.reply({ content: 'geçersiz kanal adı.', flags: EPHEMERAL });
        await interaction.channel.setName(name, `Ticket adı: ${interaction.user.tag}`);
        return interaction.reply({ content: `✏️ Kanal adı **${name}** olarak güncellendi.` });
      }
      if (interaction.isModalSubmit() && interaction.customId === 'ticket_close_modal') return closeTicket(interaction);
      if (interaction.isButton() && interaction.customId === 'giveaway_join') { await interaction.deferUpdate(); return toggleEntry(interaction); }
      if (interaction.isButton() && interaction.customId.startsWith('modconfirm:')) return handleConfirmation(interaction);
    } catch (error) {
      if (error?.code === 10062) return;
      console.error('[HATA] interactionCreate:', error);
      await reject(interaction, 'işlem tamamlanamadı. Bot yetkilerini ve kanal erişimini kontrol et.').catch(() => null);
    }
  }
};
