-- Таблица сообщений (общий чат + приватные)

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE,              -- уникальный идентификатор сообщения
    sender_id INTEGER NOT NULL,             -- id отправителя (FK -> users.id)
    receiver_id INTEGER,                    -- id получателя (FK -> users.id), NULL = общий чат
    content TEXT NOT NULL,                  -- текст сообщения
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    room_id INTEGER NOT NULL DEFAULT 0,     -- id комнаты (0 = общий чат)

    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (receiver_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- Индексы для ускорения выборки
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_receiver ON messages(receiver_id);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_room ON messages(room_id);
