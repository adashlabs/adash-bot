const { SlashCommandBuilder, PermissionFlagsBits, ChannelType, Collection } = require('discord.js');
const database = require('./database');
const help = require('./commands/help');
const setup = require('./commands/setup');
const ping = require('./commands/ping');

const EPHEMERAL = 64;
const user = (builder, required = false) => builder.addUserOption((option) => option.setName('kullanici').setDescription('Hedef kullanıcı').setRequired(required));
const reason = (builder) => builder.addStringOption((option) => option.setName('sebep').setDescription('İşlem sebebi').setMaxLength(500));

const definitions = [
  new SlashCommandBuilder().setName('yardim').setDescription('Etkileşimli yardım menüsünü açar.'),
  new SlashCommandBuilder().setName('kurulum').setDescription('Gelişmiş sunucu kurulum panelini açar.').setDefaultMemberPermissions(PermissionFlagsBits.ManageGuild),
  new SlashCommandBuilder().setName('ping').setDescription('Botun bağlantı ve sistem durumunu gösterir.'),
  new SlashCommandBuilder().setName('sunucu').setDescription('Sunucu bilgilerini gösterir.'),
  user(new SlashCommandBuilder().setName('kullanici').setDescription('Kullanıcı bilgilerini gösterir.')),
  user(new SlashCommandBuilder().setName('avatar').setDescription('Kullanıcının avatarını gösterir.')),
  new SlashCommandBuilder().setName('oyunlar').setDescription('Kanal oyunlarının durumunu gösterir.'),
  new SlashCommandBuilder().setName('tdk').setDescription('TDK sözlükte ayrıntılı arama yapar.').addStringOption((o) => o.setName('kelime').setDescription('Aranacak kelime').setRequired(true).setMaxLength(80)),
  new SlashCommandBuilder().setName('webara').setDescription('Web araması yapar.').addStringOption((o) => o.setName('sorgu').setDescription('Arama sorgusu').setRequired(true).setMaxLength(500)),
  new SlashCommandBuilder().setName('zar').setDescription('Zar atar.').addStringOption((o) => o.setName('zar').setDescription('Örnek: 2d20')),
  new SlashCommandBuilder().setName('yazitura').setDescription('Yazı tura atar.'),
  new SlashCommandBuilder().setName('sekiztop').setDescription('Sihirli küreye soru sorar.').addStringOption((o) => o.setName('soru').setDescription('Sorun').setRequired(true).setMaxLength(500)),
  new SlashCommandBuilder().setName('prefix').setDescription('Komut ön ekini değiştirir.').setDefaultMemberPermissions(PermissionFlagsBits.ManageGuild).addStringOption((o) => o.setName('deger').setDescription('Yeni prefix').setRequired(true).setMaxLength(5)),
  new SlashCommandBuilder().setName('ticketsetup').setDescription('Ticket sistemini tek adımda kurar.').setDefaultMemberPermissions(PermissionFlagsBits.ManageGuild)
    .addChannelOption((o) => o.setName('kategori').setDescription('Ticket kanallarının açılacağı kategori').setRequired(true).addChannelTypes(ChannelType.GuildCategory))
    .addChannelOption((o) => o.setName('panel_kanali').setDescription('Ticket panelinin gönderileceği metin kanalı').setRequired(true).addChannelTypes(ChannelType.GuildText))
    .addChannelOption((o) => o.setName('log_kanali').setDescription('Ticket loglarının gönderileceği metin kanalı').addChannelTypes(ChannelType.GuildText))
    .addRoleOption((o) => o.setName('destek_rolu').setDescription('Ticket yetkili/destek rolü')),
  new SlashCommandBuilder().setName('ticket').setDescription('Açık ticket kanalını yönetir.')
    .addSubcommand((s) => s.setName('ekle').setDescription('Ticket kanalına kullanıcı ekler.').addUserOption((o) => o.setName('kullanici').setDescription('Eklenecek kullanıcı').setRequired(true)))
    .addSubcommand((s) => s.setName('cikar').setDescription('Ticket kanalından kullanıcı çıkarır.').addUserOption((o) => o.setName('kullanici').setDescription('Çıkarılacak kullanıcı').setRequired(true)))
    .addSubcommand((s) => s.setName('adlandir').setDescription('Ticket kanalının adını değiştirir.').addStringOption((o) => o.setName('ad').setDescription('Yeni kanal adı').setRequired(true).setMaxLength(80))),
  new SlashCommandBuilder().setName('cekilis').setDescription('Gelişmiş çekiliş oluşturur.').setDefaultMemberPermissions(PermissionFlagsBits.ManageGuild)
    .addStringOption((o) => o.setName('sure').setDescription('10m, 2h, 3d').setRequired(true))
    .addIntegerOption((o) => o.setName('kazanan').setDescription('Kazanan sayısı').setRequired(true).setMinValue(1).setMaxValue(20))
    .addStringOption((o) => o.setName('odul').setDescription('Çekiliş ödülü').setRequired(true).setMaxLength(1000)),
  new SlashCommandBuilder().setName('cekilisyonet').setDescription('Çekilişi bitirir veya yeniden çeker.').setDefaultMemberPermissions(PermissionFlagsBits.ManageGuild)
    .addSubcommand((s) => s.setName('bitir').setDescription('Aktif çekilişi erken bitirir.').addIntegerOption((o) => o.setName('id').setDescription('Çekiliş ID').setRequired(true).setMinValue(1)))
    .addSubcommand((s) => s.setName('yeniden').setDescription('Kazananları yeniden çeker.').addIntegerOption((o) => o.setName('id').setDescription('Çekiliş ID').setRequired(true).setMinValue(1)).addIntegerOption((o) => o.setName('kazanan').setDescription('Yeni kazanan sayısı').setMinValue(1).setMaxValue(20))),
  reason(user(new SlashCommandBuilder().setName('ban').setDescription('Kullanıcıyı onayla yasaklar.'), true))
    .addIntegerOption((o) => o.setName('mesaj_sil').setDescription('Silinecek geçmiş mesaj günü').setMinValue(0).setMaxValue(7)),
  reason(user(new SlashCommandBuilder().setName('kick').setDescription('Kullanıcıyı onayla sunucudan atar.'), true)),
  new SlashCommandBuilder().setName('unban').setDescription('Kullanıcının yasağını onayla kaldırır.').setDefaultMemberPermissions(PermissionFlagsBits.BanMembers)
    .addStringOption((o) => o.setName('kullanici_id').setDescription('Yasaklı kullanıcının ID’si').setRequired(true).setMinLength(17).setMaxLength(20))
    .addStringOption((o) => o.setName('sebep').setDescription('İşlem sebebi').setMaxLength(500)),
  new SlashCommandBuilder().setName('cases').setDescription('Moderasyon vaka geçmişini görüntüler.').setDefaultMemberPermissions(PermissionFlagsBits.ModerateMembers)
    .addUserOption((o) => o.setName('kullanici').setDescription('Belirli kullanıcının geçmişi')),
  new SlashCommandBuilder().setName('modconfig').setDescription('Moderasyon otomasyonunu ve itiraz kanalını ayarlar.').setDefaultMemberPermissions(PermissionFlagsBits.ManageGuild)
    .addSubcommand((s) => s.setName('uyari').setDescription('Otomatik uyarı cezasını ayarla.').addIntegerOption((o) => o.setName('esik').setDescription('Aktif uyarı eşiği').setRequired(true).setMinValue(1).setMaxValue(10)).addStringOption((o) => o.setName('sure').setDescription('10m, 1h, 2d').setRequired(true)))
    .addSubcommand((s) => s.setName('itiraz').setDescription('İtiraz kanalını ayarla veya kapat.').addChannelOption((o) => o.setName('kanal').setDescription('İtiraz kanalı; boş bırakırsan kapatılır'))),
  new SlashCommandBuilder().setName('itiraz').setDescription('Yetkili ekibe gizli moderasyon itirazı gönderir.').addStringOption((o) => o.setName('metin').setDescription('Ayrıntılı itiraz metni').setRequired(true).setMinLength(20).setMaxLength(1500)),
  user(new SlashCommandBuilder().setName('mute').setDescription('Kullanıcıyı onayla susturur.'), true)
    .addStringOption((o) => o.setName('sure').setDescription('10m, 1h, 2d').setRequired(true))
    .addStringOption((o) => o.setName('sebep').setDescription('İşlem sebebi').setMaxLength(500)),
  reason(user(new SlashCommandBuilder().setName('unmute').setDescription('Susturmayı onayla kaldırır.'), true)),
  reason(user(new SlashCommandBuilder().setName('warn').setDescription('Kullanıcıya onayla uyarı verir.'), true)),
  user(new SlashCommandBuilder().setName('uyarilar').setDescription('Aktif uyarıları gösterir.')),
  user(new SlashCommandBuilder().setName('uyaritemizle').setDescription('Aktif uyarıları onayla temizler.').setDefaultMemberPermissions(PermissionFlagsBits.ModerateMembers), true),
  new SlashCommandBuilder().setName('kilit').setDescription('Kanal kilidini yönetir.').setDefaultMemberPermissions(PermissionFlagsBits.ManageChannels)
    .addStringOption((o) => o.setName('islem').setDescription('Kanalı kilitle veya aç').setRequired(true).addChoices({ name: 'Kilitle', value: 'kilitle' }, { name: 'Kilidi Aç', value: 'aç' })),
  new SlashCommandBuilder().setName('yavasmod').setDescription('Kanal yavaş modunu ayarlar.').setDefaultMemberPermissions(PermissionFlagsBits.ManageChannels)
    .addStringOption((o) => o.setName('sure').setDescription('0, 10s, 5m, 1h').setRequired(true)),
  new SlashCommandBuilder().setName('temizle').setDescription('Mesajları onayla temizler.').setDefaultMemberPermissions(PermissionFlagsBits.ManageMessages)
    .addIntegerOption((o) => o.setName('sayi').setDescription('Silinecek mesaj sayısı').setRequired(true).setMinValue(1).setMaxValue(100))
    .addUserOption((o) => o.setName('kullanici').setDescription('Mesajları filtrelenecek kullanıcı'))
];
for (const definition of definitions) definition.setDMPermission(false);

