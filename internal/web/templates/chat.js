// ============================================================
// chat.js — полная исправленная версия
// ФИКС 1: аватарки + цветные никнеймы в чате и боковой панели
// ФИКС 2: kick_error только через toast, не перекрывает модалку
//          + убрали преждевременный toast до ответа сервера
// ФИКС 3: большие сообщения не ломают layout (word-break, max-height)
// ФИКС 4: архитектура — createRoomLi, createMessageEl вынесены,
//          switch/case вместо else-if цепочки, textContent вместо innerHTML (XSS)
// ============================================================

const chatBox          = document.getElementById("chat-box");
const chatInput        = document.getElementById("chat-input");
const chatSendBtn      = document.getElementById("chat-send-btn");
const typingIndicator  = document.getElementById("typing-indicator");
const userListEl       = document.getElementById("user-list");

const inviteModal      = document.getElementById("view-invite");
const inviteText       = document.getElementById("invite-text");
const inviteAcceptBtn  = document.getElementById("invite-accept");
const inviteDeclineBtn = document.getElementById("invite-decline");

const publicChannelsEl  = document.getElementById("public-channels");
const privateChannelsEl = document.getElementById("private-channels");
const createRoomBtn     = document.getElementById("create-room-btn");
const userSearch        = document.getElementById("user-search");
const chatHeader        = document.getElementById("chat-header");

let socket        = null;
let typingTimeout = null;
let currentRoom   = 1;

window.currentUsers   = [];
window.myPrivateRooms = [];

const messagesByRoom = {};

// ── ФИКС 1: Детерминированные цвета по никнейму ──────────────────────────────
const NICK_COLORS = [
  "#e07b54","#d4a843","#7bbf6a","#4fb8c2","#6a7fd4",
  "#a86ad4","#d46aa0","#c45d5d","#5dac8b","#8a7d5d",
  "#5d8ab5","#b55d7e","#7d9c3a","#c4834a"
];

function nickColor(name) {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) & 0xffffffff;
  return NICK_COLORS[Math.abs(h) % NICK_COLORS.length];
}

function initials(name) {
  return name.slice(0, 2).toUpperCase();
}

// ── ФИКС 1: Построение элемента сообщения с аватаркой ───────────────────────
function createMessageEl(msg) {
  const sender  = msg.sender || msg.username || "Anon";
  const content = msg.content || msg.text || "";
  const time    = msg.created_at
    ? (typeof msg.created_at === "string"
        ? msg.created_at.slice(11, 16)   // из ISO берём HH:MM
        : new Date(msg.created_at).toLocaleTimeString([], {hour:"2-digit", minute:"2-digit"}))
    : new Date().toLocaleTimeString([], {hour:"2-digit", minute:"2-digit"});
  const color = nickColor(sender);
  const isMe  = sender === document.body.dataset.username;

  const item = document.createElement("div");
  item.className = "chat-message" + (isMe ? " chat-message-mine" : "");

  // Аватарка
  const avatar = document.createElement("div");
  avatar.className = "chat-avatar";
  avatar.textContent = initials(sender);
  avatar.style.background = color;
  avatar.title = sender;

  // Тело
  const body = document.createElement("div");
  body.className = "chat-message-body";

  const nick = document.createElement("span");
  nick.className = "chat-username";
  nick.textContent = sender;
  nick.style.color = color;

  // ФИКС 3: textContent — нет XSS, word-break задан в CSS
  const text = document.createElement("span");
  text.className = "chat-text";
  text.textContent = content;

  const ts = document.createElement("span");
  ts.className = "chat-time";
  ts.textContent = time;

  body.appendChild(nick);
  body.appendChild(text);
  body.appendChild(ts);

  if (isMe) {
    item.appendChild(body);
    item.appendChild(avatar);
  } else {
    item.appendChild(avatar);
    item.appendChild(body);
  }

  return item;
}

// ── Рендер одного сообщения ──────────────────────────────────────────────────
function renderMessage(msg) {
  const roomId = msg.room_id;
  if (!messagesByRoom[roomId]) messagesByRoom[roomId] = [];
  messagesByRoom[roomId].push(msg);

  if (roomId === currentRoom) {
    chatBox.appendChild(createMessageEl(msg));
    chatBox.scrollTop = chatBox.scrollHeight;
  }
}

// ── ФИКС 1: Список участников с аватарками и цветными именами ───────────────
function renderUserList(users) {
  if (!userListEl) return;
  userListEl.innerHTML = "";
  users.forEach(function(u) {
    const li = document.createElement("li");
    li.className = "user user-item " + u.status;
    li.dataset.username = u.name;

    const color = nickColor(u.name);

    const av = document.createElement("span");
    av.className = "user-avatar";
    av.textContent = initials(u.name);
    av.style.background = color;

    const dot = document.createElement("span");
    dot.className = "status-dot";

    const nameEl = document.createElement("span");
    nameEl.className = "user-name";
    nameEl.textContent = u.name;
    nameEl.style.color = color;

    li.appendChild(av);
    li.appendChild(dot);
    li.appendChild(nameEl);
    userListEl.appendChild(li);
  });
}

