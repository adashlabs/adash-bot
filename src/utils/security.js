const { PermissionFlagsBits } = require('discord.js');

async function resolveTarget(message, input) {
  if (!input) return null;
  const mention = String(input).match(/^<@!?(\d+)>$/);
  const idOnly = String(input).match(/^(\d{17,20})$/);
  const id = mention?.[1] || idOnly?.[1];
  if (!id) return null;

  const member = await message.guild.members.fetch(id).catch(() => null);
  const user = member?.user || await message.client.users.fetch(id).catch(() => null);
  return user ? { user, member } : null;
}

function moderationError(message, targetMember, options) {
  const {
    userPermission,
    botPermission = userPermission,
    action,
    allowSelf = false
  } = options;
  const actor = message.member;
  const bot = message.guild.members.me;

  if (!actor.permissions.has(userPermission)) {
    return 'bu işlem için gerekli yetkiye sahip değilsin.';
  }
  if (!bot.permissions.has(botPermission)) {
    return 'bu işlemi yapabilmem için gerekli Discord yetkisine sahip değilim.';
  }
  if (!targetMember) return null;
  if (!allowSelf && targetMember.id === actor.id) return 'bu işlemi kendine uygulayamazsın.';
  if (targetMember.id === message.guild.ownerId) return 'sunucu sahibine moderasyon işlemi uygulanamaz.';
  if (targetMember.id === bot.id) return 'bu işlemi kendime uygulayamam.';

  const actorIsOwner = actor.id === message.guild.ownerId;
  if (!actorIsOwner && actor.roles.highest.comparePositionTo(targetMember.roles.highest) <= 0) {
    return 'kendi en yüksek rolüne eşit veya üstteki bir üyeye işlem uygulayamazsın.';
  }
  if (bot.roles.highest.comparePositionTo(targetMember.roles.highest) <= 0) {
    return 'hedef üyenin rolü botun en yüksek rolüne eşit veya üstte.';
  }
  if (action && targetMember[action] === false) {
    return 'Discord rol hiyerarşisi bu işlemi uygulamama izin vermiyor.';
  }
  return null;
}

function canManageRole(guild, role) {
  const bot = guild.members.me;
  if (!bot.permissions.has(PermissionFlagsBits.ManageRoles)) {
    return { ok: false, reason: 'botta `Rolleri Yönet` yetkisi yok.' };
  }
  if (!role || role.id === guild.id || role.managed) {
    return { ok: false, reason: 'bu rol otomatik olarak verilemez.' };
  }
  if (bot.roles.highest.comparePositionTo(role) <= 0) {
    return { ok: false, reason: 'otomatik rol, botun en yüksek rolünün altında olmalı.' };
  }
  return { ok: true };
}

function hasSetupPermission(member) {
  return member.permissions.has(PermissionFlagsBits.ManageGuild);
}

module.exports = {
  PermissionFlagsBits,
  resolveTarget,
  moderationError,
  canManageRole,
  hasSetupPermission
};
