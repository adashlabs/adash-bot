async function resolveUser(client, idOrMention) {
  if (!idOrMention) return null;
  const str = String(idOrMention);
  const mention = str.match(/^<@!?(\d+)>$/);
  const idOnly = str.match(/^(\d{17,20})$/);
  const id = mention?.[1] || idOnly?.[1];
  if (!id) return null;
  try {
    return await client.users.fetch(id);
  } catch (e) {
    return null;
  }
}

function parseDuration(input) {
  if (!input) return null;
  const m = String(input).trim().match(/^(\d+)\s*([smhd])$/i);
  if (!m) return null;
  const n = parseInt(m[1], 10);
  if (!Number.isFinite(n) || n <= 0) return null;
  const unit = m[2].toLowerCase();
  switch (unit) {
    case 's': return n * 1000;
    case 'm': return n * 60 * 1000;
    case 'h': return n * 60 * 60 * 1000;
    case 'd': return n * 24 * 60 * 60 * 1000;
  }
  return null;
}

function formatDuration(ms) {
  const totalSec = Math.floor(ms / 1000);
  const d = Math.floor(totalSec / 86400);
  const h = Math.floor(totalSec / 3600) % 24;
  const m = Math.floor(totalSec / 60) % 60;
  const s = totalSec % 60;
  if (d > 0) return `${d}g ${h}s`;
  if (h > 0) return `${h}s ${m}dk`;
  if (m > 0) return `${m}dk ${s}sn`;
  return `${s}sn`;
}

module.exports = { resolveUser, parseDuration, formatDuration };
