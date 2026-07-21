require('dotenv').config();
const { ShardingManager } = require('discord.js');

if (!process.env.DISCORD_TOKEN) {
  console.error('HATA: DISCORD_TOKEN .env dosyasında bulunamadı.');
  process.exit(1);
}

const shardArg = process.env.SHARD_COUNT;
let totalShards = 1;
if (shardArg === 'auto') {
  totalShards = 'auto';
} else if (shardArg) {
  const n = parseInt(shardArg, 10);
  if (Number.isFinite(n) && n > 0) totalShards = n;
}

const manager = new ShardingManager('./src/index.js', {
  token: process.env.DISCORD_TOKEN,
  totalShards,
  respawn: true,
  shardList: 'auto'
});

manager.on('shardCreate', (shard) => {
  console.log(`shard ${shard.id} başlatılıyor...`);
});

manager.on('shardReady', (shard) => {
  console.log(`shard ${shard.id} hazır.`);
});

manager.on('shardDisconnect', (closeEvent, shardId) => {
  console.log(`shard ${shardId} bağlantısı kesildi.`);
});

manager.on('shardReconnecting', (shardId) => {
  console.log(`shard ${shardId} yeniden bağlanıyor...`);
});

manager.on('shardRespawn', (shardId) => {
  console.log(`shard ${shardId} yeniden başlatıldı.`);
});

process.on('unhandledRejection', (error) => {
  console.error('[YAKALANMAMIŞ HATA]', error);
});

process.on('SIGINT', () => {
  console.log('\nkapatılıyor...');
  manager.shutdown().then(() => process.exit(0)).catch(() => process.exit(1));
});

process.on('SIGTERM', () => {
  manager.shutdown().then(() => process.exit(0)).catch(() => process.exit(1));
});

manager.spawn().catch((error) => {
  console.error('shard başlatma hatası:', error);
  process.exit(1);
});