// ── ФИКС 4: createRoomLi вынесен — DRY ──────────────────────────────────────
function createRoomLi(roomId, roomName) {
  const li = document.createElement("li");
  li.textContent = "# " + roomName;
  li.dataset.room = roomId;
  if (!messagesByRoom[roomId]) messagesByRoom[roomId] = [];
  li.addEventListener("click", function() {
    switchRoom(roomId, roomName);
    document.querySelectorAll(".chat-channels li").forEach(l => l.classList.remove("active"));
    li.classList.add("active");
  });
  return li;
}

// ────────────────────────────────────────────────────────────────────────────

function sendMessage() {
  const text = (chatInput.value || "").trim();
  if (!text) return;
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify({ type: "message", room_id: currentRoom, content: text }));
  chatInput.value = "";
}

function handleKeyPress(e) {
  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault();
    sendMessage();
  } else {
    sendTyping();
  }
}

function sendTyping() {
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify({ type: "typing", room_id: currentRoom }));
}

function showTyping(user) {
  typingIndicator.textContent = user + " печатает...";
  clearTimeout(typingTimeout);
  typingTimeout = setTimeout(() => { typingIndicator.textContent = ""; }, 3000);
}

function switchRoom(roomId, roomName) {
  currentRoom = parseInt(roomId, 10);
  chatHeader.textContent = "# " + (currentRoom === 1 ? "общий-чат" : roomName);
  chatBox.innerHTML = "";
  (messagesByRoom[currentRoom] || []).forEach(msg => chatBox.appendChild(createMessageEl(msg)));
  chatBox.scrollTop = chatBox.scrollHeight;
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ type: "load_messages", room_id: currentRoom }));
  }
}

// ── WebSocket ────────────────────────────────────────────────────────────────
function connectChat() {
  if (document.body.dataset.isauth !== "true") {
    if (chatBox)     chatBox.innerHTML = "<p id='auth-warning'>Для входа в чат нужна авторизация.</p>";
    if (chatInput)   chatInput.disabled   = true;
    if (chatSendBtn) chatSendBtn.disabled = true;
    return;
  }
  const warning = document.getElementById("auth-warning");
  if (warning) warning.remove();

  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) return;
  if (socket) { try { socket.close(1000, "reconnect"); } catch(e){} socket = null; }

  socket = new WebSocket("ws://localhost:8080/ws/chat");

  socket.onopen = function() {
    if (chatInput)   chatInput.disabled   = false;
    if (chatSendBtn) chatSendBtn.disabled = false;
    socket.send(JSON.stringify({ type: "load_messages", room_id: currentRoom }));
  };

  // ФИКС 4: switch/case вместо else-if цепочки
  socket.onmessage = function(event) {
    let msg;
    try { msg = JSON.parse(event.data); } catch(e) { return; }

    switch (msg.type) {

      case "message":
        renderMessage(msg);
        break;

      case "messages":
        messagesByRoom[msg.room_id] = msg.messages || [];
        if (msg.room_id === currentRoom) {
          chatBox.innerHTML = "";
          messagesByRoom[currentRoom].forEach(m => chatBox.appendChild(createMessageEl(m)));
          chatBox.scrollTop = chatBox.scrollHeight;
        }
        break;

      case "typing":
        showTyping(msg.sender);
        break;

      case "room_list":
        privateChannelsEl.innerHTML = "";
        window.myPrivateRooms = msg.rooms || [];
        (msg.rooms || []).forEach(function(room) {
          if (room.id === 1) return;
          privateChannelsEl.appendChild(createRoomLi(room.id, room.name));
        });
        break;

      case "new_room":
        privateChannelsEl.appendChild(createRoomLi(msg.room_id, msg.room_name));
        window.myPrivateRooms.push({ id: msg.room_id, name: msg.room_name, is_private: true });
        break;

      case "user_kicked":
        // ФИКС 2: проверяем и по username и по userId (строка)
        if (msg.user === document.body.dataset.username ||
            String(msg.userId) === document.body.dataset.userid) {
          const li = privateChannelsEl.querySelector(`[data-room="${msg.room_id}"]`);
          if (li) li.remove();
          if (currentRoom === msg.room_id) switchRoom(1, "общий-чат");
          window.myPrivateRooms = window.myPrivateRooms.filter(r => r.id !== msg.room_id);
          showToast("Вы были исключены из комнаты", "warning");
        }
        break;

      // ФИКС 2: kick_error — только toast, никакого DOM overlay
      case "kick_error":
        showToast(msg.message || "Нельзя исключить создателя комнаты", "error");
        break;

      case "user_list":
        if (msg.room_id === currentRoom) {
          window.currentUsers = msg.users;
          renderUserList(msg.users);
        }
        break;

      case "invite":
        inviteText.textContent = msg.message;
        inviteModal.style.display = "block";
        inviteAcceptBtn.onclick = function() {
          socket.send(JSON.stringify({ type: "respond_invite", invite_id: msg.invite_id, status: "accepted" }));
          inviteModal.style.display = "none";
        };
        inviteDeclineBtn.onclick = function() {
          socket.send(JSON.stringify({ type: "respond_invite", invite_id: msg.invite_id, status: "declined" }));
          inviteModal.style.display = "none";
        };
        break;

      case "invite_response":
        showToast("Ваше приглашение было " + msg.status, msg.status === "accepted" ? "success" : "error");
        if (msg.status === "accepted" && msg.room_id && msg.room_name) {
          if (!privateChannelsEl.querySelector(`[data-room="${msg.room_id}"]`)) {
            privateChannelsEl.appendChild(createRoomLi(msg.room_id, msg.room_name));
          }
        }
        break;
    }
  };

  socket.onclose = function() {
    socket = null;
    if (chatInput)   chatInput.disabled   = true;
    if (chatSendBtn) chatSendBtn.disabled = true;
  };

  socket.onerror = function(err) { console.error("WS error:", err); };

  chatSendBtn.removeEventListener("click", sendMessage);
  chatSendBtn.addEventListener("click", sendMessage);
  chatInput.removeEventListener("keypress", handleKeyPress);
  chatInput.addEventListener("keypress", handleKeyPress);
}

