const database = require('./database');
const { transient } = require('./utils/ui');
const { normalizeWord, isTurkishDictionaryWord } = require('./dictionary');

const queues = new Map();

async function isDictionaryWord(word) {
  return isTurkishDictionaryWord(word);
}

async function rejectMessage(message, reason) {
  if (database.getConfig(message.guild.id, 'game_delete_invalid', true)) await message.delete().catch(() => null);
  await transient(message.channel, `❌ <@${message.author.id}> ${reason}`, 5500);
}

async function handleCounting(message) {
  const state = database.getGameState(message.guild.id);
  const expected = state.counting_value + 1;
  if (!/^\d+$/.test(message.content.trim()) || Number(message.content.trim()) !== expected || state.counting_user_id === message.author.id) {
    const sameUser = state.counting_user_id === message.author.id;
    const shouldReset = database.getConfig(message.guild.id, 'counting_reset_on_error', true);
    if (shouldReset) database.resetGame(message.guild.id, 'counting');
    await rejectMessage(message, sameUser
      ? `aynı kişi art arda sayamaz.${shouldReset ? " Oyun **1**'den yeniden başladı." : ` Sıradaki sayı hâlâ **${expected}**.`}`
      : `sıradaki sayı **${expected}** olmalıydı.${shouldReset ? " Oyun **1**'den yeniden başladı." : ''}`);
    return;
  }
  database.setCountingState(message.guild.id, expected, message.author.id);
  await message.react(expected % 100 === 0 ? '💯' : '✅').catch(() => null);
}

async function handleWordChain(message) {
  const word = normalizeWord(message.content);
  const minLength = database.getConfig(message.guild.id, 'word_min_length', 2);
  if (!new RegExp(`^[aâbcçdefgğhıiîjklmnoöprsştuüûvyz]{${minLength},30}$`).test(word)) {
    return rejectMessage(message, `yalnızca ${minLength}–30 harflik tek bir Türkçe kelime yazmalısın.`);
  }

  const state = database.getGameState(message.guild.id);
  if (state.word_user_id === message.author.id) return rejectMessage(message, 'aynı kişi art arda kelime yazamaz.');
  if (state.last_word && word[0] !== state.last_word.at(-1)) {
    return rejectMessage(message, `kelime **${state.last_word.at(-1).toLocaleUpperCase('tr-TR')}** harfiyle başlamalı.`);
  }
  if (database.hasUsedWord(message.guild.id, word)) return rejectMessage(message, `**${word}** daha önce kullanıldı.`);

  const valid = await isDictionaryWord(word);
  if (!valid) return rejectMessage(message, `**${word}** yerel Türkçe oyun sözlüğünde bulunamadı.`);

  database.setWordState(message.guild.id, word, message.author.id);
  await message.react('✅').catch(() => null);
}

function enqueue(guildId, operation) {
  const previous = queues.get(guildId) || Promise.resolve();
  const current = previous.catch(() => null).then(operation).finally(() => {
    if (queues.get(guildId) === current) queues.delete(guildId);
  });
  queues.set(guildId, current);
  return current;
}

async function handleGameMessage(message) {
  const settings = database.getSettings(message.guild.id);
  if (settings.counting_enabled && message.channel.id === settings.counting_channel_id) {
    await enqueue(message.guild.id, () => handleCounting(message));
    return true;
  }
  if (settings.word_chain_enabled && message.channel.id === settings.word_chain_channel_id) {
    await enqueue(message.guild.id, () => handleWordChain(message));
    return true;
  }
  return false;
}

module.exports = { handleGameMessage, normalizeWord, isDictionaryWord };
