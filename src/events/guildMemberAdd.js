const database = require('../database');
const { canManageRole } = require('../utils/security');
const { sendMemberGreeting } = require('../utils/memberGreeting');

module.exports = {
  name: 'guildMemberAdd',
  async execute(member) {
    database.registerGuild(member.guild);
    database.registerUser(member.user);
    const settings = database.getSettings(member.guild.id);

    if (settings.autorole_id && !member.user.bot) {
      const role = await member.guild.roles.fetch(settings.autorole_id).catch(() => null);
      const result = canManageRole(member.guild, role);
      if (result.ok) {
        await member.roles.add(role, 'adash otomatik rol').catch((error) => {
          console.error('[HATA] otomatik rol:', error);
        });
      } else {
        console.warn(`[UYARI] ${member.guild.name} otomatik rol: ${result.reason}`);
      }
    }

    await sendMemberGreeting(member, 'welcome');
  }
};
