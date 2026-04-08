package service

import (
	"database/sql"
	"fmt"
	"time"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/sqlite"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	DB *sqlite.DB
}

// Регистрация нового пользователя
func (s *AuthService) RegisterUser(email, username, password, firstName, lastName string, age int, gender string) error {
	// Проверка уникальности email
	var exists int
	err := s.DB.Conn.QueryRow(`SELECT COUNT(*) FROM users WHERE email = ?`, email).Scan(&exists)
	if err != nil {
		return err
	}
	if exists > 0 {
		return fmt.Errorf("email уже зарегистрирован")
	}

	// Проверка уникальности username
	err = s.DB.Conn.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&exists)
	if err != nil {
		return err
	}
	if exists > 0 {
		return fmt.Errorf("имя пользователя уже занято")
	}

	// Хэшируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Сохраняем пользователя
	_, err = s.DB.Conn.Exec(
		`INSERT INTO users (email, username, password_hash, first_name, last_name, age, gender)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		email, username, string(hash), firstName, lastName, age, gender,
	)
	return err
}

// Логин пользователя
func (s *AuthService) LoginUser(email, password string) (string, int, error) {
	var id int
	var hash string
	err := s.DB.Conn.QueryRow(
		`SELECT id, password_hash FROM users WHERE email = ?`, email,
	).Scan(&id, &hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, fmt.Errorf("пользователь не найден")
		}
		return "", 0, err
	}

	// Проверка пароля
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", 0, fmt.Errorf("неверный пароль")
	}

	// Создаём новую сессию
	sessionID := uuid.New().String()
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02 15:04:05")

	_, err = s.DB.Conn.Exec(
		`INSERT OR REPLACE INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		sessionID, id, expiresAt,
	)
	if err != nil {
		return "", 0, err
	}

	return sessionID, id, nil
}

// Логаут пользователя
func (s *AuthService) Logout(sessionToken string) error {
	_, err := s.DB.Conn.Exec(`DELETE FROM sessions WHERE id = ?`, sessionToken)
	return err
}
