const {
  ActionRowBuilder,
  ButtonBuilder,
  ButtonStyle,
  EmbedBuilder,
  ModalBuilder,
  TextInputBuilder,
  TextInputStyle,
  PermissionFlagsBits
} = require('discord.js');

const drafts = new Map();
const DRAFT_TTL = 15 * 60 * 1000;

function trim(value, max) {
  return String(value || '').trim().slice(0, max);
}

function parseColor(value) {
  const color = trim(value, 7).replace(/^#/, '');
  return /^[0-9a-f]{6}$/i.test(color) ? `#${color.toUpperCase()}` : '#5865F2';
}

function parseFields(value) {
  return trim(value, 2000).split('\n').map((line) => line.trim()).filter(Boolean).slice(0, 10).map((line) => {
    const [name, ...rest] = line.split('|');
    return { name: trim(name, 256) || 'Alan', value: trim(rest.join('|'), 1024) || '—', inline: false };
  });
}

function buildEmbed(draft, preview = false) {
  const embed = new EmbedBuilder().setColor(parseColor(draft.color)).setTimestamp();
  if (draft.title) embed.setTitle(draft.title);
  if (draft.description) embed.setDescription(draft.description);
  if (draft.url && /^https?:\/\//i.test(draft.url)) embed.setURL(draft.url);
  if (draft.footer) embed.setFooter({ text: draft.footer });
  if (draft.image && /^https?:\/\//i.test(draft.image)) embed.setImage(draft.image);
  if (draft.thumbnail && /^https?:\/\//i.test(draft.thumbnail)) embed.setThumbnail(draft.thumbnail);
  if (draft.author) embed.setAuthor({ name: draft.author });
  if (draft.fields?.length) embed.addFields(draft.fields);
  if (preview && !draft.title && !draft.description && !draft.fields?.length && !draft.image && !draft.thumbnail) {
    embed.setDescription('Embed önizlemesi hazır. İçeriği düzenlemek için aşağıdaki butona bas.');
  }
  return embed;
}

function buildControls(ownerId) {
  return new ActionRowBuilder().addComponents(
    new ButtonBuilder().setCustomId(`embed_builder:edit:${ownerId}`).setLabel('İçeriği Düzenle').setStyle(ButtonStyle.Primary),
    new ButtonBuilder().setCustomId(`embed_builder:send:${ownerId}`).setLabel('Kanala Gönder').setStyle(ButtonStyle.Success),
    new ButtonBuilder().setCustomId(`embed_builder:cancel:${ownerId}`).setLabel('İptal').setStyle(ButtonStyle.Danger)
  );
}

function buildModal(ownerId, draft = {}) {
  const modal = new ModalBuilder().setCustomId(`embed_builder:modal:${ownerId}`).setTitle('Embed Builder');
  const input = (id, label, style, value, max, required = false) => new ActionRowBuilder().addComponents(
    new TextInputBuilder().setCustomId(id).setLabel(label).setStyle(style).setRequired(required).setMaxLength(max).setValue(value || '')
  );
  modal.addComponents(
    input('title', 'Başlık', TextInputStyle.Short, draft.title, 256),
    input('description', 'Açıklama', TextInputStyle.Paragraph, draft.description, 4000),
    input('color', 'Renk (#5865F2)', TextInputStyle.Short, draft.color || '#5865F2', 7),
    input('footer', 'Alt bilgi', TextInputStyle.Short, draft.footer, 2048),
    input('extras', 'URL / Görsel / Küçük görsel / Yazar / Alanlar', TextInputStyle.Paragraph, [draft.url, draft.image, draft.thumbnail, draft.author, (draft.fields || []).map((field) => `${field.name} | ${field.value}`).join('\n')].join('\n'), 4000)
  );
  return modal;
}

function createDraft(ownerId, channel) {
  const draft = { ownerId, channelId: channel.id, title: '', description: '', color: '#5865F2', footer: '', url: '', image: '', thumbnail: '', author: '', fields: [], updatedAt: Date.now() };
  drafts.set(ownerId, draft);
  return draft;
}

function getDraft(ownerId) {
  const draft = drafts.get(ownerId);
  if (!draft || Date.now() - draft.updatedAt > DRAFT_TTL) {
    drafts.delete(ownerId);
    return null;
  }
  return draft;
}

module.exports = {
  name: 'embed',
  aliases: ['embedbuilder'],
  category: 'ayarlar',
  description: 'Yöneticiler için etkileşimli embed oluşturucu açar.',
  cooldown: 5,
  drafts,
  buildEmbed,
  buildControls,
  buildModal,
  createDraft,
  getDraft,
  parseColor,
  parseFields,
  async execute(message) {
    if (!message.member?.permissions.has(PermissionFlagsBits.ManageGuild)) {
      return message.reply('bu komutu yalnızca `Sunucuyu Yönet` yetkisi olan yöneticiler kullanabilir.');
    }
    const draft = createDraft(message.author.id, message.channel);
    const panel = await message.reply({ embeds: [buildEmbed(draft, true)], components: [buildControls(message.author.id)] });
    draft.panelMessageId = panel.id;
    return panel;
  }
};
