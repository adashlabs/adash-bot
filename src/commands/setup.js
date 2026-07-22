const {
  EmbedBuilder, ActionRowBuilder, StringSelectMenuBuilder, ChannelSelectMenuBuilder,
  RoleSelectMenuBuilder, ButtonBuilder, ButtonStyle, ChannelType, PermissionFlagsBits
} = require('discord.js');
const database = require('../database');
const { DEFAULT_SYSTEM_PROMPT } = require('../ai');
const { COLORS, truncate } = require('../utils/ui');

const onOff = (value) => value ? '✅ Açık' : '⛔ Kapalı';
const channel = (id) => id ? `<#${id}>` : '`Ayarlanmadı`';
const role = (id) => id ? `<@&${id}>` : '`Ayarlanmadı`';

function sectionMenu(guildId, selected) {
  return new ActionRowBuilder().addComponents(new StringSelectMenuBuilder()
    .setCustomId(`setup_section:${guildId}`)
    .setPlaceholder('Kurulum bölümü seç')
    .addOptions(
      { label: 'Genel Bakış', value: 'overview', emoji: '📋', default: selected === 'overview' },
      { label: 'Karşılama ve Rol', value: 'welcome', emoji: '👋', default: selected === 'welcome' },
      { label: 'Oyun Kanalları', value: 'games', emoji: '🎮', default: selected === 'games' },
      { label: 'Ticket Sistemi', value: 'tickets', emoji: '🎫', default: selected === 'tickets' },
      { label: 'Çekiliş Sistemi', value: 'giveaways', emoji: '🎉', default: selected === 'giveaways' },
      { label: 'Yapay Zekâ', value: 'ai', emoji: '🤖', default: selected === 'ai' },
      { label: 'Moderasyon Logu', value: 'modlog', emoji: '🛡️', default: selected === 'modlog' }
    ));
}

