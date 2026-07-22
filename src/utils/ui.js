const { EmbedBuilder } = require('discord.js');

const COLORS = {
  primary: 0x5865F2,
  success: 0x57F287,
  warning: 0xFEE75C,
  danger: 0xED4245,
  info: 0x3498DB,
  neutral: 0x2B2D31
};

function truncate(value, max) {
  const text = String(value ?? '');
  return text.length <= max ? text : `${text.slice(0, Math.max(0, max - 1))}…`;
}

function fillTemplate(template, member) {
  return truncate(String(template)
    .replaceAll('{user}', `<@${member.id}>`)
    .replaceAll('{username}', member.user.username)
    .replaceAll('{server}', member.guild.name)
    .replaceAll('{memberCount}', String(member.guild.memberCount)), 2000);
}

async function transient(channel, content, milliseconds = 5000) {
  const sent = await channel.send(typeof content === 'string' ? { content } : content).catch(() => null);
  if (sent) setTimeout(() => sent.delete().catch(() => {}), milliseconds).unref?.();
  return sent;
}

function createEmbed(options = {}) {
  const {
    type = 'primary',
    title,
    description,
    fields = [],
    footer,
    author,
    thumbnail,
    image,
    timestamp = true,
    user
  } = options;

  const color = COLORS[type] || COLORS.primary;
  const embed = new EmbedBuilder().setColor(color);

  if (title) embed.setTitle(truncate(title, 256));
  if (description) embed.setDescription(truncate(description, 4000));
  if (Array.isArray(fields) && fields.length > 0) {
    for (const field of fields.slice(0, 25)) {
      if (field?.name && field?.value) {
        embed.addFields({
          name: truncate(field.name, 256),
          value: truncate(field.value, 1024),
          inline: Boolean(field.inline)
        });
      }
    }
  }
  if (author) {
    embed.setAuthor({
      name: truncate(author.name || author.tag || String(author), 256),
      iconURL: author.iconURL || author.displayAvatarURL?.({ size: 128 })
    });
  }
  if (thumbnail) embed.setThumbnail(thumbnail);
  if (image) embed.setImage(image);
  if (footer) {
    const footerText = typeof footer === 'string' ? footer : footer.text;
    const footerIcon = typeof footer === 'object' ? footer.iconURL : undefined;
    embed.setFooter({ name: undefined, text: truncate(footerText, 2048), iconURL: footerIcon });
  } else if (user) {
    embed.setFooter({
      text: truncate(`İsteyen: ${user.tag || user.username || user.id}`, 2048),
      iconURL: user.displayAvatarURL?.({ size: 128 })
    });
  }
  if (timestamp) embed.setTimestamp();

  return embed;
}

function successEmbed(title, description, user) {
  return createEmbed({ type: 'success', title: `✅ ${title}`, description, user });
}

function errorEmbed(title, description, user) {
  return createEmbed({ type: 'danger', title: `❌ ${title}`, description, user });
}

function warningEmbed(title, description, user) {
  return createEmbed({ type: 'warning', title: `⚠️ ${title}`, description, user });
}

function infoEmbed(title, description, user) {
  return createEmbed({ type: 'info', title: `ℹ️ ${title}`, description, user });
}

module.exports = {
  COLORS,
  truncate,
  fillTemplate,
  transient,
  createEmbed,
  successEmbed,
  errorEmbed,
  warningEmbed,
  infoEmbed
};
