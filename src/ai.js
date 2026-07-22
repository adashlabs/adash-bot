const database = require('./database');
const { truncate } = require('./utils/ui');

const histories = new Map();
const cooldowns = new Map();
const HISTORY_LIMIT = 12;
const HISTORY_TTL = 30 * 60 * 1000;

const DEFAULT_SYSTEM_PROMPT = [
  'Sen Adash adlı sıcak, samimi ve doğal konuşan bir Discord sohbet arkadaşısın.',
  'Temel amacın bilgi dökmek değil; kullanıcıyla gerçek bir sohbet kurmak, onun üslubuna uyum sağlamak ve gerektiğinde sohbeti nazikçe ilerletmektir.',
  'Türkçe konuş; kısa, canlı ve insani yanıtlar ver. Kullanıcı bir şey paylaşırsa ilgiyle karşılık ver, uygun olduğunda doğal takip soruları sor.',
  'Bilmediğin bir konuda uydurma. Toplu etiket, rol etiketi veya zararlı içerik üretme.'
].join(' ');

function configured() {
  return Boolean(process.env.OPENAI_BASE_URL && process.env.OPENAI_MODEL);
}

function endpoint() {
  const base = process.env.OPENAI_BASE_URL.replace(/\/+$/, '');
  return base.endsWith('/chat/completions') ? base : `${base}/chat/completions`;
}

function safeOutput(value) {
  return String(value)
    .replace(/@everyone/gi, '@\u200beveryone')
    .replace(/@here/gi, '@\u200bhere')
    .replace(/<@&/g, '<@\u200b&')
    .replace(/<@!?\d+>/g, (mention) => mention.replace('@', '@\u200b'));
}

function chunks(value, max = 1900) {
  const result = [];
  let rest = value.trim();
  while (rest.length > max) {
    let index = rest.lastIndexOf('\n', max);
    if (index < max / 2) index = rest.lastIndexOf(' ', max);
    if (index < max / 2) index = max;
    result.push(rest.slice(0, index));
    rest = rest.slice(index).trimStart();
  }
  if (rest) result.push(rest);
  return result;
}

function cleanPrompt(message) {
  return truncate(message.content
    .replace(new RegExp(`<@!?${message.client.user.id}>`, 'g'), '')
    .replace(/<@&\d+>/g, '[rol]')
    .replace(/@everyone|@here/gi, '[toplu etiket]')
    .trim(), 4000);
}

async function requestCompletion(messages) {
  const response = await fetch(endpoint(), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${process.env.OPENAI_API_KEY}` },
    signal: AbortSignal.timeout(Number(process.env.OPENAI_TIMEOUT_MS) || 45_000),
    body: JSON.stringify({
      model: process.env.OPENAI_MODEL,
      messages,
      temperature: Number(process.env.OPENAI_TEMPERATURE) || 0.7,
      max_tokens: Number(process.env.OPENAI_MAX_TOKENS) || 900
    })
  });
  if (!response.ok) throw new Error(`OpenAI uyumlu servis HTTP ${response.status}: ${truncate(await response.text(), 300)}`);
  const data = await response.json();
  const content = data.choices?.[0]?.message?.content;
  if (typeof content !== 'string' || !content.trim()) throw new Error('AI servisi boş yanıt döndürdü.');
  return content.trim();
}

async function handleAiMention(message) {
  if (!configured()) {
    await message.reply({ content: 'Yapay zekâ henüz yapılandırılmamış. `OPENAI_BASE_URL`, `OPENAI_API_KEY` ve `OPENAI_MODEL` gerekli.', allowedMentions: { parse: [], repliedUser: false } });
    return true;
  }
  if (!database.getConfig(message.guild.id, 'ai_enabled', true)) return false;
  const prompt = cleanPrompt(message);
  if (!prompt) {
    await message.reply({ content: 'Bana bir soru da yazmalısın.', allowedMentions: { parse: [], repliedUser: false } });
    return true;
  }
  const cooldownKey = `${message.guild.id}:${message.author.id}`;
  const availableAt = cooldowns.get(cooldownKey) || 0;
  if (availableAt > Date.now()) {
    await message.reply({ content: `Yeni soru için ${Math.ceil((availableAt - Date.now()) / 1000)} saniye bekle.`, allowedMentions: { parse: [], repliedUser: false } });
    return true;
  }
  cooldowns.set(cooldownKey, Date.now() + 10_000);
  setTimeout(() => cooldowns.delete(cooldownKey), 10_000).unref();
  const historyKey = `${message.guild.id}:${message.channel.id}`;
  const stored = histories.get(historyKey);
  const history = stored && Date.now() - stored.updatedAt < HISTORY_TTL ? stored.messages : [];
  const system = database.getConfig(message.guild.id, 'ai_system_prompt', process.env.OPENAI_SYSTEM_PROMPT || DEFAULT_SYSTEM_PROMPT);
  const messages = [{ role: 'system', content: system }, ...history, { role: 'user', content: `${message.author.username}: ${prompt}` }];
  await message.channel.sendTyping();
  const typing = setInterval(() => message.channel.sendTyping().catch(() => null), 7000);
  typing.unref();
  try {
    const raw = await requestCompletion(messages);
    const answer = safeOutput(raw);
    const parts = chunks(answer);
    await message.reply({ content: parts.shift(), allowedMentions: { parse: [], repliedUser: false } });
    for (const part of parts) await message.channel.send({ content: part, allowedMentions: { parse: [] } });
    const updated = [...history, { role: 'user', content: `${message.author.username}: ${prompt}` }, { role: 'assistant', content: answer }].slice(-HISTORY_LIMIT);
    histories.set(historyKey, { messages: updated, updatedAt: Date.now() });
    if (histories.size > 500) histories.delete(histories.keys().next().value);
  } catch (error) {
    console.error('[HATA] yapay zekâ:', error);
    await message.reply({ content: 'Yapay zekâ servisine şu an ulaşılamıyor. URL, model ve API anahtarını kontrol et.', allowedMentions: { parse: [], repliedUser: false } }).catch(() => null);
  } finally {
    clearInterval(typing);
  }
  return true;
}

module.exports = { handleAiMention, requestCompletion, safeOutput, chunks, configured, DEFAULT_SYSTEM_PROMPT };
