package handlers

import (
	"log"
	"net/http"
	"time"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/service"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/web/utils"
)

// Logout удаляет сессию из базы и очищает cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		// Нет cookie → показываем popup
		data := h.MakeLayoutData(r, nil, "Выход", "Сессия не найдена")
		utils.RenderTemplate(w, data, "forum.html")
		return
	}

	sessionToken := cookie.Value
	authService := service.AuthService{DB: h.DB}
	if err := authService.Logout(sessionToken); err != nil {
		log.Println("Ошибка при удалении сессии:", err)
		h.RenderError(w, r, http.StatusInternalServerError, "Ошибка 500", "Ошибка при удалении сессии")
		return
	}

	// Успех → очищаем cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login#login", http.StatusSeeOther)
}

// CurrentUserID возвращает ID текущего пользователя по session_token.
// Если пользователь не авторизован или сессия истекла → возвращает 0.
func (h *AuthHandler) CurrentUserID(r *http.Request) int {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0 // нет cookie
	}

	var userID int
	err = h.DB.Conn.QueryRow(
		`SELECT user_id FROM sessions WHERE id = ? AND expires_at > datetime('now')`,
		cookie.Value,
	).Scan(&userID)
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Println("Ошибка при проверке сессии:", err)
		}
		return 0
	}
	return userID
}
