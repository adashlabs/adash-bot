const { randomInt } = require('node:crypto');
const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const database = require('./database');
const { COLORS, truncate } = require('./utils/ui');

const scheduled = new Map();
const MAX_DELAY = 2 ** 31 - 1;

function requirementsText(giveaway) {
  const lines = [];
  if (giveaway.required_role_id) lines.push(`Gerekli rol: <@&${giveaway.required_role_id}>`);
  if (giveaway.min_account_age_days) lines.push(`Minimum hesap yaşı: ${giveaway.min_account_age_days} gün`);
  return lines.length ? lines.join('\n') : 'Ek katılım şartı yok.';
}

function calculateChance(entriesCount, winnerCount) {
  if (entriesCount <= 0) return '0%';
  const chance = Math.min(100, (winnerCount / entriesCount) * 100);
  const ratio = Math.ceil(entriesCount / Math.min(entriesCount, winnerCount));
  return `%${chance.toFixed(1)} (Yaklaşık 1 / ${ratio})`;
}

function buildGiveawayEmbed(giveaway, entries = 0, ended = false, winners = []) {
  const chanceText = ended ? null : calculateChance(entries, giveaway.winner_count);
  return new EmbedBuilder().setColor(ended ? COLORS.danger : COLORS.primary)
    .setTitle(ended ? '🎉 Çekiliş Sonuçlandı' : '🎉 Gelişmiş Çekiliş')
    .setDescription([
      `**Ödül:** ${truncate(giveaway.prize, 1000)}`,
      `**Kazanan Sayısı:** ${giveaway.winner_count}`,
      `**Toplam Katılımcı:** ${entries}`,
      ended ? null : `**Katılımcı Başına Şans:** ${chanceText}`,
      ended ? `**Kazananlar:** ${winners.length ? winners.map((id) => `<@${id}>`).join(', ') : 'Katılımcı yok'}` : `**Bitiş:** <t:${Math.floor(giveaway.ends_at / 1000)}:F> (<t:${Math.floor(giveaway.ends_at / 1000)}:R>)`,
      '', `**Katılım Şartları**\n${requirementsText(giveaway)}`
    ].filter(Boolean).join('\n')).setFooter({ text: `Çekiliş Sahibi ID: ${giveaway.host_id} · ID: #${giveaway.id || 'oluşturuluyor'}` }).setTimestamp();
}

function buttonRow(ended = false) {
  return [new ActionRowBuilder().addComponents(new ButtonBuilder().setCustomId('giveaway_join')
    .setLabel(ended ? 'Çekiliş Bitti' : 'Katıl / Ayrıl').setEmoji('🎉').setStyle(ButtonStyle.Success).setDisabled(ended))];
}

function chooseWinners(entries, count) {
  const pool = [...entries];
  const winners = [];
  while (pool.length && winners.length < count) winners.push(pool.splice(randomInt(pool.length), 1)[0]);
  return winners;
}

async function logGiveaway(client, giveaway, content) {
  const channelId = database.getConfig(giveaway.guild_id, 'giveaway_log_channel_id');
  if (!channelId) return;
  const channel = await client.channels.fetch(channelId).catch(() => null);
  if (channel?.isTextBased()) await channel.send({ content }).catch(() => null);
}

async function finishGiveaway(client, giveaway) {
  if (!database.endGiveaway(giveaway.id)) return [];
  const entries = database.getGiveawayEntries(giveaway.id);
  const winners = chooseWinners(entries, giveaway.winner_count);
  const channel = await client.channels.fetch(giveaway.channel_id).catch(() => null);
  if (channel?.isTextBased()) {
    const message = await channel.messages.fetch(giveaway.message_id).catch(() => null);
    if (message) await message.edit({ embeds: [buildGiveawayEmbed(giveaway, entries.length, true, winners)], components: buttonRow(true) }).catch(() => null);
    await channel.send(winners.length ? `🎉 Tebrikler ${winners.map((id) => `<@${id}>`).join(', ')}! **${giveaway.prize}** kazandınız.` : `🎉 **${giveaway.prize}** çekilişi katılımcı olmadığı için sonuçlanamadı.`).catch(() => null);
  }
  await logGiveaway(client, giveaway, `🏁 Çekiliş #${giveaway.id} sonuçlandı · ödül: **${giveaway.prize}** · kazananlar: ${winners.length ? winners.map((id) => `<@${id}>`).join(', ') : 'yok'}`);
  scheduled.delete(giveaway.id);
  return winners;
}

