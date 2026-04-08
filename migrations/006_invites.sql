-- Таблица приглашений в комнаты
CREATE TABLE IF NOT EXISTS invites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    from_user TEXT NOT NULL,
    to_user TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending / accepted / declined
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);
