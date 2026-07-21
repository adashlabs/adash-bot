const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');

const API_KEY = 'sk_master_15f26886920665d86686361576a67e144e0c3788bf08098b305233e96dc351805702ea83b28499fb8fb4c07769246c89';
const API_BASE = 'https://api.synapicsearch.com/api/search';
const PER_PAGE = 5;
const SESSION_TTL = 5 * 60 * 1000;

const sessions = new Map();

setInterval(() => {
  const now = Date.now();
  for (const [id, s] of sessions) {
    if (now - s.timestamp > SESSION_TTL) sessions.delete(id);
  }
}, 60 * 1000);

function buildEmbed(session, page) {
  const start = (page - 1) * PER_PAGE;
  const end = Math.min(start + PER_PAGE, session.results.length);
  const slice = session.results.slice(start, end);

  const embed = new EmbedBuilder()
    .setTitle(`\uD83D\uDD0D "${session.query}" \u2014 Sayfa ${page}/${session.totalPages}`)
    .setColor(0x5865F2)
    .setTimestamp();

  for (let i = 0; i < slice.length; i++) {
    const r = slice[i];
    const desc = (r.description || r.text || 'A\u00E7\u0131klama yok');
    const short = desc.length > 200 ? desc.slice(0, 197) + '...' : desc;
    const thumb = r.thumbnail || r.image || r.src || r.source_url || null;
    const field = {
      name: `${start + i + 1}. ${r.title || 'Ba\u015Fl\u0131ks\u0131z'}`,
      value: `${short}\n\u{1F517} [${r.url}](${r.url})`
    };
    if (i === 0 && thumb) {
      embed.setThumbnail(thumb);
    }
    embed.addFields(field);
  }

  embed.setFooter({ text: 'Synapic Search Engine' });
  return embed;
}

function buildRow(sessionId, page, totalPages) {
  return new ActionRowBuilder().addComponents(
    new ButtonBuilder()
      .setCustomId(`wsearch:prev:${sessionId}`)
      .setLabel('\u25C0 Geri')
      .setStyle(ButtonStyle.Secondary)
      .setDisabled(page <= 1),
    new ButtonBuilder()
      .setCustomId(`wsearch:next:${sessionId}`)
      .setLabel('\u0130leri \u25B6')
      .setStyle(ButtonStyle.Secondary)
      .setDisabled(page >= totalPages)
  );
}

async function search(query) {
  const url = `${API_BASE}?q=${encodeURIComponent(query)}&apikey=${API_KEY}`;
  const res = await fetch(url);
  if (!res.ok) throw new Error(`API hatas\u0131: ${res.status}`);
  const data = await res.json();
  return data.results || [];
}

module.exports = {
  name: 'wsearch',
  category: 'genel',
  description: 'Synapic arama motoru ile web aramas\u0131 yapar. kullan\u0131m: a!wsearch <sorgu>',
  sessions,
  buildEmbed,
  buildRow,

  async execute(message, args, client) {
    const query = args.join(' ');
    if (!query) {
      return message.reply('bir arama sorgusu belirt. \u00F6rnek: `a!wsearch discord bot`');
    }

    await message.channel.sendTyping();

    let results;
    try {
      results = await search(query);
    } catch (e) {
      console.error('[HATA] wsearch:', e);
      return message.reply('arama s\u0131ras\u0131nda bir hata olu\u015Ftu.');
    }

    if (results.length === 0) {
      return message.reply('sonu\u00E7 bulunamad\u0131.');
    }

    const totalPages = Math.ceil(results.length / PER_PAGE);
    const sessionId = Date.now().toString(36) + Math.random().toString(36).slice(2, 8);

    const session = {
      userId: message.author.id,
      query,
      results,
      page: 1,
      totalPages,
      timestamp: Date.now()
    };
    sessions.set(sessionId, session);

    const embed = buildEmbed(session, 1);
    const row = buildRow(sessionId, 1, totalPages);

    await message.reply({ embeds: [embed], components: [row] });
  }
};