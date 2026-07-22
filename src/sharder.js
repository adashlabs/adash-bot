require('dotenv').config();
const { ShardingManager } = require('discord.js');

if (!process.env.DISCORD_TOKEN) {
  console.error('HATA: DISCORD_TOKEN .env dosyasında bulunamadı.');
  process.exit(1);
}

const shardArg = process.env.SHARD_COUNT;
const totalShards = 1;
if (shardArg && shardArg !== '1') {
  console.warn('SQLite kalıcılığı için tek shard zorunlu. SHARD_COUNT=1 kullanılacak.');
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

let shuttingDown = false;
function shutdown(signal) {
  if (shuttingDown) return;
  shuttingDown = true;
  console.log(`\n${signal}: shard'ler kapatılıyor...`);
  for (const shard of manager.shards.values()) shard.kill();
}

process.once('SIGINT', () => shutdown('SIGINT'));
process.once('SIGTERM', () => shutdown('SIGTERM'));

manager.spawn().catch((error) => {
  console.error('shard başlatma hatası:', error);
  process.exit(1);
});
