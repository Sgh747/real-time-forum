package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/models" // единая структура Message
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// DB — обёртка над *sql.DB, чтобы можно было добавлять методы
type DB struct {
	Conn *sql.DB
}

// Open открывает базу forum.db и применяет все миграции по порядку.
func Open(dbFile string, migrationFiles []string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия базы: %w", err)
	}

	// Включаем поддержку внешних ключей
	if _, err := conn.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ошибка включения foreign_keys: %w", err)
	}

	// Прогоняем все миграции по порядку
	for _, file := range migrationFiles {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("ошибка чтения миграции %s: %w", file, err)
		}

		if _, err := conn.Exec(string(sqlBytes)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("ошибка выполнения миграции %s: %w", file, err)
		}
	}

	return &DB{Conn: conn}, nil
}

// Close закрывает соединение
func (db *DB) Close() {
	db.Conn.Close()
}

// SaveMessage сохраняет сообщение в таблицу messages
func (db *DB) SaveMessage(senderID int, content string, createdAt time.Time, roomID int) error {
	_, err := db.Conn.Exec(
		`INSERT INTO messages (uuid, sender_id, receiver_id, content, created_at, room_id)
         VALUES (?, ?, NULL, ?, ?, ?)`,
		uuid.New().String(), senderID, content, createdAt, roomID,
	)
	return err
}

