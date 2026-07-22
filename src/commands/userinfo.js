const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { resolveTarget } = require('../utils/security');
const { COLORS } = require('../utils/ui');

module.exports = {
  name: 'userinfo',
  aliases: ['kullanıcı', 'kullanicibilgi'],
  category: 'genel',
  description: 'kullanıcının hesap, sunucu ve rol bilgilerini gösterir.',
  async execute(message, args) {
    const resolved = args[0] ? await resolveTarget(message, args[0]) : { user: message.author, member: message.member };
    if (!resolved) return message.reply('geçerli bir kullanıcı belirt.');
    const { user, member } = resolved;
    const isBot = user.bot ? '🤖 Bot' : '👤 Kullanıcı';
    const isOwner = user.id === message.guild.ownerId ? ' 👑 Sunucu Sahibi' : '';
    const allRoles = member ? member.roles.cache.filter((item) => item.id !== message.guild.id).sort((a, b) => b.position - a.position) : null;
    const roleCount = allRoles ? allRoles.size : 0;
    const rolesDisplay = allRoles && roleCount > 0 ? allRoles.first(8).map((item) => `<@&${item.id}>`).join(' ') + (roleCount > 8 ? ` *(+${roleCount - 8} daha)*` : '') : 'Rol yok';

    const keyPerms = [];
    if (member) {
      if (member.permissions.has('Administrator')) keyPerms.push('Yönetici');
      else {
        if (member.permissions.has('ManageGuild')) keyPerms.push('Sunucuyu Yönet');
        if (member.permissions.has('ManageChannels')) keyPerms.push('Kanalları Yönet');
        if (member.permissions.has('ManageRoles')) keyPerms.push('Rolleri Yönet');
        if (member.permissions.has('ManageMessages')) keyPerms.push('Mesajları Yönet');
        if (member.permissions.has('KickMembers')) keyPerms.push('Üyeleri At');
        if (member.permissions.has('BanMembers')) keyPerms.push('Üyeleri Yasakla');
      }
    }
    const permsDisplay = keyPerms.length ? keyPerms.join(', ') : 'Standart yetkiler';

    const embed = new EmbedBuilder()
      .setColor(member?.displayColor || COLORS.primary)
      .setTitle(`👤 ${user.tag}${isOwner}`)
      .setThumbnail(user.displayAvatarURL({ size: 512, extension: 'png' }))
      .addFields(
        { name: 'Kullanıcı ID', value: `\`${user.id}\``, inline: true },
        { name: 'Tür', value: isBot, inline: true },
        { name: 'Hesap Açılışı', value: `<t:${Math.floor(user.createdTimestamp / 1000)}:R>`, inline: true },
        { name: 'Sunucuya Katılım', value: member?.joinedTimestamp ? `<t:${Math.floor(member.joinedTimestamp / 1000)}:R>` : 'Sunucuda değil', inline: true },
        { name: 'Öne Çıkan Yetkiler', value: permsDisplay, inline: true },
        { name: `Roller (${roleCount})`, value: rolesDisplay }
      )
      .setFooter({ text: `İsteyen: ${message.author.tag}`, iconURL: message.author.displayAvatarURL() })
      .setTimestamp();

    const avatarUrl = user.displayAvatarURL({ size: 4096, extension: 'png' });
    const row = new ActionRowBuilder().addComponents(
      new ButtonBuilder().setStyle(ButtonStyle.Link).setURL(avatarUrl).setLabel('Avatarı Gör').setEmoji('🖼️')
    );
    if (member?.avatar) {
      const guildAvatar = member.avatarURL({ size: 4096, extension: 'png' });
      if (guildAvatar) {
        row.addComponents(new ButtonBuilder().setStyle(ButtonStyle.Link).setURL(guildAvatar).setLabel('Sunucu Profil Resmi').setEmoji('🎨'));
      }
    }

    await message.reply({ embeds: [embed], components: [row] });
  }
};
