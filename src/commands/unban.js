const { PermissionFlagsBits, EmbedBuilder } = require('discord.js');
const { requestConfirmation } = require('../utils/confirmations');
const { sendModLog } = require('../utils/modLog');
const { COLORS, truncate } = require('../utils/ui');
const database = require('../database');

module.exports = {
  name: 'unban', aliases: ['yasakaç', 'yasakac'], category: 'moderasyon',
  description: 'kullanıcının yasağını düğmeli onay sonrasında kaldırır.', cooldown: 3,
  async execute(message, args) {
    if (!message.member.permissions.has(PermissionFlagsBits.BanMembers)) return message.reply('bu işlem için `Üyeleri Yasakla` yetkisi gerekiyor.');
    if (!message.guild.members.me.permissions.has(PermissionFlagsBits.BanMembers)) return message.reply('botta `Üyeleri Yasakla` yetkisi yok.');
    const id = String(args[0] || '').replace(/[<@!>]/g, '');
    if (!/^\d{17,20}$/.test(id)) return message.reply('yasaklı kullanıcının ID’sini belirt. Örnek: `a!unban 123456789012345678 sebep`');
    const ban = await message.guild.bans.fetch(id).catch(() => null);
    if (!ban) return message.reply('bu kullanıcı bu sunucuda yasaklı değil veya botun ban listesini görme yetkisi yok.');
    const reason = truncate(args.slice(1).join(' ') || 'Sebep belirtilmedi', 500);
    await requestConfirmation(message, { title: 'Kullanıcı Yasağını Kaldır', target: `${ban.user.tag} (\`${id}\`)`, reason }, async (interaction) => {
      const current = await message.guild.bans.fetch(id).catch(() => null);
      if (!current) return interaction.followUp({ content: 'kullanıcı artık yasaklı değil.', flags: 64 });
      await message.guild.members.unban(id, reason);
      database.logModAction(message.guild.id, id, interaction.user.id, 'unban', reason);
      await sendModLog(message.guild, { action: 'unban', targetId: id, moderatorId: interaction.user.id, reason, color: COLORS.success });
      await interaction.followUp({ embeds: [new EmbedBuilder().setColor(COLORS.success).setTitle('🔓 Yasak kaldırıldı')
        .addFields({ name: 'Kullanıcı', value: `${ban.user.tag} (\`${id}\`)` }, { name: 'Sebep', value: reason }).setTimestamp()] });
    });
  }
};