const mappings = {
  sunucu: ['serverinfo', () => []],
  kullanici: ['userinfo', (i) => [i.options.getUser('kullanici')?.id].filter(Boolean)],
  avatar: ['avatar', (i) => [i.options.getUser('kullanici')?.id].filter(Boolean)],
  oyunlar: ['games', () => []],
  tdk: ['tdk', (i) => [i.options.getString('kelime')]],
  webara: ['wsearch', (i) => [i.options.getString('sorgu')]],
  zar: ['roll', (i) => [i.options.getString('zar')].filter(Boolean)],
  ticketsetup: ['ticketsetup', (i) => [
    i.options.getChannel('kategori').id,
    i.options.getChannel('panel_kanali').id,
    i.options.getChannel('log_kanali')?.id,
    i.options.getRole('destek_rolu')?.id
  ].filter(Boolean)],
  yazitura: ['coinflip', () => []],
  sekiztop: ['8ball', (i) => [i.options.getString('soru')]],
  prefix: ['prefix', (i) => [i.options.getString('deger')]],
  cekilis: ['giveaway', (i) => [i.options.getString('sure'), String(i.options.getInteger('kazanan')), i.options.getString('odul')]],
  ticket: ['ticket', (i) => {
    const subcommand = i.options.getSubcommand();
    if (subcommand === 'adlandir') return ['adlandır', i.options.getString('ad')];
    return [subcommand === 'ekle' ? 'ekle' : 'çıkar', i.options.getUser('kullanici').id];
  }],
  unban: ['unban', (i) => [i.options.getString('kullanici_id'), i.options.getString('sebep')].filter(Boolean)],
  cekilisyonet: ['giveawaymanage', (i) => [i.options.getSubcommand(), String(i.options.getInteger('id')), String(i.options.getInteger('kazanan') || '')].filter(Boolean)],
  cases: ['cases', (i) => i.options.getUser('kullanici') ? [i.options.getUser('kullanici').id] : []],
  modconfig: ['modconfig', (i) => i.options.getSubcommand() === 'uyari' ? ['warn', String(i.options.getInteger('esik')), i.options.getString('sure')] : ['appeal', i.options.getChannel('kanal')?.id || 'kapalı']],
  itiraz: ['appeal', (i) => [i.options.getString('metin')]],
  ban: ['ban', (i) => [i.options.getUser('kullanici').id, i.options.getString('sebep'), `--days=${i.options.getInteger('mesaj_sil') || 0}`].filter(Boolean)],
  kick: ['kick', userReasonArgs], unmute: ['unmute', userReasonArgs], warn: ['warn', userReasonArgs],
  mute: ['mute', (i) => [i.options.getUser('kullanici').id, i.options.getString('sure'), i.options.getString('sebep')].filter(Boolean)],
  uyarilar: ['warnings', (i) => [i.options.getUser('kullanici')?.id].filter(Boolean)],
  uyaritemizle: ['clearwarns', (i) => [i.options.getUser('kullanici').id]],
  temizle: ['clear', (i) => [String(i.options.getInteger('sayi')), i.options.getUser('kullanici')?.id].filter(Boolean)],
  kilit: ['lock', (i) => [i.options.getString('islem')]],
  yavasmod: ['slowmode', (i) => [i.options.getString('sure')]]
};

