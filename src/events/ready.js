const { ActivityType } = require('discord.js');
const database = require('../database');

module.exports = {
  name: 'clientReady',
  once: true,
  execute(client) {
    console.log(` adash | ${client.user.tag} olarak giriş yapıldı`);
    console.log(` ${client.guilds.cache.size} sunucu | ${client.users.cache.size} kullanıcı`);
    console.log(`  Veritabanı: ${database.path}`);

    client.guilds.cache.forEach((guild) => database.registerGuild(guild));

    client.user.setActivity('a!help | adash', { type: ActivityType.Watching });
    client.user.setStatus('online');
  }
};
