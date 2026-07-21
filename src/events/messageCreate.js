const database = require('../database');

module.exports = {
  name: 'messageCreate',
  async execute(message) {
    if (message.author.bot) return;
    if (!message.guild) return;

    database.registerGuild(message.guild);
    database.registerUser(message.author);

    const prefix = database.getPrefix(message.guild.id);
    if (!message.content.startsWith(prefix)) return;

    const body = message.content.slice(prefix.length).trim();
    if (body.length === 0) return;

    const parts = body.split(/\s+/);
    const commandName = parts.shift().toLowerCase();
    const args = parts;

    const command = message.client.commands.get(commandName);
    if (!command) return;

    database.logCommand(message.guild.id, message.author.id, commandName, args.join(' '));

    try {
      await command.execute(message, args, message.client);
    } catch (error) {
      console.error(`[HATA] ${prefix}${commandName} çalıştırılırken:`, error);
      try {
        await message.reply('komut çalıştırılırken bir hata oluştu.');
      } catch (_) {}
    }
  }
};