window.connectChat = connectChat;
window.closeSocket = function(reason) {
  if (socket) { try { socket.close(1000, reason || "logout"); } catch(e){} socket = null; }
};

// ── Публичные каналы ─────────────────────────────────────────────────────────
publicChannelsEl.addEventListener("click", function(e) {
  if (e.target.tagName !== "LI") return;
  const roomId   = e.target.dataset.room;
  const roomName = e.target.textContent;
  switchRoom(roomId, roomName);
  document.querySelectorAll(".chat-channels li").forEach(l => l.classList.remove("active"));
  e.target.classList.add("active");
});

// ── Поиск ────────────────────────────────────────────────────────────────────
if (userSearch) {
  userSearch.addEventListener("input", function() {
    const q = userSearch.value.toLowerCase();
    renderUserList(window.currentUsers.filter(u => u.name.toLowerCase().includes(q)));
  });
}

// ── Создание комнаты ─────────────────────────────────────────────────────────
if (createRoomBtn) {
  createRoomBtn.addEventListener("click", function() {
    const roomName = prompt("Введите название приватного чата:");
    if (roomName && roomName.trim() && socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: "create_room", room_name: roomName.trim() }));
    }
  });
}

// ── Контекстное меню ─────────────────────────────────────────────────────────
document.addEventListener("contextmenu", function(e) {
  const target = e.target.closest(".user-item");
  if (!target) return;
  e.preventDefault();

  const username = target.dataset.username;
  const roomId   = currentRoom;

  const oldMenu = document.querySelector(".context-menu");
  if (oldMenu) oldMenu.remove();

  const menu = document.createElement("div");
  menu.className = "context-menu";

  if (roomId === 1) {
    const privateRooms = window.myPrivateRooms.filter(r => r.is_private);
    if (privateRooms.length === 0) {
      const empty = document.createElement("div");
      empty.className = "menu-item disabled";
      empty.textContent = "Нет приватных чатов";
      menu.appendChild(empty);
    } else {
      privateRooms.forEach(function(room) {
        const item = document.createElement("div");
        item.className = "menu-item";
        item.textContent = `Пригласить в "${room.name}"`;
        item.onclick = function() {
          socket.send(JSON.stringify({ type: "send_invite", to_user: username, room_id: room.id }));
          showToast(`Приглашение отправлено в "${room.name}" → ${username}`, "info");
          menu.remove();
        };
        menu.appendChild(item);
      });
    }
  } else {
    // ФИКС 2: не показываем toast здесь — ждём ответа сервера (user_kicked / kick_error)
    const item = document.createElement("div");
    item.className = "menu-item";
    item.textContent = "Исключить из чата";
    item.onclick = function() {
      socket.send(JSON.stringify({ type: "kick_user", room_id: roomId, target_user: username }));
      menu.remove();
    };
    menu.appendChild(item);
  }

  document.body.appendChild(menu);
  menu.style.left = e.pageX + "px";
  menu.style.top  = e.pageY + "px";

  document.addEventListener("click", () => menu.remove(), { once: true });
});