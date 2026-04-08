package models

import "time"

type Post struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`  // соответствует posts.user_id
	Username  string    `json:"username"` // берём из users.username, отдаём как "author"
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`

	Votes    int `json:"votes"` // SUM(v.value)
	Likes    int `json:"likes"`
	Dislikes int `json:"dislikes"`

	Categories string   `json:"categories"` // строка из GROUP_CONCAT
	Tags       []string `json:"tags,omitempty"`
}

// Категория поста
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Тег поста
type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Голос за пост
type Vote struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	PostID    int       `json:"post_id"`
	Value     int       `json:"value"` // 1 = like, -1 = dislike
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID        int       `json:"id"`
	PostID    int       `json:"post_id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Likes     int       `json:"likes"`
	Dislikes  int       `json:"dislikes"`
}
