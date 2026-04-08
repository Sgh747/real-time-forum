const chatBox = document.getElementById("chat-box");
const chatInput = document.getElementById("chat-input");
const chatSendBtn = document.getElementById("chat-send-btn");
const typingIndicator = document.getElementById("typing-indicator");
const userListEl = document.getElementById("user-list");

// Элементы модалки приглашения
const inviteModal = document.getElementById("view-invite");
const inviteText = document.getElementById("invite-text");
const inviteAcceptBtn = document.getElementById("invite-accept");
const inviteDeclineBtn = document.getElementById("invite-decline");

// Элементы для каналов и поиска
const publicChannelsEl = document.getElementById("public-channels");
const privateChannelsEl = document.getElementById("private-channels");
const createRoomBtn = document.getElementById("create-room-btn");
const userSearch = document.getElementById("user-search");
const chatHeader = document.getElementById("chat-header");

let socket = null;
let typingTimeout = null;
let currentRoom = 1; // общий чат по умолчанию
window.currentUsers = []; // список пользователей для поиска
window.myPrivateRooms = []; // список приватных комнат пользователя

// Хранилище сообщений по комнатам
const messagesByRoom = {};

function renderMessage(msg) {
  const roomId = msg.room_id;

  if (!messagesByRoom[roomId]) {
    messagesByRoom[roomId] = [];
  }
  messagesByRoom[roomId].push(msg);

  if (roomId === currentRoom) {
    const item = document.createElement("div");
    item.className = "chat-message";

    const sender = msg.sender || msg.username || "Anon";
    const content = msg.content || msg.text || "";
    const time =
      msg.created_at ||
      msg.timestamp ||
      new Date().toLocaleTimeString();

    item.innerHTML =
      "<strong class=\"chat-username\">" + sender + "</strong>: " +
      "<span class=\"chat-text\">" + content + "</span> " +
      "<span class=\"chat-time\">(" + time + ")</span>";

    chatBox.appendChild(item);
    chatBox.scrollTop = chatBox.scrollHeight;
  }
}

function renderUserList(users) {
  if (!userListEl) return;
  userListEl.innerHTML = "";
  users.forEach(function(u) {
    const li = document.createElement("li");
    li.className = "user user-item " + u.status;
    li.dataset.username = u.name;
    li.innerHTML = "<span class=\"status-dot\"></span> " + u.name;
    userListEl.appendChild(li);
  });
}

function sendMessage() {
  if (!chatInput.value) return;
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify({
    type: "message",
    room_id: currentRoom,
    content: chatInput.value
  }));
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
  typingTimeout = setTimeout(function() {
    typingIndicator.textContent = "";
  }, 3000);
}

function switchRoom(roomId, roomName) {
  currentRoom = parseInt(roomId, 10);

  if (currentRoom === 1) {
    chatHeader.textContent = "# общий-чат";
  } else {
    chatHeader.textContent = "# " + roomName;
  }

  chatBox.innerHTML = "";
  (messagesByRoom[currentRoom] || []).forEach(renderMessage);

  socket.send(JSON.stringify({ type: "load_messages", room_id: currentRoom }));
}

