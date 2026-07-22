const { ActivityType } = require('discord.js');
const database = require('../database');
const { restoreGiveaways } = require('../giveaways');
const { slashCommands } = require('../slash');

module.exports = {
  name: 'clientReady',
  once: true,
  async execute(client) {
    console.log(`adash | ${client.user.tag} olarak giriş yapıldı`);
    console.log(`${client.guilds.cache.size} sunucu | ${client.users.cache.size} kullanıcı`);
    console.log(`Veritabanı: ${database.path}`);

    client.guilds.cache.forEach((guild) => database.registerGuild(guild));
    const isPrimaryShard = !client.shard || client.shard.ids.includes(0);
    if (isPrimaryShard) {
      await client.application.commands.set(slashCommands);
      await restoreGiveaways(client);
      console.log(`${slashCommands.length} global slash komutu kaydedildi.`);
    }

    client.user.setActivity('/yardim · a!help', { type: ActivityType.Watching });
    client.user.setStatus('online');
  }
};
