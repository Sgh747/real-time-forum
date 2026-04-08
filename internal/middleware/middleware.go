package middleware

import (
	"log"
	"net/http"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/service"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/sqlite"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/web/utils"
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

// RequireAuth — middleware, который проверяет авторизацию.
// Для неавторизованных пользователей рендерит popup с 403.
// Для авторизованных пропускает запрос дальше.
func RequireAuth(db *sqlite.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := CurrentUser(r, db)
		if userID == 0 {
			msg := "Доступ запрещён. Авторизуйтесь, чтобы использовать эту функцию."

			// Загружаем контекст (список постов) для корректного рендера popup
			fs := service.ForumService{DB: db}
			posts, err := fs.ListPosts()
			if err != nil {
				log.Printf("RequireAuth: failed to load posts for forbidden popup: %v", err)
				posts = nil
			}

			// Рендерим popup 403 и сразу возвращаемся.
			utils.RenderForbiddenPopup(w, posts, msg, "forum.html")
			return
		}

		next.ServeHTTP(w, r)
	}
}
