const database = require('../database');

module.exports = {
  name: 'guildCreate',
  execute(guild) {
    database.registerGuild(guild);
  }
};
