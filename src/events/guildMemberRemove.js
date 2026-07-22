const { sendMemberGreeting } = require('../utils/memberGreeting');

module.exports = {
  name: 'guildMemberRemove',
  async execute(member) {
    await sendMemberGreeting(member, 'farewell');
  }
};
