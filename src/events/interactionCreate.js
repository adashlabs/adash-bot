const ping = require('../commands/ping');
const wsearch = require('../commands/wsearch');

const CATEGORY_LABELS = {
  genel: { label: 'Genel', description: 'bilgi ve yardım komutları (ping, help, arama).' },
  moderasyon: { label: 'Moderasyon', description: 'sunucu yönetim komutları (ban, kick, mute, warn, clear, lock).' }
};

function buildCategoryMessage(client, category) {
  const info = CATEGORY_LABELS[category] || { label: category, description: 'komutlar.' };
  const all = [...client.commands.values()];
  const filtered = all
    .filter((cmd) => (cmd.category || 'genel') === category)
    .sort((a, b) => a.name.localeCompare(b.name));

  const lines = [
    `**${info.label}**`,
    info.description,
    '',
    'komutlar:'
  ];

  if (filtered.length === 0) {
    lines.push('  (henüz komut yok)');
  } else {
    for (const cmd of filtered) {
      const padded = cmd.name.padEnd(10);
      lines.push(`  \`a!${padded}\` ${cmd.description}`);
    }
  }

  return lines.join('\n');
}

async function handleHelpSelect(interaction) {
  const category = interaction.values[0];
  const content = buildCategoryMessage(interaction.client, category);
  await interaction.update({ content });
}

async function handlePingRefresh(interaction) {
  const { content, components } = ping.buildOutput(interaction.client, interaction.guild.id);
  await interaction.update({ content, components });
}

async function handleWsearchButton(interaction) {
  const parts = interaction.customId.split(':');
  const action = parts[1];
  const sessionId = parts[2];

  const session = wsearch.sessions.get(sessionId);
  if (!session) {
    return interaction.reply({
      content: 'arama oturumu süresi dolmuş. lütfen `a!wsearch <sorgu>` ile yeniden arayın.',
      ephemeral: true
    });
  }

  if (interaction.user.id !== session.userId) {
    return interaction.reply({ content: 'bu buton sana ait değil.', ephemeral: true });
  }

  let newPage = session.page;
  if (action === 'next') newPage += 1;
  if (action === 'prev') newPage -= 1;

  if (newPage < 1 || newPage > session.totalPages) {
    return interaction.reply({ content: 'geçersiz sayfa.', ephemeral: true });
  }

  session.page = newPage;
  session.timestamp = Date.now();

  const embed = wsearch.buildEmbed(session, newPage);
  const row = wsearch.buildRow(sessionId, newPage, session.totalPages);

  await interaction.update({ embeds: [embed], components: [row] });
}

module.exports = {
  name: 'interactionCreate',
  async execute(interaction) {
    try {
      if (interaction.isStringSelectMenu() && interaction.customId === 'help_select') {
        return await handleHelpSelect(interaction);
      }

      if (interaction.isButton()) {
        if (interaction.customId === 'ping_refresh') {
          return await handlePingRefresh(interaction);
        }
        if (interaction.customId.startsWith('wsearch:')) {
          return await handleWsearchButton(interaction);
        }
      }
    } catch (error) {
      console.error('[HATA] interactionCreate:', error);
      try {
        if (interaction.deferred || interaction.replied) {
          await interaction.followUp({ content: 'bir hata oluştu.', ephemeral: true });
        } else {
          await interaction.reply({ content: 'bir hata oluştu.', ephemeral: true });
        }
      } catch (_) {}
    }
  }
};