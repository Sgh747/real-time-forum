package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/middleware"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/models"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/sqlite"
	"github.com/gorilla/websocket"
)

// ChatHandler отвечает за работу чата
type ChatHandler struct {
	DB        *sqlite.DB
	clients   map[*websocket.Conn]int // подключённые клиенты: conn -> user_id
	mu        sync.Mutex
	upgrader  websocket.Upgrader
	Broadcast chan models.Message
}

type UserStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Конструктор
func NewChatHandler(db *sqlite.DB) *ChatHandler {
	return &ChatHandler{
		DB:      db,
		clients: make(map[*websocket.Conn]int),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		Broadcast: make(chan models.Message),
	}
}

// Обработчик WebSocket соединений
func (h *ChatHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	userID := middleware.CurrentUser(r, h.DB)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка апгрейда соединения: %v", err)
		return
	}
	defer conn.Close()

	h.mu.Lock()
	h.clients[conn] = userID
	h.mu.Unlock()

	// отправляем список комнат пользователя
	rooms, err := h.DB.GetRoomsByUser(userID)
	if err == nil {
		payload := map[string]interface{}{
			"type":  "room_list",
			"rooms": rooms,
		}
		conn.WriteJSON(payload)
	}

	// отправляем список пользователей общего чата
	h.sendUserList(1)

	log.Printf("Новый клиент подключился: user_id=%d", userID)

	// При входе — загрузка последних сообщений из общего чата (room_id=1)
	messages, err := h.DB.GetMessages(20, 1)
	if err == nil {
		for _, m := range messages {
			// добавляем type=message для истории
			payload := map[string]interface{}{
				"type":        "message",
				"id":          m.ID,
				"uuid":        m.UUID,
				"sender_id":   m.SenderID,
				"sender":      m.Sender,
				"receiver_id": m.ReceiverID,
				"content":     m.Content,
				"created_at":  m.CreatedAt,
				"room_id":     m.RoomID,
			}
			conn.WriteJSON(payload)
		}
	}

	for {
		var raw map[string]interface{}
		if err := conn.ReadJSON(&raw); err != nil {
			log.Printf("Ошибка чтения сообщения: %v", err)
			break
		}

		switch raw["type"] {
		case "create_room":
			roomName, _ := raw["room_name"].(string)
			if roomName == "" {
				roomName = fmt.Sprintf("Комната %d", userID)
			}

			res, err := h.DB.Conn.Exec(
				`INSERT INTO rooms (uuid, name, is_private, created_at, creator_id)
     			VALUES (lower(hex(randomblob(16))), ?, 1, ?, ?)`,
				roomName, time.Now(), userID,
			)
			if err != nil {
				log.Printf("Ошибка создания комнаты: %v", err)
				continue
			}
			roomID, _ := res.LastInsertId()

			_, err = h.DB.Conn.Exec(
				`INSERT INTO rooms_users (room_id, user_id) VALUES (?, ?)`,
				roomID, userID,
			)
			if err != nil {
				log.Printf("Ошибка добавления создателя в комнату: %v", err)
				continue
			}

			payload := map[string]interface{}{
				"type":      "new_room",
				"room_id":   roomID,
				"room_name": roomName,
			}
			conn.WriteJSON(payload)
			h.sendUserList(int(roomID))

		case "load_messages":
			var roomID int
			switch v := raw["room_id"].(type) {
			case float64:
				roomID = int(v)
			case string:
				id, err := strconv.Atoi(v)
				if err != nil {
					log.Printf("Ошибка конвертации room_id: %v", err)
					roomID = 1 // общий чат по умолчанию
				} else {
					roomID = id
				}
			default:
				roomID = 1 // общий чат по умолчанию
			}

			messages, err := h.DB.GetMessages(20, roomID)
			if err != nil {
				log.Printf("Ошибка загрузки сообщений для комнаты %d: %v", roomID, err)
				continue
			}
			payload := map[string]interface{}{
				"type":     "messages",
				"room_id":  roomID,
				"messages": messages,
			}
			conn.WriteJSON(payload)
			h.sendUserList(roomID)

		case "send_invite":
			toUser, _ := raw["to_user"].(string)
			roomID := int(raw["room_id"].(float64))
			fromUser, _ := h.DB.GetUsernameByID(userID)

			if toUser == fromUser {
				log.Printf("Пользователь попытался пригласить сам себя")
				continue
			}

			resInvite, err := h.DB.Conn.Exec(
				`INSERT INTO invites (room_id, from_user, to_user, status, created_at) VALUES (?, ?, ?, 'pending', ?)`,
				roomID, fromUser, toUser, time.Now(),
			)
			if err != nil {
				log.Printf("Ошибка сохранения приглашения: %v", err)
				continue
			}
			inviteID, _ := resInvite.LastInsertId()

			var roomName string
			_ = h.DB.Conn.QueryRow(`SELECT name FROM rooms WHERE id = ?`, roomID).Scan(&roomName)

			payload := map[string]interface{}{
				"type":      "invite",
				"invite_id": inviteID,
				"from_user": fromUser,
				"room_id":   roomID,
				"room_name": roomName,
				"message":   fmt.Sprintf("Хотите принять приглашение от %s войти в приватный чат \"%s\"?", fromUser, roomName),
			}

			h.mu.Lock()
			for client, uid := range h.clients {
				username, _ := h.DB.GetUsernameByID(uid)
				if username == toUser {
					client.WriteJSON(payload)
				}
			}
			h.mu.Unlock()

		case "respond_invite":
			inviteID := int(raw["invite_id"].(float64))
			status := raw["status"].(string)

			_, err := h.DB.Conn.Exec(`UPDATE invites SET status = ? WHERE id = ?`, status, inviteID)
			if err != nil {
				log.Printf("Ошибка обновления приглашения: %v", err)
				continue
			}

			// если приглашение принято — добавляем пользователя в комнату
			if status == "accepted" {
				row := h.DB.Conn.QueryRow(`SELECT room_id FROM invites WHERE id = ?`, inviteID)
				var roomID int
				if err := row.Scan(&roomID); err == nil {
					_, _ = h.DB.Conn.Exec(`INSERT INTO rooms_users (room_id, user_id) VALUES (?, ?)`, roomID, userID)

					// отправляем обновлённый список комнат приглашённому
					rooms, _ := h.DB.GetRoomsByUser(userID)
					conn.WriteJSON(map[string]interface{}{
						"type":  "room_list",
						"rooms": rooms,
					})
				}
			}

			// находим инициатора приглашения
			row := h.DB.Conn.QueryRow(`SELECT from_user, room_id FROM invites WHERE id = ?`, inviteID)
			var fromUser string
			var roomID int
			if err := row.Scan(&fromUser, &roomID); err == nil {
				// получаем имя комнаты
				var roomName string
				_ = h.DB.Conn.QueryRow(`SELECT name FROM rooms WHERE id = ?`, roomID).Scan(&roomName)

				payload := map[string]interface{}{
					"type":      "invite_response",
					"status":    status,
					"room_id":   roomID,
					"room_name": roomName,
				}

				h.mu.Lock()
				for client, uid := range h.clients {
					username, _ := h.DB.GetUsernameByID(uid)
					if username == fromUser {
						client.WriteJSON(payload)
					}
				}
				h.mu.Unlock()
			}

		case "typing":
			username, _ := h.DB.GetUsernameByID(userID)
			payload := map[string]interface{}{
				"type":   "typing",
				"sender": username,
			}
			h.mu.Lock()
			for client := range h.clients {
				client.WriteJSON(payload)
			}
			h.mu.Unlock()

			// В обработке "message" убираем автоподмену room_id=1
		case "message":
			var msg models.Message
			if content, ok := raw["content"].(string); ok {
				msg.Content = content
			}
			msg.SenderID = userID
			msg.CreatedAt = time.Now()

			// корректная обработка room_id
			var roomID int
			switch v := raw["room_id"].(type) {
			case float64:
				roomID = int(v)
			case string:
				id, err := strconv.Atoi(v)
				if err != nil {
					log.Printf("Некорректный room_id: %v", err)
					continue // не сохраняем сообщение
				}
				roomID = id
			default:
				log.Printf("room_id отсутствует или некорректен")
				continue
			}

			// проверка: состоит ли пользователь в комнате
			inRoom := false
			if roomID == 1 {
				// общий чат всегда доступен
				inRoom = true
			} else {
				var err error
				inRoom, err = h.DB.IsUserInRoom(roomID, userID)
				if err != nil {
					log.Printf("Ошибка проверки участия в комнате: %v", err)
					continue
				}
			}

			if !inRoom {
				log.Printf("Пользователь %d не состоит в комнате %d", userID, roomID)
				continue
			}

			msg.RoomID = roomID

			if err := h.DB.SaveMessage(msg.SenderID, msg.Content, msg.CreatedAt, msg.RoomID); err != nil {
				log.Printf("Ошибка сохранения сообщения: %v", err)
				continue
			}

			savedMsg, err := h.DB.GetLastMessageByUser(msg.SenderID)
			if err == nil {
				h.Broadcast <- savedMsg
			} else {
				h.Broadcast <- msg
			}

		// Новый кейс: исключение пользователя из приватного чата
		case "kick_user":
			roomID := int(raw["room_id"].(float64))
			targetUser := raw["target_user"].(string)

			// получаем ID текущего пользователя
			currentUserID := userID
			currentUsername, _ := h.DB.GetUsernameByID(currentUserID)

			// получаем creator_id комнаты
			var creatorID int
			err := h.DB.Conn.QueryRow(`SELECT creator_id FROM rooms WHERE id = ?`, roomID).Scan(&creatorID)
			if err != nil {
				log.Printf("Ошибка получения создателя комнаты: %v", err)
				continue
			}

			// проверка прав:
			// - если текущий пользователь не создатель и пытается исключить кого-то другого → запрещаем
			if currentUserID != creatorID && targetUser != currentUsername {
				log.Printf("Пользователь %s не имеет права исключать других", currentUsername)

				// отправляем инициатору событие об ошибке
				errorPayload := map[string]interface{}{
					"type":    "kick_error",
					"room_id": roomID,
					"message": "Вы не можете исключить создателя приватного чата",
				}
				conn.WriteJSON(errorPayload)

				continue
			}

			// исключаем
			uid, err := h.DB.GetUserIDByUsername(targetUser)
			if err != nil {
				log.Printf("Ошибка поиска пользователя: %v", err)
				continue
			}

			if err := h.DB.RemoveUserFromRoom(roomID, uid); err != nil {
				log.Printf("Ошибка удаления пользователя из комнаты: %v", err)
				continue
			}

			payload := map[string]interface{}{
				"type":    "user_kicked",
				"room_id": roomID,
				"user":    targetUser,
			}

			// отправляем событие исключённому пользователю
			h.mu.Lock()
			for client, uid := range h.clients {
				username, _ := h.DB.GetUsernameByID(uid)
				if username == targetUser {
					client.WriteJSON(payload)
				}
			}
			h.mu.Unlock()

			// обновляем список участников у всех
			h.sendUserList(roomID)
		}
	}
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	log.Printf("Клиент отключился: user_id=%d", userID)

	rooms, _ = h.DB.GetRoomsByUser(userID)
	for _, r := range rooms {
		h.sendUserList(r.ID)
	}
	h.sendUserList(1)
}