function buildSetupPanel(guild, section = 'overview') {
  const settings = database.getSettings(guild.id);
  const prefix = database.getPrefix(guild.id);
  const embed = new EmbedBuilder().setColor(COLORS.primary).setFooter({ text: 'Bu paneli yalnızca Sunucuyu Yönet yetkisi olanlar kullanabilir.' });
  const rows = [sectionMenu(guild.id, section)];

  if (section === 'welcome') {
    embed.setTitle('👋 Karşılama, Uğurlama ve Otomatik Rol').setDescription([
      `**Hoş geldin:** ${onOff(settings.welcome_enabled)} · ${channel(settings.welcome_channel_id)}`,
      `**Görüşürüz:** ${onOff(settings.farewell_enabled)} · ${channel(settings.farewell_channel_id)}`,
      `**Otomatik rol:** ${role(settings.autorole_id)}`,
      '',
      'Kanalları ve rolü menülerden seç. Mesajlarda `{user}`, `{username}`, `{server}`, `{memberCount}` kullanabilirsin.'
    ].join('\n'));
    rows.push(
      new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder().setCustomId(`setup_channel:welcome:${guild.id}`).setPlaceholder('Hoş geldin kanalı').setChannelTypes(ChannelType.GuildText)),
      new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder().setCustomId(`setup_channel:farewell:${guild.id}`).setPlaceholder('Görüşürüz kanalı').setChannelTypes(ChannelType.GuildText)),
      new ActionRowBuilder().addComponents(new RoleSelectMenuBuilder().setCustomId(`setup_role:autorole:${guild.id}`).setPlaceholder('Yeni üyelere verilecek rol')),
      new ActionRowBuilder().addComponents(
        new ButtonBuilder().setCustomId(`setup_toggle:welcome:${guild.id}`).setLabel('Hoş geldin Aç/Kapat').setStyle(ButtonStyle.Success),
        new ButtonBuilder().setCustomId(`setup_toggle:farewell:${guild.id}`).setLabel('Görüşürüz Aç/Kapat').setStyle(ButtonStyle.Secondary),
        new ButtonBuilder().setCustomId(`setup_edit_messages:${guild.id}`).setLabel('Mesajları Düzenle').setStyle(ButtonStyle.Primary)
      )
    );
  } else if (section === 'games') {
    embed.setTitle('🎮 Oyun Kanalları').setDescription([
      `**Sayı saymaca:** ${onOff(settings.counting_enabled)} · ${channel(settings.counting_channel_id)}`,
      `**Kelime türetmece:** ${onOff(settings.word_chain_enabled)} · ${channel(settings.word_chain_channel_id)}`,
      'Sayı oyununda sıradaki sayı yazılır; aynı kişi art arda oynayamaz. Kelimeler ağ gerektirmeyen yerel Türkçe sözlükten doğrulanır.'
    ].join('\n'));
    rows.push(
      new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder().setCustomId(`setup_channel:counting:${guild.id}`).setPlaceholder('Sayı saymaca kanalı').setChannelTypes(ChannelType.GuildText)),
      new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder().setCustomId(`setup_channel:word:${guild.id}`).setPlaceholder('Kelime türetmece kanalı').setChannelTypes(ChannelType.GuildText)),
      new ActionRowBuilder().addComponents(
        new ButtonBuilder().setCustomId(`setup_toggle:counting:${guild.id}`).setLabel('Sayı Oyunu Aç/Kapat').setStyle(ButtonStyle.Success),
        new ButtonBuilder().setCustomId(`setup_toggle:word:${guild.id}`).setLabel('Kelime Oyunu Aç/Kapat').setStyle(ButtonStyle.Success),
        new ButtonBuilder().setCustomId(`setup_reset:counting:${guild.id}`).setLabel('Sayıyı Sıfırla').setStyle(ButtonStyle.Danger),
        new ButtonBuilder().setCustomId(`setup_reset:word:${guild.id}`).setLabel('Kelimeleri Sıfırla').setStyle(ButtonStyle.Danger)
      )
    );
    rows.push(new ActionRowBuilder().addComponents(new ButtonBuilder()
      .setCustomId(`setup_game_rules:${guild.id}`).setLabel('Oyun Kurallarını Düzenle').setStyle(ButtonStyle.Primary)));
  } else if (section === 'tickets') {
    const supportRoleId = database.getConfig(guild.id, 'ticket_support_role_id');
    embed.setTitle('🎫 Ticket Sistemi').setDescription([
      `📁 **Ticket kategorisi:** ${channel(settings.ticket_category_id)}`,
      `📌 **Ticket panel kanalı:** ${channel(database.getConfig(guild.id, 'ticket_panel_channel_id'))}`,
      `📜 **Ticket log kanalı:** ${channel(settings.ticket_log_channel_id)}`,
      `🛡️ **Destek rolü:** ${role(supportRoleId)}`,
      `📝 **Panel başlığı:** ${database.getConfig(guild.id, 'ticket_panel_title', '🎫 Destek Talebi')}`,
      `🏷️ **Düğme yazısı:** ${database.getConfig(guild.id, 'ticket_panel_button_label', 'Destek Talebi Aç')}`,
      `😀 **Düğme emojisi:** ${database.getConfig(guild.id, 'ticket_panel_button_emoji', '🎫')}`,
      '',
      'Menülerden hedefleri seç veya `a!ticketsetup` / `/ticketsetup` komutuyla tek adımda tamamla.'
    ].join('\n'));
    rows.push(
      new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder()
        .setCustomId(`setup_channel:ticketcategory:${guild.id}`).setPlaceholder('1. Ticket kategorisi (📁 Kategori)').setChannelTypes(ChannelType.GuildCategory)),
      new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder()
        .setCustomId(`setup_channel:ticketlog:${guild.id}`).setPlaceholder('3. Ticket log kanalı (📜 Metin Kanalı)').setChannelTypes(ChannelType.GuildText)),
      new ActionRowBuilder().addComponents(new RoleSelectMenuBuilder()
        .setCustomId(`setup_role:ticketsupport:${guild.id}`).setPlaceholder('4. Ticket destek rolü (🛡️ Rol)')),
      new ActionRowBuilder().addComponents(
        new ButtonBuilder().setCustomId(`setup_ticket_texts:${guild.id}`).setLabel('Panel ve Karşılama Metinleri').setStyle(ButtonStyle.Primary),
        new ButtonBuilder().setCustomId(`ticketsetup_deploy:${guild.id}`).setLabel('🚀 Paneli Kanala Gönder').setStyle(ButtonStyle.Success),
        new ButtonBuilder().setCustomId(`setup_clear:ticketsupport:${guild.id}`).setLabel('Destek Rolünü Temizle').setStyle(ButtonStyle.Secondary)
      )
    );
  } else if (section === 'giveaways') {
    const requiredRoleId = database.getConfig(guild.id, 'giveaway_required_role_id');
    embed.setTitle('🎉 Çekiliş Sistemi').setDescription([
      `**Log kanalı:** ${channel(database.getConfig(guild.id, 'giveaway_log_channel_id'))}`,
      `**Gerekli rol:** ${role(requiredRoleId)}`,
      `**Minimum hesap yaşı:** ${database.getConfig(guild.id, 'giveaway_min_account_days', 0)} gün`,
      '',
      'Bu şartlar yeni oluşturulan çekilişlere kaydedilir. Mevcut çekilişler sonradan değişmez.'
    ].join('\n'));
    rows.push(
      new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder()
        .setCustomId(`setup_channel:giveawaylog:${guild.id}`).setPlaceholder('Çekiliş log kanalı').setChannelTypes(ChannelType.GuildText)),
      new ActionRowBuilder().addComponents(new RoleSelectMenuBuilder()
        .setCustomId(`setup_role:giveawayrole:${guild.id}`).setPlaceholder('Çekilişe katılım rolü')),
      new ActionRowBuilder().addComponents(
        new ButtonBuilder().setCustomId(`setup_giveaway_rules:${guild.id}`).setLabel('Katılım Kurallarını Düzenle').setStyle(ButtonStyle.Primary),
        new ButtonBuilder().setCustomId(`setup_giveaway_create_btn:${guild.id}`).setLabel('🎉 Çekiliş Başlat').setStyle(ButtonStyle.Success),
        new ButtonBuilder().setCustomId(`setup_clear:giveawayrole:${guild.id}`).setLabel('Rol Şartını Temizle').setStyle(ButtonStyle.Secondary)
      )
    );
  } else if (section === 'ai') {
    const aiEnabled = database.getConfig(guild.id, 'ai_enabled', true);
    embed.setTitle('🤖 Yapay Zekâ Asistanı').setDescription([
      `**Sunucuda durum:** ${onOff(aiEnabled)}`,
      `**Özel sohbet kanalı:** ${channel(settings.ai_channel_id)}`,
      `**Servis yapılandırması:** ${process.env.OPENAI_BASE_URL && process.env.OPENAI_API_KEY && process.env.OPENAI_MODEL ? '✅ Hazır' : '⛔ Eksik'}`,
      `**Model:** \`${process.env.OPENAI_MODEL || 'Ayarlanmadı'}\``,
      `**Sistem promptu:** ${truncate(database.getConfig(guild.id, 'ai_system_prompt', DEFAULT_SYSTEM_PROMPT), 180)}`,
      '',
      'Özel AI kanalında etiket atmaya gerek kalmadan otomatik sohbet edilir. Kanal dışında botu etiketleyerek soru sorulabilir.'
    ].join('\n'));
    rows.push(
      new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder()
        .setCustomId(`setup_channel:aichannel:${guild.id}`).setPlaceholder('Özel Yapay Zekâ sohbet kanalı (🤖 Metin Kanalı)').setChannelTypes(ChannelType.GuildText)),
      new ActionRowBuilder().addComponents(
        new ButtonBuilder().setCustomId(`setup_ai_prompt:${guild.id}`).setLabel('Sistem Promptunu Düzenle').setStyle(ButtonStyle.Primary),
        new ButtonBuilder().setCustomId(`setup_toggle:ai:${guild.id}`).setLabel('Yapay Zekâyı Aç/Kapat').setStyle(aiEnabled ? ButtonStyle.Danger : ButtonStyle.Success),
        new ButtonBuilder().setCustomId(`setup_clear:aichannel:${guild.id}`).setLabel('Sohbet Kanalını Temizle').setStyle(ButtonStyle.Secondary)
      )
    );
  } else if (section === 'modlog') {
    embed.setTitle('🛡️ Moderasyon Kayıtları').setDescription([
      `**Log kanalı:** ${channel(settings.mod_log_channel_id)}`,
      '',
      'Ban, kick, timeout, uyarı ve mesaj temizleme işlemleri seçilen kanala embed olarak gönderilir.'
    ].join('\n'));
    rows.push(new ActionRowBuilder().addComponents(new ChannelSelectMenuBuilder()
      .setCustomId(`setup_channel:modlog:${guild.id}`).setPlaceholder('Moderasyon log kanalı').setChannelTypes(ChannelType.GuildText)));
  } else {
    embed.setTitle('⚙️ adash Kurulum Paneli').setDescription([
      `**Prefix:** \`${prefix}\``,
      `**Hoş geldin:** ${onOff(settings.welcome_enabled)} · ${channel(settings.welcome_channel_id)}`,
      `**Görüşürüz:** ${onOff(settings.farewell_enabled)} · ${channel(settings.farewell_channel_id)}`,
      `**Otomatik rol:** ${role(settings.autorole_id)}`,
      `**Sayı saymaca:** ${onOff(settings.counting_enabled)} · ${channel(settings.counting_channel_id)}`,
      `**Kelime türetmece:** ${onOff(settings.word_chain_enabled)} · ${channel(settings.word_chain_channel_id)}`,
      `**Ticket kategorisi:** ${channel(settings.ticket_category_id)}`,
      `**Çekiliş logu:** ${channel(database.getConfig(guild.id, 'giveaway_log_channel_id'))}`,
      `**Yapay zekâ:** ${onOff(database.getConfig(guild.id, 'ai_enabled', true))}`,
      `**Mod log:** ${channel(settings.mod_log_channel_id)}`,
      '',
      'Yukarıdaki menüden bir bölüm seçerek kurulumu tamamla.'
    ].join('\n'));
  }
  return { embeds: [embed], components: rows };
}

module.exports = {
  name: 'setup',
  aliases: ['kurulum', 'ayarlar'],
  category: 'ayarlar',
  description: 'buton, menü ve modal kullanan sunucu kurulum panelini açar.',
  userPermissions: [PermissionFlagsBits.ManageGuild],
  buildSetupPanel,
  async execute(message) {
    if (!message.member.permissions.has(PermissionFlagsBits.ManageGuild)) {
      return message.reply('bu panel için `Sunucuyu Yönet` yetkisi gerekiyor.');
    }
    const permissions = message.channel.permissionsFor(message.guild.members.me);
    if (!permissions?.has([PermissionFlagsBits.ViewChannel, PermissionFlagsBits.SendMessages, PermissionFlagsBits.EmbedLinks])) {
      const notice = 'Bu kanalda kurulum paneli gönderemiyorum. Bana `Kanalları Görüntüle`, `Mesaj Gönder` ve `Bağlantıları Göm` yetkilerini ver.';
      console.warn(`[UYARI] setup ${message.guild.id}/${message.channel.id}: bot kanal yetkileri eksik.`);
      return message.author.send(notice).catch(() => null);
    }
    await message.reply(buildSetupPanel(message.guild));
  }
};
