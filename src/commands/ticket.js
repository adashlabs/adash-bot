const { PermissionFlagsBits, EmbedBuilder } = require('discord.js');
const database = require('../database');
const { resolveTarget } = require('../utils/security');
const { isSupport } = require('../tickets');
const { COLORS } = require('../utils/ui');
module.exports = {
  name: 'ticket',
  aliases: ['talep'],
  category: 'moderasyon',
  description: 'ticket kanalına üye ekler/çıkarır veya kanalı yeniden adlandırır.',
  cooldown: 2,
  async execute(message, args) {
    const ticket = database.getTicket(message.channel.id);
    if (!ticket || ticket.closed_at) {
      const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle('⚠️ Geçersiz Kanal')
        .setDescription('Bu komut yalnızca açık bir ticket kanalında kullanılabilir.');
      return message.reply({ embeds: [embed] });
    }
    if (!isSupport(message.member, message.guild.id)) {
      const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('❌ Yetersiz Yetki')
        .setDescription('Ticket yönetmek için destek rolüne veya `Kanalları Yönet` yetkisine sahip olmalısın.');
      return message.reply({ embeds: [embed] });
    }
    const action = args[0]?.toLocaleLowerCase('tr-TR');
    if (['add', 'ekle', 'remove', 'çıkar', 'cikar'].includes(action)) {
      const target = await resolveTarget(message, args[1]);
      if (!target?.member) {
        const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle('⚠️ Kullanıcı Bulunamadı')
          .setDescription('Lütfen sunucudaki geçerli bir kullanıcıyı etiketle veya ID’sini belirt.');
        return message.reply({ embeds: [embed] });
      }
      if (['add', 'ekle'].includes(action)) {
        await message.channel.permissionOverwrites.edit(target.member, { ViewChannel: true, SendMessages: true, ReadMessageHistory: true });
        const embed = new EmbedBuilder().setColor(COLORS.success).setTitle('👤 Üye Eklendi')
          .setDescription(`<@${target.user.id}> bu ticket kanalına başarıyla eklendi.`)
          .setFooter({ text: `İşlem yapan: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() }).setTimestamp();
        return message.reply({ embeds: [embed] });
      }
      if (target.user.id === ticket.owner_id) {
        const embed = new EmbedBuilder().setColor(COLORS.danger).setTitle('❌ İşlem Engellendi')
          .setDescription('Ticket sahibi kendi destek kanalından çıkarılamaz.');
        return message.reply({ embeds: [embed] });
      }
      await message.channel.permissionOverwrites.delete(target.member);
      const embed = new EmbedBuilder().setColor(COLORS.success).setTitle('👤 Üye Çıkarıldı')
        .setDescription(`<@${target.user.id}> bu ticket kanalından çıkarıldı.`)
        .setFooter({ text: `İşlem yapan: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() }).setTimestamp();
      return message.reply({ embeds: [embed] });
    }
    if (['rename', 'adlandır', 'adlandir'].includes(action)) {
      const name = args.slice(1).join('-').toLocaleLowerCase('tr-TR').replace(/[^a-z0-9çğıöşü-]/g, '-').replace(/-+/g, '-').slice(0, 80);
      if (name.length < 2) {
        const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle('⚠️ Geçersiz Kanal Adı')
          .setDescription('Lütfen en az 2 karakterlik geçerli bir kanal adı belirt.');
        return message.reply({ embeds: [embed] });
      }
      await message.channel.setName(name, `Ticket adı: ${message.author.tag}`);
      const embed = new EmbedBuilder().setColor(COLORS.success).setTitle('✏️ Kanal Adı Değiştirildi')
        .setDescription(`Ticket kanalı adı **${name}** olarak güncellendi.`)
        .setFooter({ text: `İşlem yapan: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() }).setTimestamp();
      return message.reply({ embeds: [embed] });
    }

    const embed = new EmbedBuilder().setColor(COLORS.primary).setTitle('🎫 Ticket Yönetim Rehberi')
      .addFields(
        { name: 'Üye Ekle', value: '`a!ticket ekle @üye`' },
        { name: 'Üye Çıkar', value: '`a!ticket çıkar @üye`' },
        { name: 'Kanalı Adlandır', value: '`a!ticket adlandır yeni-kanal-adı`' }
      )
      .setFooter({ text: `İsteyen: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() }).setTimestamp();
    return message.reply({ embeds: [embed] });
  }
};
