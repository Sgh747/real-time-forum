CREATE TABLE IF NOT EXISTS comment_votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    comment_id INTEGER NOT NULL,
    value INTEGER NOT NULL, -- 1 = like, -1 = dislike
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, comment_id),
    FOREIGN KEY(user_id) REFERENCES users(id),
    FOREIGN KEY(comment_id) REFERENCES comments(id)
);
