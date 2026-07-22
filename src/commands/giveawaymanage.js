const { PermissionFlagsBits } = require('discord.js');
const database = require('../database');
const { requestConfirmation } = require('../utils/confirmations');
const { finishGiveaway, rerollGiveaway } = require('../giveaways');

module.exports = {
  name: 'giveawaymanage',
  aliases: ['çekilişyönet', 'cekilisyonet'],
  category: 'ayarlar',
  description: 'çekilişi erken bitirir veya sonuçlanmış çekilişi yeniden çeker.',
  cooldown: 3,
  async execute(message, args, client) {
    if (!message.member.permissions.has(PermissionFlagsBits.ManageGuild)) return message.reply('bu işlem için `Sunucuyu Yönet` yetkisi gerekiyor.');
    const action = args[0]?.toLocaleLowerCase('tr-TR');
    const id = Number(args[1]);
    const giveaway = Number.isInteger(id) ? database.getGiveawayById(id) : null;
    if (!giveaway || giveaway.guild_id !== message.guild.id) return message.reply('bu sunucuya ait geçerli bir çekiliş ID’si belirt.');
    if (['bitir', 'end'].includes(action)) {
      if (giveaway.ended_at) return message.reply('bu çekiliş zaten sonuçlanmış.');
      return requestConfirmation(message, { title: 'Çekilişi Erken Bitir', target: `Çekiliş #${id}`, reason: giveaway.prize, details: 'Katılım hemen kapatılacak ve kazananlar çekilecek.' }, async (interaction) => {
        await finishGiveaway(client, giveaway);
        await interaction.followUp(`✅ Çekiliş #${id} sonuçlandırıldı.`);
      });
    }
    if (['yeniden', 'reroll'].includes(action)) {
      if (!giveaway.ended_at) return message.reply('yalnızca sonuçlanmış çekiliş yeniden çekilebilir.');
      const count = Math.min(20, Math.max(1, Number(args[2]) || giveaway.winner_count));
      return requestConfirmation(message, { title: 'Kazananları Yeniden Çek', target: `Çekiliş #${id}`, reason: giveaway.prize, details: `${count} yeni kazanan seçilecek.` }, async (interaction) => {
        const winners = await rerollGiveaway(client, id, count);
        await interaction.followUp(winners?.length ? `✅ Yeni kazananlar: ${winners.map((userId) => `<@${userId}>`).join(', ')}` : 'katılımcı bulunamadı.');
      });
    }
    return message.reply('kullanım: `a!giveawaymanage bitir <id>` veya `a!giveawaymanage yeniden <id> [sayı]`');
  }
};