// Горутинa для рассылки сообщений
func (h *ChatHandler) HandleMessages() {
	for {
		msg := <-h.Broadcast
		h.broadcast(msg)
	}
}

// Рассылка сообщений всем клиентам
func (h *ChatHandler) broadcast(msg models.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	payload := map[string]interface{}{
		"type":        "message",
		"id":          msg.ID,
		"uuid":        msg.UUID,
		"sender_id":   msg.SenderID,
		"sender":      msg.Sender,
		"receiver_id": msg.ReceiverID,
		"content":     msg.Content,
		"created_at":  msg.CreatedAt,
		"room_id":     msg.RoomID,
	}

	for client := range h.clients {
		err := client.WriteJSON(payload)
		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			client.Close()
			delete(h.clients, client)
		}
	}
}

func (h *ChatHandler) buildUserList(roomID int) []models.UserStatus {
	var users []models.User
	var err error

	if roomID == 1 {
		users, err = h.DB.GetAllUsers()
	} else {
		users, err = h.DB.GetUsersByRoom(roomID)
	}

	if err != nil {
		log.Printf("Ошибка получения списка пользователей: %v", err)
		return nil
	}

	var list []models.UserStatus
	for _, u := range users {
		status := "offline"
		for _, id := range h.clients {
			if id == u.ID {
				status = "online"
				break
			}
		}
		list = append(list, models.UserStatus{
			Name:   u.Username,
			Status: status,
		})
	}
	return list
}

func (h *ChatHandler) sendUserList(roomID int) {
	list := h.buildUserList(roomID)
	payload := map[string]interface{}{
		"type":    "user_list",
		"room_id": roomID,
		"users":   list,
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for client, uid := range h.clients {
		if roomID == 1 {
			client.WriteJSON(payload)
			continue
		}
		inRoom, err := h.DB.IsUserInRoom(roomID, uid)
		if err != nil {
			continue
		}
		if inRoom {
			client.WriteJSON(payload)
		}
	}
}
