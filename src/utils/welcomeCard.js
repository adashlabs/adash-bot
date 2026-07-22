const { createCanvas, loadImage } = require('@napi-rs/canvas');

function fitText(ctx, text, maxWidth, startSize = 58, minSize = 28) {
  let size = startSize;
  while (size > minSize) {
    ctx.font = `700 ${size}px Arial`;
    if (ctx.measureText(text).width <= maxWidth) return size;
    size -= 2;
  }
  ctx.font = `700 ${minSize}px Arial`;
  return minSize;
}

function ellipsis(ctx, value, maxWidth) {
  const text = String(value || '');
  if (ctx.measureText(text).width <= maxWidth) return text;
  let result = text;
  while (result.length > 1 && ctx.measureText(`${result}…`).width > maxWidth) result = result.slice(0, -1);
  return `${result.trimEnd()}…`;
}

async function createMemberCard(member, type = 'welcome') {
  const canvas = createCanvas(1200, 420);
  const ctx = canvas.getContext('2d');
  const welcome = type === 'welcome';
  const accent = welcome ? '#57F287' : '#ED4245';

  const gradient = ctx.createLinearGradient(0, 0, 1200, 420);
  gradient.addColorStop(0, '#111827');
  gradient.addColorStop(0.55, '#202938');
  gradient.addColorStop(1, welcome ? '#12372a' : '#421f29');
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, 1200, 420);

  ctx.globalAlpha = 0.08;
  for (let x = -100; x < 1300; x += 90) {
    ctx.fillStyle = accent;
    ctx.beginPath();
    ctx.arc(x, 50 + ((x / 90) % 2) * 250, 160, 0, Math.PI * 2);
    ctx.fill();
  }
  ctx.globalAlpha = 1;

  ctx.fillStyle = 'rgba(0,0,0,0.28)';
  ctx.beginPath();
  ctx.roundRect(34, 34, 1132, 352, 30);
  ctx.fill();

  const avatarUrl = member.user.displayAvatarURL({ extension: 'png', size: 256 });
  const avatar = await loadImage(avatarUrl);
  ctx.save();
  ctx.beginPath();
  ctx.arc(214, 210, 132, 0, Math.PI * 2);
  ctx.clip();
  ctx.drawImage(avatar, 82, 78, 264, 264);
  ctx.restore();

  ctx.strokeStyle = accent;
  ctx.lineWidth = 10;
  ctx.beginPath();
  ctx.arc(214, 210, 138, 0, Math.PI * 2);
  ctx.stroke();

  const textX = 405;
  const textWidth = 720;
  ctx.textBaseline = 'alphabetic';

  const headline = welcome ? 'HOŞ GELDİN' : 'GÖRÜŞMEK ÜZERE';
  ctx.fillStyle = accent;
  ctx.font = '700 30px Arial';
  ctx.fillText(headline, textX, 130);

  const displayName = member.user.globalName || member.user.username || 'Yeni üye';
  fitText(ctx, displayName, textWidth);
  ctx.fillStyle = '#FFFFFF';
  ctx.fillText(ellipsis(ctx, displayName, textWidth), textX, 205);

  const guildName = member.guild.name || 'Discord sunucusu';
  fitText(ctx, guildName, textWidth, 38, 24);
  ctx.fillStyle = '#CBD5E1';
  ctx.fillText(ellipsis(ctx, guildName, textWidth), textX, 265);

  ctx.fillStyle = '#94A3B8';
  ctx.font = '500 25px Arial';
  const detail = welcome
    ? `Seninle birlikte ${member.guild.memberCount || 0} üyeyiz.`
    : `Topluluğumuzda ${member.guild.memberCount || 0} üye kaldı.`;
  ctx.fillText(ellipsis(ctx, detail, textWidth), textX, 320);

  return canvas.encode('png');
}

module.exports = { createMemberCard };
