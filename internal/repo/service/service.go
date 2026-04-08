package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/models"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/sqlite"
)

type ForumService struct {
	DB *sqlite.DB
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Post struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Username   string    `json:"username"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	CreatedAt  string    `json:"created_at"`
	Comments   []Comment `json:"comments"`
	Rating     int       `json:"rating"`
	Likes      int       `json:"likes"`
	Dislikes   int       `json:"dislikes"`
	Categories []string  `json:"categories"`
}

type Comment struct {
	ID        int    `json:"id"`
	PostID    int    `json:"post_id"`
	UserID    int    `json:"user_id"`
	User      string `json:"user"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	Rating    int    `json:"rating"`
	Likes     int    `json:"likes"`
	Dislikes  int    `json:"dislikes"`
}

// ---------- Создание поста ----------
func (s *ForumService) CreatePost(userID int, title, content string) (int, error) {
	res, err := s.DB.Conn.Exec(
		`INSERT INTO posts (user_id, title, content, created_at) VALUES (?, ?, ?, ?)`,
		userID, title, content, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// ---------- Комментарии ----------
func (s *ForumService) AddComment(postID, userID int, content string) error {
	_, err := s.DB.Conn.Exec(
		`INSERT INTO comments (post_id, user_id, content, created_at) VALUES (?, ?, ?, ?)`,
		postID, userID, content, time.Now(),
	)
	return err
}

// ---------- Голосование ----------
func (s *ForumService) AddVote(userID, postID, commentID, value int) error {
	if value != 1 && value != -1 {
		return fmt.Errorf("invalid vote value")
	}

	// Если голос для комментария
	if commentID != 0 {
		var currentValue int
		err := s.DB.Conn.QueryRow(
			`SELECT value FROM votes WHERE user_id = ? AND comment_id = ?`,
			userID, commentID,
		).Scan(&currentValue)

		if err == sql.ErrNoRows {
			// нет голоса → вставляем
			_, err = s.DB.Conn.Exec(
				`INSERT INTO votes (user_id, comment_id, value, created_at)
                 VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
				userID, commentID, value,
			)
			return err
		} else if err != nil {
			return err
		}

		if currentValue == value {
			// повторное нажатие → удаляем
			_, err = s.DB.Conn.Exec(
				`DELETE FROM votes WHERE user_id = ? AND comment_id = ?`,
				userID, commentID,
			)
			return err
		}

		// меняем голос
		_, err = s.DB.Conn.Exec(
			`UPDATE votes SET value = ?, created_at = CURRENT_TIMESTAMP
             WHERE user_id = ? AND comment_id = ?`,
			value, userID, commentID,
		)
		return err
	}

	// Если голос для поста → используем готовую логику
	return s.DB.VotePost(userID, postID, value)
}

// ---------- Категории ----------
func (s *ForumService) ListCategories() ([]Category, error) {
	rows, err := s.DB.Conn.Query(`SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (s *ForumService) AddPostCategories(postID int, categoryIDs []int) error {
	for _, catID := range categoryIDs {
		_, err := s.DB.Conn.Exec(
			`INSERT OR IGNORE INTO post_categories (post_id, category_id) VALUES (?, ?)`,
			postID, catID,
		)
		if err != nil {
			return fmt.Errorf("failed to add category %d to post %d: %w", catID, postID, err)
		}
	}
	return nil
}

func (s *ForumService) queryPosts(extra string, args ...interface{}) ([]Post, error) {
	query := `
        SELECT p.id, p.user_id, u.username, p.title, p.content, p.created_at,
               IFNULL(SUM(v.value), 0) as rating,
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
        GROUP BY p.id
        ` + extra
	rows, err := s.DB.Conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanPosts(rows)
}

// ---------- Методы выборки ----------
func (s *ForumService) ListPosts() ([]Post, error) {
	return s.queryPosts("ORDER BY p.created_at DESC")
}

func (s *ForumService) GetPostsByRatingDesc() ([]Post, error) {
	return s.queryPosts("ORDER BY rating DESC")
}

func (s *ForumService) GetPostsByRatingAsc() ([]Post, error) {
	return s.queryPosts("ORDER BY rating ASC")
}

func (s *ForumService) GetRecentPosts(limit int) ([]Post, error) {
	return s.queryPosts("LIMIT ?", limit)
}

func (s *ForumService) GetPostByID(id int) (*models.Post, error) {
	post, err := s.DB.GetPostByID(id)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *ForumService) GetPostsByCategory(category string) ([]Post, error) {
	return s.queryPosts(`
        WHERE EXISTS (
            SELECT 1 FROM post_categories pcx
            JOIN categories cx ON pcx.category_id = cx.id
            WHERE pcx.post_id = p.id AND cx.name = ?
        )`, category)
}

// Посты по пользователю
func (s *ForumService) GetPostsByUser(userID int) ([]Post, error) {
	return s.queryPosts("WHERE p.user_id = ? ORDER BY p.created_at DESC", userID)
}

// Посты, лайкнутые пользователем
func (s *ForumService) GetLikedPosts(userID int) ([]Post, error) {
	return s.queryPosts(`
        WHERE EXISTS (
            SELECT 1 FROM votes v_like
            WHERE v_like.post_id = p.id 
              AND v_like.comment_id IS NULL 
              AND v_like.user_id = ? 
              AND v_like.value = 1
        )`, userID)
}

// ---------- Удаление поста ----------
func (s *ForumService) DeletePost(id, userID int) error {
	_, err := s.DB.Conn.Exec(
		`DELETE FROM posts WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	return err
}

// ---------- Теги ----------
func (s *ForumService) ListTags() ([]string, error) {
	rows, err := s.DB.Conn.Query(`SELECT name FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, nil
}

func (s *ForumService) AddPostTags(postID int, tags []string) error {
	for _, tag := range tags {
		// создаём тег, если его нет
		_, err := s.DB.Conn.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, tag)
		if err != nil {
			return fmt.Errorf("failed to insert tag %s: %w", tag, err)
		}

		// связываем пост и тег
		_, err = s.DB.Conn.Exec(
			`INSERT OR IGNORE INTO post_tags (post_id, tag_id)
             SELECT ?, id FROM tags WHERE name = ?`,
			postID, tag,
		)
		if err != nil {
			return fmt.Errorf("failed to link tag %s to post %d: %w", tag, postID, err)
		}
	}
	return nil
}

func (s *ForumService) GetPostsByTag(tag string) ([]Post, error) {
	return s.queryPosts(`
        WHERE EXISTS (
            SELECT 1 FROM post_tags pt
            JOIN tags t ON pt.tag_id = t.id
            WHERE pt.post_id = p.id AND t.name = ?
        )`, tag)
}

// ---------- Комментарии ----------
func (s *ForumService) GetCommentsByPostID(postID int) ([]Comment, error) {
	rows, err := s.DB.Conn.Query(`
        SELECT c.id, c.post_id, c.user_id, u.username, c.content, c.created_at,
               IFNULL(SUM(v.value), 0) as rating,
               COALESCE(SUM(CASE WHEN v.value = 1 THEN 1 ELSE 0 END), 0) AS likes,
               COALESCE(SUM(CASE WHEN v.value = -1 THEN 1 ELSE 0 END), 0) AS dislikes
        FROM comments c
        JOIN users u ON u.id = c.user_id
        LEFT JOIN votes v ON v.comment_id = c.id
        WHERE c.post_id = ?
        GROUP BY c.id
        ORDER BY c.created_at ASC
    `, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		var rating, likes, dislikes sql.NullInt64
		err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.User,
			&comment.Content, &comment.CreatedAt, &rating, &likes, &dislikes)
		if err != nil {
			return nil, err
		}
		if rating.Valid {
			comment.Rating = int(rating.Int64)
		}
		if likes.Valid {
			comment.Likes = int(likes.Int64)
		}
		if dislikes.Valid {
			comment.Dislikes = int(dislikes.Int64)
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

// ---------- Общая функция ----------
func (s *ForumService) scanPosts(rows *sql.Rows) ([]Post, error) {
	var posts []Post
	for rows.Next() {
		var post Post
		var categories sql.NullString
		var rating, likes, dislikes sql.NullInt64

		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Username,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&rating,
			&likes,
			&dislikes,
			&categories,
		)
		if err != nil {
			return nil, err
		}

		if rating.Valid {
			post.Rating = int(rating.Int64)
		}
		if likes.Valid {
			post.Likes = int(likes.Int64)
		}
		if dislikes.Valid {
			post.Dislikes = int(dislikes.Int64)
		}

		if categories.Valid && categories.String != "" {
			post.Categories = strings.Split(categories.String, ", ")
		} else {
			post.Categories = []string{}
		}

		comments, err := s.GetCommentsByPostID(post.ID)
		if err == nil {
			post.Comments = comments
		}
		posts = append(posts, post)
	}
	return posts, nil
}

// ---------- Приватные комнаты ----------
func (s *ForumService) CreateRoom(name string, isPrivate bool) (int, error) {
	res, err := s.DB.Conn.Exec(
		`INSERT INTO rooms (uuid, name, is_private, created_at) VALUES (lower(hex(randomblob(16))), ?, ?, ?)`,
		name, isPrivate, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (s *ForumService) GetRoomByID(roomID int) (*models.Room, error) {
	row := s.DB.Conn.QueryRow(`SELECT id, uuid, name, is_private, created_at FROM rooms WHERE id = ?`, roomID)
	var r models.Room
	err := row.Scan(&r.ID, &r.UUID, &r.Name, &r.IsPrivate, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ---------- Приглашения ----------
func (s *ForumService) SendInvite(roomID int, fromUser, toUser string) (int, error) {
	res, err := s.DB.Conn.Exec(
		`INSERT INTO invites (room_id, from_user, to_user, status, created_at) VALUES (?, ?, ?, 'pending', ?)`,
		roomID, fromUser, toUser, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (s *ForumService) UpdateInviteStatus(inviteID int, status string) error {
	_, err := s.DB.Conn.Exec(
		`UPDATE invites SET status = ? WHERE id = ?`,
		status, inviteID,
	)
	return err
}

func (s *ForumService) GetPendingInvites(toUser string) ([]models.Invite, error) {
	rows, err := s.DB.Conn.Query(
		`SELECT id, room_id, from_user, to_user, status, created_at 
         FROM invites WHERE to_user = ? AND status = 'pending'`,
		toUser,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []models.Invite
	for rows.Next() {
		var inv models.Invite
		err := rows.Scan(&inv.ID, &inv.RoomID, &inv.FromUser, &inv.ToUser, &inv.Status, &inv.CreatedAt)
		if err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, nil
}