function connectChat() {
  if (document.body.dataset.isauth !== "true") {
    // если НЕ авторизован → показываем надпись
    if (chatBox)
      chatBox.innerHTML = "<p id='auth-warning'>Для входа в чат нужна авторизация.</p>";
    if (chatInput) chatInput.disabled = true;
    if (chatSendBtn) chatSendBtn.disabled = true;
    return;
  } else {
    // если авторизован → удаляем только надпись
    const warning = document.getElementById("auth-warning");
    if (warning) warning.remove();
  }

  if (
    socket &&
    (socket.readyState === WebSocket.OPEN ||
      socket.readyState === WebSocket.CONNECTING)
  ) {
    return;
  }

  if (socket) {
    try {
      socket.close(1000, "reconnect");
    } catch (e) {}
    socket = null;
  }

  socket = new WebSocket("ws://localhost:8080/ws/chat");

  socket.onopen = function() {
    chatInput.disabled = false;
    chatSendBtn.disabled = false;

    // сразу запросить историю сообщений для текущей комнаты
    socket.send(JSON.stringify({ type: "load_messages", room_id: currentRoom }));
  };

  socket.onmessage = function(event) {
    const msg = JSON.parse(event.data);

    if (msg.type === "message") {
      if (msg.room_id === currentRoom) {
        renderMessage(msg);
      }
    } else if (msg.type === "messages") {
      messagesByRoom[msg.room_id] = msg.messages || [];
      chatBox.innerHTML = "";
      (messagesByRoom[currentRoom] || []).forEach(renderMessage);

    } else if (msg.type === "typing") {
      showTyping(msg.sender);

    } else if (msg.type === "room_list") {
      privateChannelsEl.innerHTML = "";
      window.myPrivateRooms = msg.rooms || [];
      (msg.rooms || []).forEach(function(room) {
        if (room.id === 1) return; // общий чат пропускаем
        const li = document.createElement("li");
        li.textContent = "# " + room.name;
        li.dataset.room = room.id;
        privateChannelsEl.appendChild(li);

        if (!messagesByRoom[room.id]) {
          messagesByRoom[room.id] = [];
        }

        li.addEventListener("click", function() {
          switchRoom(room.id, room.name);
          document.querySelectorAll(".chat-channels li").forEach(function(li) {
            li.classList.remove("active");
          });
          li.classList.add("active");
        });
      });

    } else if (msg.type === "new_room") {
      // обработка создания новой комнаты
      const li = document.createElement("li");
      li.textContent = "# " + msg.room_name;
      li.dataset.room = msg.room_id;
      privateChannelsEl.appendChild(li);

      if (!messagesByRoom[msg.room_id]) {
        messagesByRoom[msg.room_id] = [];
      }
      li.addEventListener("click", function() {
        switchRoom(msg.room_id, msg.room_name);
        document.querySelectorAll(".chat-channels li").forEach(function(li) {
          li.classList.remove("active");
        });
        li.classList.add("active");
      });

      // 🔑 добавляем комнату в массив приватных комнат
      window.myPrivateRooms.push({
        id: msg.room_id,
        name: msg.room_name,
        is_private: true
      }); 
    } else if (msg.type === "user_kicked") {
      // проверяем, совпадает ли исключённый пользователь с текущим
      if (msg.user === document.body.dataset.username ||
          msg.userId === parseInt(document.body.dataset.userid, 10)) {

        // удаляем комнату из DOM
        const li = privateChannelsEl.querySelector(`[data-room="${msg.room_id}"]`);
        if (li) li.remove();

        // если пользователь находился в этой комнате → переключаемся в общий чат
        if (currentRoom === msg.room_id) {
          switchRoom(1, "общий-чат");
        }
        
        // удаляем комнату из массива приватных комнат
        window.myPrivateRooms = window.myPrivateRooms.filter(r => r.id !== msg.room_id);
        showToast("Вы были исключены из комнаты", "warning");
      }
      } else if (msg.type === "kick_error") {
        // обработка ошибки при попытке исключить создателя
        showToast(msg.message, "error");
    } else if (msg.type === "user_list") {
      if (msg.room_id === currentRoom) {
      window.currentUsers = msg.users;
      renderUserList(msg.users);
    }
    } else if (msg.type === "invite") {
      inviteText.textContent = msg.message;
      inviteModal.style.display = "block";

      inviteAcceptBtn.onclick = function() {
        socket.send(JSON.stringify({
          type: "respond_invite",
          invite_id: msg.invite_id,
          status: "accepted"
        }));
        inviteModal.style.display = "none";
      };

      inviteDeclineBtn.onclick = function() {
        socket.send(JSON.stringify({
          type: "respond_invite",
          invite_id: msg.invite_id,
          status: "declined"
        }));
        inviteModal.style.display = "none";
      };

    } else if (msg.type === "invite_response") {
      showToast("Ваше приглашение было " + msg.status,
        msg.status === "accepted" ? "success" : "error");

      if (msg.status === "accepted" && msg.room_id && msg.room_name) {
        // проверяем, нет ли уже такой комнаты
        if (!privateChannelsEl.querySelector(`[data-room="${msg.room_id}"]`)) {
          const li = document.createElement("li");
          li.textContent = "# " + msg.room_name;
          li.dataset.room = msg.room_id;
          privateChannelsEl.appendChild(li);

          if (!messagesByRoom[msg.room_id]) {
            messagesByRoom[msg.room_id] = [];
          }

          li.addEventListener("click", function() {
            switchRoom(msg.room_id, msg.room_name);
            document.querySelectorAll(".chat-channels li").forEach(function(li) {
              li.classList.remove("active");
            });
            li.classList.add("active");
          });
        }
      }
    }
  };

  socket.onclose = function() {
    socket = null;
    chatInput.disabled = true;
    chatSendBtn.disabled = true;
  };

  socket.onerror = function(err) {
    console.error("WS error:", err);
  };

  chatSendBtn.removeEventListener("click", sendMessage);
  chatSendBtn.addEventListener("click", sendMessage);
  chatInput.removeEventListener("keypress", handleKeyPress);
  chatInput.addEventListener("keypress", handleKeyPress);
}

