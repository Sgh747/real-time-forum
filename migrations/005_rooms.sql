-- Таблица приватных и публичных комнат
CREATE TABLE IF NOT EXISTS rooms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    is_private BOOLEAN NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    creator_id INTEGER REFERENCES users(id)
);

-- Таблица связей пользователей и комнат
CREATE TABLE IF NOT EXISTS rooms_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(room_id, user_id) -- чтобы один пользователь не добавлялся дважды
);

-- Создаём комнату "Общий чат" с id = 1
INSERT INTO rooms (id, uuid, name, is_private, created_at)
VALUES (1, lower(hex(randomblob(16))), 'Общий чат', 0, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING;
