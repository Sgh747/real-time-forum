PRAGMA foreign_keys = ON;

-- таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,       -- уникальный email
    username TEXT NOT NULL UNIQUE,    -- уникальное имя пользователя
    password_hash TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    age INTEGER CHECK(age >= 0),
    gender TEXT CHECK(gender IN ('male','female','other')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- таблица сессий
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,              -- UUID токен
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id)                   -- гарантируем, что у одного пользователя только одна активная сессия
);

-- таблица постов
CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL CHECK(length(title) <= 200),      -- максимум 200 символов
    content TEXT NOT NULL CHECK(length(content) <= 2000), -- максимум 2000 символов
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- таблица комментариев
CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL CHECK(length(content) <= 500),  -- максимум 500 символов
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