// GetMessages возвращает последние N сообщений (например, для истории чата)
func (db *DB) GetMessages(limit int, roomID int) ([]models.Message, error) {
	// Сначала выбираем последние N сообщений в порядке DESC
	rows, err := db.Conn.Query(
		`SELECT m.id, m.uuid, m.sender_id, u.username AS sender,
                m.receiver_id, m.content, m.created_at, m.room_id
         FROM messages m
         JOIN users u ON m.sender_id = u.id
         WHERE m.room_id = ?
         ORDER BY m.created_at DESC
         LIMIT ?`,
		roomID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.UUID, &msg.SenderID, &msg.Sender,
			&msg.ReceiverID, &msg.Content, &msg.CreatedAt, &msg.RoomID); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Переворачиваем массив, чтобы порядок был от старых к новым
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetLastMessageByUser возвращает последнее сообщение пользователя
func (db *DB) GetLastMessageByUser(userID int) (models.Message, error) {
	row := db.Conn.QueryRow(`
        SELECT m.id, m.uuid, m.sender_id, u.username,
               m.receiver_id, m.content, m.created_at, m.room_id
        FROM messages m
        JOIN users u ON m.sender_id = u.id
        WHERE m.sender_id = ?
        ORDER BY m.created_at DESC
        LIMIT 1
    `, userID)

	var msg models.Message
	err := row.Scan(&msg.ID, &msg.UUID, &msg.SenderID, &msg.Sender,
		&msg.ReceiverID, &msg.Content, &msg.CreatedAt, &msg.RoomID)
	return msg, err
}

// GetAllUsers возвращает всех зарегистрированных пользователей
func (db *DB) GetAllUsers() ([]models.User, error) {
	rows, err := db.Conn.Query(`SELECT id, username FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Получить все комнаты, где состоит пользователь
func (db *DB) GetRoomsByUser(userID int) ([]models.Room, error) {
	rows, err := db.Conn.Query(`
        SELECT r.id, r.uuid, r.name, r.is_private, r.created_at
        FROM rooms r
        JOIN rooms_users ru ON r.id = ru.room_id
        WHERE ru.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []models.Room
	for rows.Next() {
		var room models.Room
		if err := rows.Scan(&room.ID, &room.UUID, &room.Name, &room.IsPrivate, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

// CreateRoom создаёт новую комнату с указанным именем и добавляет создателя
func (db *DB) CreateRoom(name string, isPrivate bool, creatorID int) (int, error) {
	res, err := db.Conn.Exec(
		`INSERT INTO rooms (uuid, name, is_private, created_at, creator_id) 
         VALUES (lower(hex(randomblob(16))), ?, ?, ?)`,
		name, isPrivate, time.Now(),
	)
	if err != nil {
		return 0, err
	}

	roomID64, _ := res.LastInsertId()
	roomID := int(roomID64)

	// добавляем создателя в комнату
	if err := db.AddUserToRoom(roomID, creatorID); err != nil {
		return 0, err
	}

	return roomID, nil
}

// Получить всех пользователей комнаты
func (db *DB) GetUsersByRoom(roomID int) ([]models.User, error) {
	rows, err := db.Conn.Query(`
        SELECT u.id, u.username 
        FROM users u
        JOIN rooms_users ru ON u.id = ru.user_id
        WHERE ru.room_id = ?
		ORDER BY u.username ASC`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Получить userID по username
func (db *DB) GetUserIDByUsername(username string) (int, error) {
	var id int
	err := db.Conn.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Добавить пользователя в комнату
func (db *DB) AddUserToRoom(roomID, userID int) error {
	_, err := db.Conn.Exec(
		`INSERT INTO rooms_users (room_id, user_id) VALUES (?, ?)`,
		roomID, userID,
	)
	return err
}

func (db *DB) RemoveUserFromRoom(roomID, userID int) error {
	_, err := db.Conn.Exec(`DELETE FROM rooms_users WHERE room_id = ? AND user_id = ?`, roomID, userID)
	return err
}

// Проверить, состоит ли пользователь в комнате
func (db *DB) IsUserInRoom(roomID, userID int) (bool, error) {
	row := db.Conn.QueryRow(
		`SELECT COUNT(*) FROM rooms_users WHERE room_id = ? AND user_id = ?`,
		roomID, userID,
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUsernameByID возвращает username по userID
func (db *DB) GetUsernameByID(userID int) (string, error) {
	var username string
	err := db.Conn.QueryRow(`SELECT username FROM users WHERE id = ?`, userID).Scan(&username)
	if err != nil {
		return "", err
	}
	return username, nil
}

func (db *DB) SavePost(post models.Post) error {
	_, err := db.Conn.Exec(
		`INSERT INTO posts (user_id, title, content, created_at)
         VALUES (?, ?, ?, ?)`,
		post.UserID, post.Title, post.Content, post.CreatedAt,
	)
	return err
}

func (db *DB) GetPosts(limit int) ([]models.Post, error) {
	rows, err := db.Conn.Query(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.created_at
         FROM posts p
         JOIN users u ON p.user_id = u.id
         ORDER BY p.created_at DESC
         LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.UserID, &post.Username,
			&post.Title, &post.Content, &post.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (db *DB) UpdatePost(post models.Post) error {
	_, err := db.Conn.Exec(
		`UPDATE posts SET title = ?, content = ? WHERE id = ?`,
		post.Title, post.Content, post.ID,
	)
	return err
}

// Обновление категорий поста
func (db *DB) UpdatePostCategories(postID int, categoryIDs []int) error {
	// сначала удаляем старые связи
	_, err := db.Conn.Exec(`DELETE FROM post_categories WHERE post_id = ?`, postID)
	if err != nil {
		return err
	}

	// добавляем новые
	for _, catID := range categoryIDs {
		_, err := db.Conn.Exec(
			`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`,
			postID, catID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) DeletePost(postID int) error {
	_, err := db.Conn.Exec(`DELETE FROM posts WHERE id = ?`, postID)
	return err
}

func (db *DB) VotePost(userID, postID, value int) error {
	var currentValue int
	err := db.Conn.QueryRow(
		`SELECT value FROM votes WHERE user_id = ? AND post_id = ?`,
		userID, postID,
	).Scan(&currentValue)

	if err == sql.ErrNoRows {
		// голоса ещё нет → вставляем
		_, err = db.Conn.Exec(
			`INSERT INTO votes (user_id, post_id, value, created_at)
             VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			userID, postID, value,
		)
		return err
	} else if err != nil {
		return err
	}

	if currentValue == value {
		// повторное нажатие → удаляем голос
		_, err = db.Conn.Exec(
			`DELETE FROM votes WHERE user_id = ? AND post_id = ?`,
			userID, postID,
		)
		return err
	}

	// меняем голос
	_, err = db.Conn.Exec(
		`UPDATE votes SET value = ?, created_at = CURRENT_TIMESTAMP
         WHERE user_id = ? AND post_id = ?`,
		value, userID, postID,
	)
	return err
}

func (db *DB) AddTagToPost(postID int, tagName string) error {
	// создаём тег, если его нет
	_, err := db.Conn.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, tagName)
	if err != nil {
		return err
	}

	// связываем пост и тег
	_, err = db.Conn.Exec(
		`INSERT OR IGNORE INTO post_tags (post_id, tag_id)
         SELECT ?, id FROM tags WHERE name = ?`,
		postID, tagName,
	)
	return err
}

func (db *DB) GetPostByID(postID int) (models.Post, error) {
	row := db.Conn.QueryRow(`
        SELECT p.id, p.user_id, u.username, p.title, p.content, p.created_at,
               IFNULL(SUM(v.value), 0) as votes,
               COALESCE(SUM(CASE WHEN v.value = 1 THEN 1 ELSE 0 END), 0) AS likes,
               COALESCE(SUM(CASE WHEN v.value = -1 THEN 1 ELSE 0 END), 0) AS dislikes,
               (
                 SELECT GROUP_CONCAT(name, ', ')
                 FROM (
                   SELECT DISTINCT c2.name
                   FROM post_categories pc2
                   JOIN categories c2 ON pc2.category_id = c2.id
                   WHERE pc2.post_id = p.id
                   ORDER BY c2.name
                 )
               ) AS categories
        FROM posts p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN votes v ON v.post_id = p.id AND v.comment_id IS NULL
        WHERE p.id = ?
        GROUP BY p.id
    `, postID)

	var post models.Post
	err := row.Scan(
		&post.ID,
		&post.UserID,
		&post.Username,
		&post.Title,
		&post.Content,
		&post.CreatedAt,
		&post.Votes,
		&post.Likes,
		&post.Dislikes,
		&post.Categories,
	)
	return post, err
}

func (db *DB) VoteComment(userID, commentID, value int) error {
	var currentValue int
	err := db.Conn.QueryRow(
		`SELECT value FROM votes WHERE user_id = ? AND comment_id = ?`,
		userID, commentID,
	).Scan(&currentValue)

	if err == sql.ErrNoRows {
		_, err = db.Conn.Exec(
			`INSERT INTO votes (user_id, comment_id, value, created_at)
             VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			userID, commentID, value,
		)
		return err
	} else if err != nil {
		return err
	}

	if currentValue == value {
		_, err = db.Conn.Exec(
			`DELETE FROM votes WHERE user_id = ? AND comment_id = ?`,
			userID, commentID,
		)
		return err
	}

	_, err = db.Conn.Exec(
		`UPDATE votes SET value = ?, created_at = CURRENT_TIMESTAMP
         WHERE user_id = ? AND comment_id = ?`,
		value, userID, commentID,
	)
	return err
}

func (db *DB) GetCommentsByPostID(postID int) ([]models.Comment, error) {
	rows, err := db.Conn.Query(`
        SELECT c.id, c.post_id, c.user_id, u.username, c.content, c.created_at,
               COALESCE(SUM(CASE WHEN v.value = 1 THEN 1 ELSE 0 END), 0) AS likes,
               COALESCE(SUM(CASE WHEN v.value = -1 THEN 1 ELSE 0 END), 0) AS dislikes
        FROM comments c
        JOIN users u ON c.user_id = u.id
        LEFT JOIN votes v ON v.comment_id = c.id
        WHERE c.post_id = ?
        GROUP BY c.id
        ORDER BY c.created_at ASC
    `, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(
			&c.ID,
			&c.PostID,
			&c.UserID,
			&c.Username,
			&c.Content,
			&c.CreatedAt,
			&c.Likes,
			&c.Dislikes,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}
