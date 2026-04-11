package middleware

import (
	"net/http"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/sqlite"
)

// CurrentUser — возвращает ID пользователя по куке session_token.
// Если куки нет или сессия не найдена/просрочена, возвращает 0.
func CurrentUser(r *http.Request, db *sqlite.DB) int {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0
	}

	var userID int
	err = db.Conn.QueryRow(`
        SELECT user_id 
        FROM sessions 
        WHERE id = ? AND expires_at > datetime('now')
    `, cookie.Value).Scan(&userID)
	if err != nil {
		return 0
	}

	return userID
}

// RequireAuth — middleware, проверяет авторизацию.
// Неавторизованным возвращает JSON 401 (SPA сам обработает).
func RequireAuth(db *sqlite.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := CurrentUser(r, db)
		if userID == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"success":false,"error":"Требуется авторизация"}`))
			return
		}
		next.ServeHTTP(w, r)
	}
}
