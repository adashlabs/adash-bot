# 🤖 Discord Bot Development Skill Specification (`discord-bot`)

This skill defines the authoritative architecture, UI/UX guidelines, component limits, security invariants, and interaction design patterns for building high-quality, production-grade **Discord.js v14** bots.

---

## 📋 1. Core Architectural Invariants

### 1.1 Zero Plain-Text Fallbacks
Every output, error, warning, or status message returned to users MUST be formatted using standardized **Rich Embeds** with visual badge emojis and ISO timestamps. Plain text responses are strictly prohibited for system notices.

### 1.2 Discord API Component Limits (HARD CONSTRAINTS)
- **ActionRows Per Message:** Maximum **5 ActionRows** (`components.length <= 5`). Attempting to send 6+ rows throws `DiscordAPIError[50035]: components: Must be 5 or fewer in length`.
- **Select Menus:** Maximum **25 options** per `StringSelectMenuBuilder`. Every Select Menu (`StringSelectMenu`, `ChannelSelectMenu`, `RoleSelectMenu`, `UserSelectMenu`) MUST occupy its own dedicated `ActionRowBuilder`.
- **Modals:** Maximum **5 TextInput components** per `ModalBuilder` (exactly 1 `TextInputBuilder` per `ActionRowBuilder`).
- **Embed Limits:**
  - `Title`: Maximum 256 characters.
  - `Description`: Maximum 4000 characters.
  - `Field Name`: Maximum 256 characters.
  - `Field Value`: Maximum 1024 characters.
  - `Footer Text`: Maximum 2048 characters.
  - `Total Embed Payload`: Maximum 6000 characters per message.

---

## 🎨 2. UI/UX & Color Palette Standards

### 2.1 Standardized Embed Palette
```js
const COLORS = {
  primary: 0x5865F2, // Discord Blurple (General info, search results, profile cards)
  success: 0x57F287, // Green (Completed actions, unban, ticket opened, giveaway entry)
  warning: 0xFEE75C, // Yellow (Warnings, timeout, confirmations, appeals)
  danger:  0xED4245, // Red (Bans, kicks, channel lock, closed tickets/giveaways)
  info:    0x3498DB  // Blue (System status, ping, statistics, help guides)
};
```

### 2.2 Standard Response Helpers (`src/utils/ui.js`)
```js
function successEmbed(title, description, user) {
  return createEmbed({ type: 'success', title: `✅ ${title}`, description, user });
}

function errorEmbed(title, description, user) {
  return createEmbed({ type: 'danger', title: `❌ ${title}`, description, user });
}

function warningEmbed(title, description, user) {
  return createEmbed({ type: 'warning', title: `⚠️ ${title}`, description, user });
}

function infoEmbed(title, description, user) {
  return createEmbed({ type: 'info', title: `ℹ️ ${title}`, description, user });
}
```

---

## 🎫 3. Advanced Ticket Systems Architecture

### 3.1 Category (`GuildCategory`) vs. Text Channel (`GuildText`)
- **Ticket Parent:** Must strictly be a `GuildCategory` (type 4). Ticket text channels (`#talep-normal-ahmet`) are created *inside* this parent category.
- **Validation Rule:** Never allow admins to save a standard text channel as `ticket_category_id`.
```js
if (channel.type !== ChannelType.GuildCategory && channel.type !== 4) {
  return errorEmbed('Geçersiz Kategori', 'Ticket kategorisi bir metin kanalı değil, Discord Kategori başlığı olmalıdır.');
}
```

### 3.2 Single-Step Setup Wizard (`ticketsetup`)
Provide both a 1-step command (`a!ticketsetup "DESTEK" #panel #log @destek`) and an interactive setup wizard:
- `ChannelSelectMenuBuilder` for `kategori` (`ChannelType.GuildCategory`).
- `ChannelSelectMenuBuilder` for `panel_kanali` (`ChannelType.GuildText`).
- `ChannelSelectMenuBuilder` for `log_kanali` (`ChannelType.GuildText`).
- `RoleSelectMenuBuilder` for `destek_rolu`.
- Action Buttons: `📝 Metinleri Düzenle` (Modal), `🚀 Paneli Gönder` (Deploy Button).