window.connectChat = connectChat;
window.closeSocket = function(reason) {
  if (socket) {
    try {
      socket.close(1000, reason || "logout");
    } catch (e) {}
    socket = null;
  }
};

publicChannelsEl.addEventListener("click", function(e) {
  if (e.target.tagName === "LI") {
    const roomId = e.target.dataset.room;
    const roomName = e.target.textContent;
    switchRoom(roomId, roomName);
    document.querySelectorAll(".chat-channels li").forEach(function(li) {
      li.classList.remove("active");
    });
    e.target.classList.add("active");
  }
});


if (userSearch) {
  userSearch.addEventListener("input", function() {
    const query = userSearch.value.toLowerCase();
    const filtered = window.currentUsers.filter(function(u) {
      return u.name.toLowerCase().includes(query);
    });
    renderUserList(filtered);
  });
}

if (createRoomBtn) {
  createRoomBtn.addEventListener("click", function() {
    const roomName = prompt("Введите название приватного чата:");
    if (roomName) {
      socket.send(JSON.stringify({
        type: "create_room",
        room_name: roomName
      }));
    }
  });
}

document.addEventListener("contextmenu", function(e) {
  const target = e.target.closest(".user-item"); // ищем ближайший li

  if (!target) {
    // если клик не по .user-item → разрешаем стандартное меню
    return;
  }

  e.preventDefault(); // блокируем только для .user-item

  const username = target.dataset.username;
  const roomId = currentRoom;

  const oldMenu = document.querySelector(".context-menu");
  if (oldMenu) oldMenu.remove();

  const menu = document.createElement("div");
  menu.className = "context-menu";

  if (roomId === 1) {
    // список всех приватных комнат
    if (window.myPrivateRooms.length === 0) {
      menu.innerHTML = "<div class=\"menu-item\">Нет приватных чатов</div>";
    } else {
      window.myPrivateRooms.forEach(function(room) {
        if (!room.is_private) return;
        const item = document.createElement("div");
        item.className = "menu-item";
        item.textContent = "Пригласить в приватный чат \"" + room.name + "\"";
        item.onclick = function() {
          socket.send(JSON.stringify({
            type: "send_invite",
            to_user: username,
            room_id: room.id
          }));
          showToast("Приглашение отправлено в \"" + room.name + "\" пользователю " + username, "info");
          menu.remove();
        };
        menu.appendChild(item);
      });
    }
  } else {
    // приватная комната → пункт "Исключить из чата"
    const item = document.createElement("div");
    item.className = "menu-item";
    item.textContent = "Исключить из чата";
    item.onclick = function() {
      socket.send(JSON.stringify({
        type: "kick_user",
        room_id: roomId,
        target_user: username
      }));
      showToast("Пользователь " + username + " исключён из комнаты", "warning");
      menu.remove();
    };
    menu.appendChild(item);
  }

  document.body.appendChild(menu);
  menu.style.left = e.pageX + "px";
  menu.style.top = e.pageY + "px";

  document.addEventListener("click", function() {
    menu.remove();
  }, { once: true });
});
