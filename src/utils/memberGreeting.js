const { AttachmentBuilder, EmbedBuilder } = require('discord.js');
const database = require('../database');
const { createMemberCard } = require('./welcomeCard');
const { COLORS, fillTemplate } = require('./ui');

async function sendMemberGreeting(member, type) {
  const settings = database.getSettings(member.guild.id);
  const welcome = type === 'welcome';
  const enabled = welcome ? settings.welcome_enabled : settings.farewell_enabled;
  const channelId = welcome ? settings.welcome_channel_id : settings.farewell_channel_id;
  if (!enabled || !channelId) return false;

  const channel = await member.guild.channels.fetch(channelId).catch(() => null);
  if (!channel?.isTextBased()) return false;

  const template = welcome ? settings.welcome_message : settings.farewell_message;
  const content = fillTemplate(template, member);
  const embed = new EmbedBuilder()
    .setColor(welcome ? COLORS.success : COLORS.danger)
    .setDescription(content)
    .setThumbnail(member.user.displayAvatarURL({ size: 256 }))
    .setTimestamp();

  try {
    const buffer = await createMemberCard(member, type);
    const filename = `${type}-${member.id}.png`;
    embed.setImage(`attachment://${filename}`);
    await channel.send({ embeds: [embed], files: [new AttachmentBuilder(buffer, { name: filename })] });
  } catch (error) {
    console.error(`[HATA] ${type} kartı:`, error);
    await channel.send({ embeds: [embed] }).catch(() => null);
  }
  return true;
}

module.exports = { sendMemberGreeting };