### 3.3 Channel Control Panel (Opened Ticket GUI)
Every opened ticket channel MUST include an interactive 2-row control bar:
- **Row 1:**
  - 🙋 `ticket_claim`: Claims the ticket for the staff member.
  - ➕ `ticket_add_btn`: Opens Modal to type Member Tag/ID to add to channel.
  - ✏️ `ticket_rename_btn`: Opens Modal to rename the ticket channel.
- **Row 2:**
  - 📌 `ticket_status`: Toggles ticket status (`🟢 Açık` → `🟡 İnceleniyor` → `🔵 Kullanıcı Yanıtı Bekleniyor`).
  - 🔒 `ticket_close_btn`: Opens Modal to enter close reason, logs transcript, and deletes channel cleanly.

---

## 🎉 4. Giveaway Systems Architecture

### 4.1 Real-Time Participant Win Probability
Calculate and display the exact win chance per participant in both the embed and ephemeral entry responses:
$$\text{Win Probability} = \min\left(100, \frac{\text{Winner Count}}{\text{Total Entries}} \times 100\right)\%$$
```js
function calculateChance(entriesCount, winnerCount) {
  if (entriesCount <= 0) return '0%';
  const chance = Math.min(100, (winnerCount / entriesCount) * 100);
  const ratio = Math.ceil(entriesCount / Math.min(entriesCount, winnerCount));
  return `%${chance.toFixed(1)} (Yaklaşık 1 / ${ratio})`;
}
```

### 4.2 Entry Requirements & Persistence
- **Account Age Check:** Validate `(Date.now() - user.createdTimestamp) / 86400000 >= minAccountAgeDays`.
- **Role Requirement Check:** Validate `member.roles.cache.has(requiredRoleId)`.
- **Persistence & Timer Restoration:** Store giveaways in SQLite and auto-resume timers via `scheduleGiveaway(client, draft)` upon bot restarts.

---

## 🛡️ 5. Moderation & Security Architecture

### 5.1 Permission & Hierarchy Rules
- **Server Owner Protection:** Server owners cannot be targeted by moderation actions.
- **Role Hierarchy:**
  ```js
  if (actor.id !== guild.ownerId && actor.roles.highest.comparePositionTo(targetMember.roles.highest) <= 0) {
    return 'Kendi en yüksek rolünüze eşit veya üstteki bir üyeye işlem uygulayamazsınız.';
  }
  if (bot.roles.highest.comparePositionTo(targetMember.roles.highest) <= 0) {
    return 'Hedef üyenin rolü botun en yüksek rolüne eşit veya üstte.';
  }
  ```
- **Un-cached Members:** Allow banning users by Snowflake ID even if they are no longer in the guild.

### 5.2 Direct vs. Confirmed Moderation
- **Instant Actions:** Direct operations like `clear` (`a!sil`) must execute immediately without confirmation prompts.
- **Destructive Actions:** Actions like `ban`, `kick`, `mute`, `unban`, `lock`, `slowmode` must present a clean button confirmation dialog (`✅ Evet, uygula` / `✖️ Hayır, iptal`).

### 5.3 Mention Sanitization
Always sanitize mentions on user-generated or AI outputs to prevent `@everyone` or `@here` ping abuse:
```js
allowedMentions: { parse: [] }
```

---

## ⚡ 6. Interaction Routing & Async Safety

### 6.1 Safe Interaction Response Pattern
Always handle interaction deferrals safely before 3-second deadlines:
```js
async function respond(interaction, payload, ephemeral = false) {
  if (interaction.deferred || interaction.replied) {
    return interaction.followUp({ ...payload, flags: ephemeral ? 64 : undefined });
  }
  return interaction.reply({ ...payload, flags: ephemeral ? 64 : undefined });
}
```

### 6.2 Setup Access Permission Guard
```js
function hasSetupAccess(interaction) {
  if (!interaction.guild || !interaction.member) return false;
  if (interaction.user.id === interaction.guild.ownerId) return true;
  const hasPerm = interaction.member.permissions.has(PermissionFlagsBits.ManageGuild)
    || interaction.member.permissions.has(PermissionFlagsBits.Administrator);
  if (!hasPerm) return false;
  const parts = interaction.customId.split(':');
  const targetGuildId = parts.length > 1 ? parts.at(-1) : null;
  if (targetGuildId && /^\d{17,20}$/.test(targetGuildId) && targetGuildId !== interaction.guild.id) {
    return false;
  }
  return true;
}
```
