-- миграция для категорий постов
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

-- связь постов и категорий (многие-ко-многим)
CREATE TABLE IF NOT EXISTS post_categories (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, category_id)
);

-- индекс для ускорения фильтрации постов по категориям
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name ON categories(name);
CREATE INDEX IF NOT EXISTS idx_post_categories_post ON post_categories(post_id);
CREATE INDEX IF NOT EXISTS idx_post_categories_category ON post_categories(category_id);

-- уникальный индекс в миграции для того, чтобы категории не дублировались
CREATE UNIQUE INDEX IF NOT EXISTS ux_post_category_unique
ON post_categories(post_id, category_id);


-- категории
INSERT OR IGNORE INTO categories (name) VALUES
    ('Музыка'),
    ('Авторское'),
    ('Фильмы'),
    ('Игры'),
    ('Спорт'),
    ('Автоспорт'),
    ('Авто'),
    ('Технологии'),
    ('Тех.обслуживание'),
    ('Наука'),
    ('Книги'),
    ('Путешествия'),
    ('Еда'),
    ('Новости'),
    ('Другое');
