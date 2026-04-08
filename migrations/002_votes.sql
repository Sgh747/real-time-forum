-- 002_votes.sql
PRAGMA foreign_keys = OFF;
BEGIN;

-- Создать таблицу votes, если её ещё нет
CREATE TABLE IF NOT EXISTS votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    post_id INTEGER,
    comment_id INTEGER,
    value INTEGER NOT NULL, -- 1 = like, -1 = dislike
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(comment_id) REFERENCES comments(id) ON DELETE CASCADE,
    CHECK(value IN (1, -1))
);

-- Удаляем старые индексы (если они существуют)
DROP INDEX IF EXISTS uniq_votes_user_comment;
DROP INDEX IF EXISTS uniq_votes_user_post;
DROP INDEX IF EXISTS idx_votes_user_post;
DROP INDEX IF EXISTS idx_votes_user_comment;

-- Удаляем возможные дубликаты: оставляем последнюю запись (по id)
DELETE FROM votes
WHERE id NOT IN (
  SELECT MAX(id) FROM votes GROUP BY user_id, post_id, comment_id
);

COMMIT;
PRAGMA foreign_keys = ON;

-- Частичные уникальные индексы: разделяют голоса по посту и по комментарию
CREATE UNIQUE INDEX IF NOT EXISTS ux_votes_user_post_unique
ON votes(user_id, post_id)
WHERE comment_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_votes_user_comment_unique
ON votes(user_id, comment_id)
WHERE comment_id IS NOT NULL;

-- Вспомогательные индексы для ускорения запросов
CREATE INDEX IF NOT EXISTS idx_votes_user_post ON votes(user_id, post_id);
CREATE INDEX IF NOT EXISTS idx_votes_user_comment ON votes(user_id, comment_id);
