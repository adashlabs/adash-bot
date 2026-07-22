const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { truncate, COLORS } = require('../utils/ui');

const API_BASE = 'https://api.synapicsearch.com/api/search';
const PER_PAGE = 5;
const SESSION_TTL = 5 * 60 * 1000;
const sessions = new Map();

const cleanupTimer = setInterval(() => {
  const now = Date.now();
  for (const [id, session] of sessions) if (now - session.timestamp > SESSION_TTL) sessions.delete(id);
}, 60 * 1000);
cleanupTimer.unref();

function safeUrl(value) {
  try {
    const url = new URL(value);
    return ['http:', 'https:'].includes(url.protocol) ? url.href : null;
  } catch {
    return null;
  }
}

function buildEmbed(session, page) {
  const start = (page - 1) * PER_PAGE;
  const slice = session.results.slice(start, start + PER_PAGE);
  const embed = new EmbedBuilder().setTitle(`🔎 ${truncate(session.query, 180)} · ${page}/${session.totalPages}`)
    .setColor(COLORS.primary).setFooter({ text: `${session.source || 'DuckDuckGo + Wikipedia'} · sonuçları açarken bağlantıyı kontrol et` }).setTimestamp();

  for (let index = 0; index < slice.length; index += 1) {
    const result = slice[index];
    const url = safeUrl(result.url);
    const description = truncate(result.description || result.text || 'Açıklama yok', 350);
    embed.addFields({
      name: truncate(`${start + index + 1}. ${result.title || 'Başlıksız'}`, 256),
      value: url ? `${description}\n🔗 [Sonucu aç](${url})` : description
    });
    if (index === 0) {
      const image = safeUrl(result.thumbnail || result.image || result.src || result.source_url);
      if (image) embed.setThumbnail(image);
    }
  }
  return embed;
}

function buildRow(sessionId, page, totalPages) {
  return new ActionRowBuilder().addComponents(
    new ButtonBuilder().setCustomId(`wsearch:prev:${sessionId}`).setLabel('Geri').setEmoji('◀️').setStyle(ButtonStyle.Secondary).setDisabled(page <= 1),
    new ButtonBuilder().setCustomId(`wsearch:next:${sessionId}`).setLabel('İleri').setEmoji('▶️').setStyle(ButtonStyle.Secondary).setDisabled(page >= totalPages)
  );
}

async function fallbackSearch(query) {
  const results = [];
  try {
    const ddgUrl = `https://api.duckduckgo.com/?q=${encodeURIComponent(query)}&format=json&no_html=1&skip_disambig=1`;
    const ddgRes = await fetch(ddgUrl, { signal: AbortSignal.timeout(6000), headers: { 'User-Agent': 'Mozilla/5.0' } });
    if (ddgRes.ok) {
      const ddgData = await ddgRes.json();
      if (ddgData.AbstractText) {
        results.push({
          title: ddgData.Heading || query,
          description: ddgData.AbstractText,
          url: ddgData.AbstractURL || `https://duckduckgo.com/?q=${encodeURIComponent(query)}`
        });
      }
      if (Array.isArray(ddgData.RelatedTopics)) {
        for (const item of ddgData.RelatedTopics) {
          if (item.Text && item.FirstURL) {
            results.push({ title: item.Text.slice(0, 60), description: item.Text, url: item.FirstURL });
          }
        }
      }
    }
  } catch (_) {}

  if (results.length < 5) {
    try {
      const wikiUrl = `https://tr.wikipedia.org/w/api.php?action=query&list=search&srsearch=${encodeURIComponent(query)}&format=json&origin=*`;
      const wikiRes = await fetch(wikiUrl, { signal: AbortSignal.timeout(6000) });
      if (wikiRes.ok) {
        const wikiData = await wikiRes.json();
        const searchHits = wikiData?.query?.search || [];
        for (const hit of searchHits) {
          const snippet = String(hit.snippet || '').replace(/<[^>]*>/g, '');
          results.push({
            title: hit.title,
            description: snippet,
            url: `https://tr.wikipedia.org/wiki/${encodeURIComponent(hit.title.replaceAll(' ', '_'))}`
          });
        }
      }
    } catch (_) {}
  }
  return results.slice(0, 50);
}

async function search(query) {
  const key = process.env.SYNAPIC_API_KEY;
  if (key) {
    try {
      const url = `${API_BASE}?q=${encodeURIComponent(query)}&apikey=${encodeURIComponent(key)}`;
      const response = await fetch(url, { signal: AbortSignal.timeout(10000) });
      if (response.ok) {
        const data = await response.json();
        if (Array.isArray(data.results) && data.results.length > 0) return data.results.slice(0, 50);
      }
    } catch (_) {}
  }
  return fallbackSearch(query);
}

module.exports = {
  name: 'wsearch',
  aliases: ['webara'],
  category: 'genel',
  description: 'API anahtarı olmadan DuckDuckGo ve Türkçe Vikipedi ile sayfalı web araması yapar.',
  cooldown: 8,
  sessions,
  buildEmbed,
  buildRow,
  async execute(message, args) {
    const query = args.join(' ').trim();
    if (!query) return message.reply('arama sorgusu belirt. örnek: `a!wsearch discord bot`');
    // If no key provided, search() will automatically use free fallback search (DuckDuckGo + Wikipedia)
    await message.channel.sendTyping();

    let results;
    try {
      results = await search(query);
    } catch (error) {
      console.error('[HATA] wsearch:', error);
      return message.reply('arama servisine şu an ulaşılamıyor.');
    }
    if (!results.length) return message.reply('sonuç bulunamadı.');

    const sessionId = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 8)}`;
    const session = {
      userId: message.author.id, query, results, page: 1,
      source: process.env.SYNAPIC_API_KEY ? 'Synapic Search / DuckDuckGo + Wikipedia' : 'DuckDuckGo + Wikipedia',
      totalPages: Math.ceil(results.length / PER_PAGE), timestamp: Date.now()
    };
    sessions.set(sessionId, session);
    await message.reply({ embeds: [buildEmbed(session, 1)], components: [buildRow(sessionId, 1, session.totalPages)] });
  }
};