function scheduleGiveaway(client, giveaway) {
  clearTimeout(scheduled.get(giveaway.id));
  const delay = Math.max(0, giveaway.ends_at - Date.now());
  const timer = setTimeout(async () => {
    if (delay > MAX_DELAY) return scheduleGiveaway(client, giveaway);
    await finishGiveaway(client, giveaway).catch((error) => console.error('[HATA] çekiliş bitirme:', error));
  }, Math.min(delay, MAX_DELAY));
  timer.unref();
  scheduled.set(giveaway.id, timer);
}

async function restoreGiveaways(client) {
  for (const giveaway of database.getDueGiveaways()) await finishGiveaway(client, giveaway).catch((error) => console.error('[HATA] gecikmiş çekiliş:', error));
  for (const giveaway of database.raw.prepare('SELECT * FROM giveaways WHERE ended_at IS NULL AND ends_at > ?').all(Date.now())) scheduleGiveaway(client, giveaway);
}

async function toggleEntry(interaction) {
  const giveaway = database.getGiveawayByMessage(interaction.message.id);
  if (!giveaway || giveaway.ended_at || giveaway.ends_at <= Date.now()) return interaction.followUp({ content: 'bu çekiliş artık aktif değil.', flags: 64 });
  if (giveaway.required_role_id && !interaction.member.roles.cache.has(giveaway.required_role_id)) return interaction.followUp({ content: `katılmak için <@&${giveaway.required_role_id}> rolüne sahip olmalısın.`, flags: 64 });
  const accountAgeDays = Math.floor((Date.now() - interaction.user.createdTimestamp) / 86400000);
  if (accountAgeDays < giveaway.min_account_age_days) return interaction.followUp({ content: `hesabın en az ${giveaway.min_account_age_days} günlük olmalı. Mevcut: ${accountAgeDays} gün.`, flags: 64 });
  const joined = database.joinGiveaway(giveaway.id, interaction.user.id);
  if (!joined) database.leaveGiveaway(giveaway.id, interaction.user.id);
  const entries = database.getGiveawayEntries(giveaway.id);
  await interaction.editReply({ embeds: [buildGiveawayEmbed(giveaway, entries.length)], components: buttonRow() });
  const chance = calculateChance(entries.length, giveaway.winner_count);
  const feedback = joined
    ? `🎉 Çekilişe katıldın! Toplam Katılımcı: **${entries.length}** · Tahmini Şansın: **${chance}**`
    : '👋 Çekilişten ayrıldın.';
  await interaction.followUp({ content: feedback, flags: 64 });
}

async function rerollGiveaway(client, giveawayId, count = 1) {
  const giveaway = database.getGiveawayById(giveawayId);
  if (!giveaway?.ended_at) return null;
  const winners = chooseWinners(database.getGiveawayEntries(giveaway.id), count);
  const channel = await client.channels.fetch(giveaway.channel_id).catch(() => null);
  if (channel?.isTextBased()) await channel.send(winners.length ? `🔁 Çekiliş #${giveaway.id} yeniden çekildi: ${winners.map((id) => `<@${id}>`).join(', ')}` : 'yeniden çekilecek katılımcı bulunamadı.');
  await logGiveaway(client, giveaway, `🔁 Çekiliş #${giveaway.id} yeniden çekildi · ${winners.map((id) => `<@${id}>`).join(', ') || 'kazanan yok'}`);
  return winners;
}

module.exports = { buildGiveawayEmbed, buttonRow, scheduleGiveaway, restoreGiveaways, toggleEntry, finishGiveaway, rerollGiveaway, calculateChance };
