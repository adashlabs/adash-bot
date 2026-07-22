const { randomUUID } = require('node:crypto');
const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { COLORS, truncate } = require('./ui');

const sessions = new Map();
const TTL = 60_000;

function disabledRow(approved) {
  return new ActionRowBuilder().addComponents(
    new ButtonBuilder().setCustomId('confirmation_complete').setLabel(approved ? 'Onaylandı' : 'İptal Edildi').setStyle(approved ? ButtonStyle.Success : ButtonStyle.Secondary).setDisabled(true)
  );
}

async function requestConfirmation(message, data, executor) {
  const id = randomUUID().replaceAll('-', '').slice(0, 16);
  const embed = new EmbedBuilder().setColor(COLORS.warning).setTitle(`⚠️ ${data.title || 'İşlem Onayı'}`)
    .setDescription('Bu işlem henüz uygulanmadı. Bilgileri kontrol edip **Evet, uygula** düğmesine bas.')
    .addFields(
      { name: 'Hedef', value: data.target || '—', inline: true },
      { name: 'Yetkili', value: `<@${message.author.id}>`, inline: true },
      { name: 'Sebep', value: truncate(data.reason || 'Sebep belirtilmedi', 1024) }
    ).setFooter({ text: 'Onay 60 saniye geçerlidir.' }).setTimestamp();
  if (data.details) embed.addFields({ name: 'Detay', value: truncate(data.details, 1024) });
  const row = new ActionRowBuilder().addComponents(
    new ButtonBuilder().setCustomId(`modconfirm:yes:${id}`).setLabel('Evet, uygula').setEmoji('✅').setStyle(ButtonStyle.Danger),
    new ButtonBuilder().setCustomId(`modconfirm:no:${id}`).setLabel('Hayır, iptal').setEmoji('✖️').setStyle(ButtonStyle.Secondary)
  );
  const prompt = await message.reply({ embeds: [embed], components: [row] });
  const timer = setTimeout(async () => {
    const session = sessions.get(id);
    if (!session) return;
    sessions.delete(id);
    await session.prompt.edit({ components: [disabledRow(false)] }).catch(() => null);
  }, TTL);
  timer.unref();
  sessions.set(id, { userId: message.author.id, guildId: message.guild.id, executor, prompt, timer });
}

async function handleConfirmation(interaction) {
  const [, decision, id] = interaction.customId.split(':');
  const session = sessions.get(id);
  if (!session) return interaction.reply({ content: 'bu onayın süresi dolmuş veya işlem tamamlanmış.', flags: 64 });
  if (session.userId !== interaction.user.id || session.guildId !== interaction.guildId) {
    return interaction.reply({ content: 'bu moderasyon onayı sana ait değil.', flags: 64 });
  }
  await interaction.deferUpdate();
  clearTimeout(session.timer);
  sessions.delete(id);
  if (decision !== 'yes') {
    await interaction.editReply({ components: [disabledRow(false)] });
    return interaction.followUp({ content: 'İşlem iptal edildi.', flags: 64 });
  }
  await interaction.editReply({ components: [disabledRow(true)] });
  try {
    await session.executor(interaction);
  } catch (error) {
    console.error('[HATA] onaylı moderasyon:', error);
    await interaction.followUp({ content: 'İşlem uygulanamadı. Yetki ve rol sırasını yeniden kontrol et.', flags: 64 }).catch(() => null);
  }
}

module.exports = { requestConfirmation, handleConfirmation, sessions };
