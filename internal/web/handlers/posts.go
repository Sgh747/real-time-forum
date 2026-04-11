package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/middleware"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/service"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/web/utils"
)

// Получение последних постов
func (h *AuthHandler) GetRecentPosts() []service.Post {
	forumService := service.ForumService{DB: h.DB}
	posts, err := forumService.GetRecentPosts(10)
	if err != nil {
		return nil
	}
	return posts
}

// Список постов с фильтрацией
func (h *AuthHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	forumService := service.ForumService{DB: h.DB}

	// ✅ Проверка параметра для показа одного поста
	idStr := r.URL.Query().Get("posts-id")
	if idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			h.RenderError(w, r, http.StatusBadRequest, "Ошибка 400", "Некорректный параметр posts-id")
			return
		}
		h.ShowPost(w, r, id)
		return
	}

	// Фильтры
	category := r.URL.Query().Get("category")
	mine := r.URL.Query().Get("mine")
	liked := r.URL.Query().Get("liked")
	sortParam := r.URL.Query().Get("sort")

	var posts []service.Post
	var err error

	if category != "" {
		posts, err = forumService.GetPostsByCategory(category)
	} else if mine != "" {
		userID := middleware.CurrentUser(r, h.DB)
		if userID == 0 {
			h.RenderError(w, r, http.StatusForbidden, "Ошибка 403", "Авторизуйтесь, чтобы видеть свои посты")
			return
		}
		posts, err = forumService.GetPostsByUser(userID)
	} else if liked != "" {
		userID := middleware.CurrentUser(r, h.DB)
		if userID == 0 {
			h.RenderError(w, r, http.StatusForbidden, "Ошибка 403", "Авторизуйтесь, чтобы видеть понравившиеся посты")
			return
		}
		posts, err = forumService.GetLikedPosts(userID)
	} else if sortParam == "rating_desc" {
		posts, err = forumService.GetPostsByRatingDesc()
	} else if sortParam == "rating_asc" {
		posts, err = forumService.GetPostsByRatingAsc()
	} else {
		posts, err = forumService.ListPosts()
	}

	if err != nil {
		h.RenderError(w, r, http.StatusInternalServerError, "Ошибка 500", "Ошибка при получении постов")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

// Проверка локального редиректа
func isLocalRedirect(u string) bool {
	if u == "" {
		return false
	}
	if strings.HasPrefix(u, "/") && !strings.Contains(u, "://") {
		return true
	}
	return false
}

// Голосование (лайк/дизлайк)
func (h *AuthHandler) Vote(w http.ResponseWriter, r *http.Request) {
	log.Printf("Vote method=%s, post_id=%s, comment_id=%s, value=%s",
		r.Method, r.FormValue("post_id"), r.FormValue("comment_id"), r.FormValue("value"))

	if r.Method != http.MethodPost {
		h.RenderError(w, r, http.StatusMethodNotAllowed, "Ошибка 405", "Метод не поддерживается")
		return
	}

	forumService := service.ForumService{DB: h.DB}
	userID := middleware.CurrentUser(r, h.DB)

	// Не авторизован → 403
	if userID == 0 {
		posts := h.GetRecentPosts()
		utils.RenderForbiddenPopup(w, posts, "Доступ запрещён. Авторизуйтесь, чтобы голосовать.", "forum.html")
		return
	}

	postID, _ := strconv.Atoi(r.FormValue("post_id"))
	commentID, _ := strconv.Atoi(r.FormValue("comment_id"))
	value, _ := strconv.Atoi(r.FormValue("value"))

	// Валидация голоса
	if value != 1 && value != -1 {
		posts := h.GetRecentPosts()
		data := h.MakeLayoutData(r, posts, "Список постов", "Некорректное значение голоса")
		utils.RenderTemplate(w, data, "forum.html")
		return
	}

	if err := forumService.AddVote(userID, postID, commentID, value); err != nil {
		log.Printf("Vote error: %v", err)
		h.RenderError(w, r, http.StatusInternalServerError, "Ошибка 500", "Ошибка при голосовании")
		return
	}

	// БАГ ФИКС: фронт использует fetch и ждёт JSON, а не redirect
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}