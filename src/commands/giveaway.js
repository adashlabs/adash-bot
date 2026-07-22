const {
  PermissionFlagsBits, ChannelType, EmbedBuilder, ActionRowBuilder,
  ChannelSelectMenuBuilder, RoleSelectMenuBuilder, ButtonBuilder, ButtonStyle
} = require('discord.js');
const { parseDuration } = require('../utils/resolvers');
const database = require('../database');
const { buildGiveawayEmbed, buttonRow, scheduleGiveaway } = require('../giveaways');
const { truncate, COLORS } = require('../utils/ui');

function buildGiveawayWizard(guild) {
  const requiredRoleId = database.getConfig(guild.id, 'giveaway_required_role_id');
  const logChannelId = database.getConfig(guild.id, 'giveaway_log_channel_id');
  const minAccountDays = database.getConfig(guild.id, 'giveaway_min_account_days', 0);

  const channelStr = (id) => id ? `<#${id}>` : '`Ayarlanmadı`';
  const roleStr = (id) => id ? `<@&${id}>` : '`Ayarlanmadı`';

  const embed = new EmbedBuilder()
    .setColor(COLORS.primary)
    .setTitle('🎉 Etkileşimli Çekiliş Kurulum & Başlatma Paneli')
    .setDescription([
      'Aşağıdaki açılır menüler ve butonları kullanarak çekiliş şartlarını ve yeni çekilişinizi kolayca yapılandırın.',
      '',
      `📜 **Çekiliş Log Kanalı:** ${channelStr(logChannelId)}`,
      `🛡️ **Gerekli Katılım Rolü:** ${roleStr(requiredRoleId)}`,
      `⌛ **Minimum Hesap Yaşı:** ${minAccountDays} gün`
    ].join('\n'))
    .setFooter({ text: 'Şartları ayarladıktan sonra "🎉 Çekiliş Başlat" butonuna basın.' })
    .setTimestamp();

  const rows = [
    new ActionRowBuilder().addComponents(
      new ChannelSelectMenuBuilder()
        .setCustomId(`setup_channel:giveawaylog:${guild.id}`)
        .setPlaceholder('1. Çekiliş Log Kanalını Seç (📜 Metin Kanalı)')
        .setChannelTypes(ChannelType.GuildText)
    ),
    new ActionRowBuilder().addComponents(
      new RoleSelectMenuBuilder()
        .setCustomId(`setup_role:giveawayrole:${guild.id}`)
        .setPlaceholder('2. Çekiliş İçin Gerekli Rolü Seç (🛡️ Rol)')
    ),
    new ActionRowBuilder().addComponents(
      new ButtonBuilder().setCustomId(`setup_giveaway_rules:${guild.id}`).setLabel('Katılım Kurallarını Düzenle').setEmoji('⚙️').setStyle(ButtonStyle.Primary),
      new ButtonBuilder().setCustomId(`setup_giveaway_create_btn:${guild.id}`).setLabel('Çekiliş Başlat').setEmoji('🎉').setStyle(ButtonStyle.Success),
      new ButtonBuilder().setCustomId(`setup_clear:giveawayrole:${guild.id}`).setLabel('Rol Şartını Temizle').setEmoji('🧹').setStyle(ButtonStyle.Secondary)
    )
  ];

  return { embeds: [embed], components: rows };
}
module.exports = {
  name: 'giveaway',
  aliases: ['çekiliş', 'cekilis'],
  category: 'ayarlar',
  description: 'rol ve hesap yaşı şartlarını kullanan kalıcı çekiliş oluşturur.',
  cooldown: 5,
  buildGiveawayWizard,
  async execute(message, args, client) {
    if (!message.member.permissions.has(PermissionFlagsBits.ManageGuild)) return message.reply('çekiliş için `Sunucuyu Yönet` yetkisi gerekiyor.');
    const permissions = message.channel.permissionsFor(message.guild.members.me);
    if (!permissions?.has([PermissionFlagsBits.ViewChannel, PermissionFlagsBits.SendMessages, PermissionFlagsBits.EmbedLinks])) return message.reply('botun bu kanalda mesaj ve embed gönderme yetkisi yok.');
    if (!args[0] || args.length === 0) {
      return message.reply(buildGiveawayWizard(message.guild));
    }
    const duration = parseDuration(args[0]);
    const winnerCount = Number(args[1]);
    const prize = truncate(args.slice(2).join(' ').trim(), 1000);
    if (!duration || duration < 10_000 || duration > 30 * 24 * 60 * 60 * 1000) {
      return message.reply(buildGiveawayWizard(message.guild));
    }
    if (!Number.isInteger(winnerCount) || winnerCount < 1 || winnerCount > 20 || !prize) {
      return message.reply(buildGiveawayWizard(message.guild));
    }
    const requiredRoleId = database.getConfig(message.guild.id, 'giveaway_required_role_id');
    const minAccountAgeDays = database.getConfig(message.guild.id, 'giveaway_min_account_days', 0);
    const draft = {
      guild_id: message.guild.id, channel_id: message.channel.id, host_id: message.author.id,
      prize, winner_count: winnerCount, required_role_id: requiredRoleId,
      min_account_age_days: minAccountAgeDays, ends_at: Date.now() + duration
    };
    const sent = await message.channel.send({ embeds: [buildGiveawayEmbed(draft, 0)], components: buttonRow() });
    const id = database.createGiveaway(
      draft.guild_id, draft.channel_id, sent.id, draft.host_id, prize, winnerCount,
      requiredRoleId, minAccountAgeDays, draft.ends_at
    );
    scheduleGiveaway(client, { ...draft, id, message_id: sent.id });
    await message.reply(`✅ Çekiliş oluşturuldu: ${sent.url}`);
  }
};
