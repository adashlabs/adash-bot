const database = require('../database');
const games = require('../games');
const { transient, errorEmbed } = require('../utils/ui');
const { handleAiMention } = require('../ai');

module.exports = {
  name: 'messageCreate',
  async execute(message) {
    if (message.author.bot || !message.guild) return;

    database.registerGuild(message.guild);
    database.registerUser(message.author);
    const settings = database.getSettings(message.guild.id);
    const prefix = database.getPrefix(message.guild.id);

    if (!message.content.startsWith(prefix)) {
      if (settings.ai_channel_id && message.channel.id === settings.ai_channel_id) {
        if (await handleAiMention(message)) return;
      }
      if (message.mentions.users.has(message.client.user.id)) {
        if (await handleAiMention(message)) return;
      }
      await games.handleGameMessage(message);
      return;
    }

    const body = message.content.slice(prefix.length).trim();
    if (!body) return;
    const parts = body.split(/\s+/);
    const requestedName = parts.shift().toLocaleLowerCase('tr-TR');
    const canonicalName = message.client.aliases.get(requestedName) || requestedName;
    const command = message.client.commands.get(canonicalName);
    if (!command) return;

    const cooldownSeconds = Number(command.cooldown || 2);
    const cooldownKey = `${message.guild.id}:${message.author.id}:${canonicalName}`;
    const expiresAt = message.client.cooldowns.get(cooldownKey) || 0;
    if (expiresAt > Date.now()) {
      const remaining = Math.ceil((expiresAt - Date.now()) / 1000);
      await transient(message.channel, `<@${message.author.id}> bu komutu yeniden kullanmak için **${remaining} saniye** bekle.`, 3000);
      return;
    }
    message.client.cooldowns.set(cooldownKey, Date.now() + cooldownSeconds * 1000);
    setTimeout(() => message.client.cooldowns.delete(cooldownKey), cooldownSeconds * 1000).unref?.();

    database.logCommand(message.guild.id, message.author.id, canonicalName, parts.join(' '));
    try {
      await command.execute(message, parts, message.client);
    } catch (error) {
      console.error(`[HATA] ${prefix}${canonicalName}:`, error);
      const embed = errorEmbed('Komut Hatası', 'Komut çalıştırılırken beklenmeyen bir hata oluştu. Yetkileri ve rol sırasını kontrol et.', message.author);
      await message.reply({ embeds: [embed] }).catch(() => null);
    }
  }
};
