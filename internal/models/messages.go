package models

import (
	"database/sql"
	"time"
)

// Message — единая структура для работы с сообщениями
type Message struct {
	ID         int           `json:"id"`
	UUID       string        `json:"uuid"`
	SenderID   int           `json:"sender_id"`
	Sender     string        `json:"sender"`
	ReceiverID sql.NullInt64 `json:"receiver_id"`
	Content    string        `json:"content"`
	CreatedAt  time.Time     `json:"created_at"`
	RoomID     int           `json:"room_id"` // новое поле для поддержки комнат
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type UserStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// 🔒 Новая структура для приватных комнат
type Room struct {
	ID        int       `json:"id"`
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	IsPrivate bool      `json:"is_private"`
	CreatedAt time.Time `json:"created_at"`
}

// 📩 Структура для приглашений в приватные комнаты
type Invite struct {
	ID        int       `json:"id"`
	RoomID    int       `json:"room_id"`
	FromUser  string    `json:"from_user"`
	ToUser    string    `json:"to_user"`
	Status    string    `json:"status"` // pending, accepted, declined
	CreatedAt time.Time `json:"created_at"`
}