function userReasonArgs(interaction) {
  return [interaction.options.getUser('kullanici').id, interaction.options.getString('sebep')].filter(Boolean);
}

function messageAdapter(interaction) {
  const selectedUser = interaction.options.getUser('kullanici');
  const users = new Collection();
  if (selectedUser) users.set(selectedUser.id, selectedUser);
  let initialResponse = true;
  return {
    id: interaction.id,
    author: interaction.user,
    member: interaction.member,
    guild: interaction.guild,
    channel: interaction.channel,
    client: interaction.client,
    createdTimestamp: interaction.createdTimestamp,
    mentions: { users: { first: () => users.first() } },
    async reply(payload) {
      if (initialResponse) {
        initialResponse = false;
        return interaction.editReply(payload);
      }
      return interaction.followUp(payload);
    },
    async delete() {}
  };
}

async function runSlash(interaction) {
  try {
    if (interaction.commandName === 'yardim') return interaction.reply({ ...help.buildHelp(interaction.client, database.getPrefix(interaction.guildId), 'genel', interaction.user.id), flags: EPHEMERAL });
    if (interaction.commandName === 'kurulum') return interaction.reply({ ...setup.buildSetupPanel(interaction.guild), flags: EPHEMERAL });
    if (interaction.commandName === 'ping') return interaction.reply(ping.buildOutput(interaction.client, interaction.guildId));
    const mapping = mappings[interaction.commandName];
    if (!mapping) return interaction.reply({ content: 'slash komutu eşleştirilemedi.', flags: EPHEMERAL });
    await interaction.deferReply();
    const [commandName, argsBuilder] = mapping;
    const command = interaction.client.commands.get(commandName);
    if (!command) return interaction.editReply('komut yüklenemedi.');
    database.logCommand(interaction.guildId, interaction.user.id, commandName, argsBuilder(interaction).join(' '));
    return await command.execute(messageAdapter(interaction), argsBuilder(interaction), interaction.client);
  } catch (error) {
    console.error(`[HATA] runSlash (${interaction.commandName}):`, error);
    const message = 'Komut çalıştırılırken bir hata oluştu. Yetkileri ve rol sırasını kontrol et.';
    if (interaction.deferred || interaction.replied) {
      await interaction.editReply({ content: message }).catch(() => null);
    } else {
      await interaction.reply({ content: message, flags: EPHEMERAL }).catch(() => null);
    }
  }
}

module.exports = { slashCommands: definitions.map((item) => item.toJSON()), runSlash, messageAdapter };
