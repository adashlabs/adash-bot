const { EmbedBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { lookupTdk, isTurkishDictionaryWord, normalizeWord } = require('../dictionary');
const { COLORS, truncate } = require('../utils/ui');

function meaningText(means) {
  if (!Array.isArray(means) || !means.length) return 'Anlam bilgisi bulunamadı.';
  return means.slice(0, 6).map((meaning, index) => {
    const properties = Array.isArray(meaning.ozelliklerListe) ? meaning.ozelliklerListe.map((item) => item.tam_adi || item.ozellik_adi).filter(Boolean).join(', ') : '';
    return `**${index + 1}.** ${properties ? `*${properties}* · ` : ''}${truncate(meaning.anlam || String(meaning), 550)}`;
  }).join('\n');
}

function exampleText(means) {
  if (!Array.isArray(means)) return null;
  const examples = means.flatMap((meaning) => meaning.orneklerListe || []).slice(0, 3);
  if (!examples.length) return null;
  return examples.map((example) => `• “${truncate(example.ornek || '', 350)}”${example.yazar?.[0]?.tam_adi ? ` — *${example.yazar[0].tam_adi}*` : ''}`).join('\n');
}

function sourceText(items, keys, limit = 5) {
  if (!Array.isArray(items) || !items.length) return null;
  const values = items.slice(0, limit).map((item) => {
    if (typeof item === 'string') return item;
    if (!item || typeof item !== 'object') return null;
    return keys.map((key) => item[key]).find((value) => typeof value === 'string' && value.trim()) || null;
  }).filter(Boolean);
  return values.length ? values.map((value) => `• ${truncate(value, 300)}`).join('\n') : null;
}

module.exports = {
  name: 'tdk', aliases: ['sözlük', 'sozluk'], category: 'genel',
  description: 'TDK’nin güncel, deyim, terim ve etimoloji kaynaklarında ayrıntılı kelime arar.', cooldown: 4,
  async execute(message, args) {
    const word = normalizeWord(args.join(' '));
    if (!word || word.length > 80) return message.reply('aranacak bir kelime belirt. Örnek: `a!tdk kitap`');
    await message.channel.sendTyping();
    const entry = await lookupTdk(word);
    if (!entry) {
      return message.reply(isTurkishDictionaryWord(word)
        ? `✅ **${word}** yerel Türkçe sözlükte doğrulandı; TDK ayrıntı servisi şu an yanıt vermedi. Resmî sayfa: <https://sozluk.gov.tr/?ara=${encodeURIComponent(word)}>`
        : `❌ **${word}** için sözlük sonucu bulunamadı.`);
    }
    const embed = new EmbedBuilder().setColor(COLORS.primary).setTitle(`📖 ${entry.word}`)
      .setURL(`https://sozluk.gov.tr/?ara=${encodeURIComponent(word)}`)
      .setDescription(meaningText(entry.means))
      .addFields({ name: 'Köken / Lisan', value: truncate(entry.lisan || 'Türkçe veya bilgi belirtilmemiş', 1024), inline: true })
      .setFooter({ text: 'TDK tüm sözlükler · tdk-all-api' }).setTimestamp();
    const examples = exampleText(entry.means);
    if (examples) embed.addFields({ name: 'Kullanım Örnekleri', value: truncate(examples, 1024) });
    if (Array.isArray(entry.compounds) && entry.compounds.length) embed.addFields({ name: 'Birleşik Kelimeler', value: truncate(entry.compounds.slice(0, 15).join(', '), 1024) });
    const proverbs = sourceText(entry.proverbs, ['soz', 'madde', 'sozum', 'anlam']);
    if (proverbs) embed.addFields({ name: 'Atasözleri ve Deyimler', value: truncate(proverbs, 1024) });
    const etymology = sourceText(entry.etymological, ['etimoloji', 'anlam', 'madde']);
    if (etymology) embed.addFields({ name: 'Etimoloji', value: truncate(etymology, 1024) });
    const compilation = sourceText(entry.compilation, ['madde', 'anlam', 'tanim']);
    if (compilation) embed.addFields({ name: 'Derleme Sözlüğü', value: truncate(compilation, 1024) });
    const terms = sourceText(entry.glossaryOfScienceAndArtTerms, ['madde', 'anlam', 'tanim']);
    if (terms) embed.addFields({ name: 'Bilim ve Sanat Terimleri', value: truncate(terms, 1024) });
    const western = sourceText(entry.westOpposite, ['madde', 'anlam', 'tanim']);
    if (western) embed.addFields({ name: 'Batı Kökenli Kelimeler', value: truncate(western, 1024) });
    const guide = sourceText(entry.guide, ['madde', 'anlam', 'tanim']);
    if (guide) embed.addFields({ name: 'Yabancı Sözlere Karşılıklar', value: truncate(guide, 1024) });
    const row = new ActionRowBuilder().addComponents(new ButtonBuilder().setStyle(ButtonStyle.Link)
      .setURL(`https://sozluk.gov.tr/?ara=${encodeURIComponent(word)}`).setLabel('TDK’de Aç'));
    await message.reply({ embeds: [embed], components: [row] });
  }
};
