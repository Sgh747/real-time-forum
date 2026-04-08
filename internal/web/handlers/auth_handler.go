package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/middleware"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/service"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/sqlite"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/web/utils"
)

type AuthHandler struct {
	DB *sqlite.DB
}

// MakeLayoutData — собирает данные для forum.html и standalone страниц
func (h *AuthHandler) MakeLayoutData(r *http.Request, content interface{}, title, errMsg string) service.LayoutData {
	userID := middleware.CurrentUser(r, h.DB)

	var username string
	if userID != 0 {
		_ = h.DB.Conn.QueryRow(`SELECT username FROM users WHERE id = ?`, userID).Scan(&username)
	}

	// Загружаем категории через сервис
	forumService := service.ForumService{DB: h.DB}
	categories, _ := forumService.ListCategories()

	return service.LayoutData{
		Title:      title,
		Error:      errMsg,
		IsAuth:     userID != 0,
		Username:   username,
		Content:    content,
		Code:       0,
		BgClass:    "",
		CurrentURL: r.RequestURI,
		UserID:     userID,
		Categories: categories,
	}
}

// универсальный вывод ошибки в forum.html
func (h *AuthHandler) RenderError(w http.ResponseWriter, r *http.Request, status int, title, msg string) {
	w.WriteHeader(status)
	data := h.MakeLayoutData(r, nil, title, msg)
	utils.RenderTemplate(w, data, "forum.html")
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"success":false,"error":"%s"}`, msg)
}

func writeJSONSuccess(w http.ResponseWriter, redirect string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success":true,"redirect":"%s"}`, redirect)
}

func writeJSONSuccessWithUser(w http.ResponseWriter, redirect, username string, userID int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w,
		`{"success":true,"redirect":"%s","username":"%s","userId":%d}`,
		redirect, username, userID)
}

// ✅ Домашняя страница
func (h *AuthHandler) Home(w http.ResponseWriter, r *http.Request) {
	data := h.MakeLayoutData(r, nil, "Главная", "")
	utils.RenderTemplate(w, data, "forum.html")
}

// ✅ Регистрация
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		firstName := strings.TrimSpace(r.FormValue("first_name"))
		lastName := strings.TrimSpace(r.FormValue("last_name"))
		ageStr := r.FormValue("age")
		gender := r.FormValue("gender")
		age, _ := strconv.Atoi(ageStr)

		email := r.FormValue("email")
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		// Валидация
		parts := strings.Split(email, "@")
		if len(parts) != 2 || !strings.Contains(parts[1], ".") {
			writeJSONError(w, http.StatusBadRequest, "Некорректный email")
			return
		}
		if username == "" || len(password) < 6 {
			writeJSONError(w, http.StatusBadRequest, "Некорректные данные")
			return
		}

		authService := service.AuthService{DB: h.DB}
		if err := authService.RegisterUser(email, username, password, firstName, lastName, age, gender); err != nil {
			writeJSONError(w, http.StatusConflict, "Email или имя заняты")
			return
		}

		// успех → JSON с redirect
		writeJSONSuccess(w, "/login#login")
		return
	}

	// для GET или других методов — рендерим HTML
	h.RenderError(w, r, http.StatusMethodNotAllowed, "Ошибка 405", "Метод не поддерживается")
}

// ✅ Логин
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		password := r.FormValue("password")

		authService := service.AuthService{DB: h.DB}
		sessionID, userID, err := authService.LoginUser(email, password)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Неверный email или пароль")
			return
		}

		// сохраняем session_token в cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionID,
			Path:     "/",                            // доступна для всех путей
			Expires:  time.Now().Add(24 * time.Hour), // срок действия
			HttpOnly: true,                           // защищает от JS
			SameSite: http.SameSiteLaxMode,           // кука отправляется на /ws/chat
		})

		var username string
		_ = h.DB.Conn.QueryRow(`SELECT username FROM users WHERE id = ?`, userID).Scan(&username)

		log.Printf("Пользователь %d успешно вошёл", userID)

		// успех → JSON с redirect и данными пользователя
		writeJSONSuccessWithUser(w, "/login#posts", username, userID)
		return
	}

	// для GET или других методов — рендерим HTML
	h.RenderError(w, r, http.StatusMethodNotAllowed, "Ошибка 405", "Метод не поддерживается")
}

// ✅ Показ поста
func (h *AuthHandler) ShowPost(w http.ResponseWriter, r *http.Request, id int) {
	forumService := service.ForumService{DB: h.DB}
	post, err := forumService.GetPostByID(id)
	if err != nil {
		h.RenderError(w, r, http.StatusInternalServerError, "Ошибка 500", "Ошибка сервера")
		return
	}
	if post == nil {
		h.RenderError(w, r, http.StatusNotFound, "Ошибка 404", "Пост не найден")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

// ✅ Создание поста
func (h *AuthHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.CurrentUser(r, h.DB)
	if userID == 0 {
		http.Error(w, "Вы должны войти, чтобы создать пост", http.StatusForbidden)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	contentStr := strings.TrimSpace(r.FormValue("content"))
	if title == "" || contentStr == "" {
		http.Error(w, "Пост не может быть пустым", http.StatusBadRequest)
		return
	}

	// Проверка: хотя бы одна категория обязательна
	selectedRaw := r.Form["categories"]
	if len(selectedRaw) == 0 {
		http.Error(w, "Выберите хотя бы одну категорию", http.StatusBadRequest)
		return
	}

	forumService := service.ForumService{DB: h.DB}
	postID, err := forumService.CreatePost(userID, title, contentStr)
	if err != nil {
		http.Error(w, "Не удалось создать пост", http.StatusInternalServerError)
		return
	}

	// категории из формы
	selected := r.Form["categories"]
	var categoryIDs []int
	for _, idStr := range selected {
		if id, err := strconv.Atoi(idStr); err == nil {
			categoryIDs = append(categoryIDs, id)
		}
	}
	if len(categoryIDs) > 0 {
		_ = forumService.AddPostCategories(postID, categoryIDs)
	}

	// Возвращаем JSON с новым постом
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         postID,
		"title":      title,
		"content":    contentStr,
		"categories": categoryIDs,
	})
}

// ✅ Редактирование поста
func (h *AuthHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.CurrentUser(r, h.DB)
	if userID == 0 {
		http.Error(w, "Не авторизован", http.StatusForbidden)
		return
	}

	postID, _ := strconv.Atoi(r.FormValue("id"))
	post, err := h.DB.GetPostByID(postID)
	if err != nil {
		http.Error(w, "Пост не найден", http.StatusNotFound)
		return
	}
	if post.UserID != userID {
		http.Error(w, "Нет прав на редактирование", http.StatusForbidden)
		return
	}

	post.Title = strings.TrimSpace(r.FormValue("title"))
	post.Content = strings.TrimSpace(r.FormValue("content"))

	// категории приходят как "1,2,3"
	catStr := r.FormValue("categories")
	var categoryIDs []int
	for _, s := range strings.Split(catStr, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, _ := strconv.Atoi(s)
		categoryIDs = append(categoryIDs, id)
	}

	// обновляем сам пост
	if err := h.DB.UpdatePost(post); err != nil {
		http.Error(w, "Ошибка обновления поста", http.StatusInternalServerError)
		return
	}

	// обновляем категории
	if len(categoryIDs) > 0 {
		if err := h.DB.UpdatePostCategories(post.ID, categoryIDs); err != nil {
			http.Error(w, "Ошибка обновления категорий", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.RenderError(w, r, http.StatusMethodNotAllowed, "Ошибка 405", "Метод не поддерживается")
		return
	}

	userID := middleware.CurrentUser(r, h.DB)
	if userID == 0 {
		h.RenderError(w, r, http.StatusForbidden, "Ошибка 403", "Авторизуйтесь, чтобы удалять посты")
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	forumService := service.ForumService{DB: h.DB}
	if err := forumService.DeletePost(id, userID); err != nil {
		h.RenderError(w, r, http.StatusInternalServerError, "Ошибка 500", "Не удалось удалить пост")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Добавление комментариев
func (h *AuthHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.RenderError(w, r, http.StatusMethodNotAllowed, "Ошибка 405", "Метод не поддерживается")
		return
	}

	userID := middleware.CurrentUser(r, h.DB)
	forumService := service.ForumService{DB: h.DB}
	if userID == 0 {
		http.Error(w, "Авторизуйтесь, чтобы комментировать", http.StatusForbidden)
		return
	}

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil {
		http.Error(w, "Некорректный ID поста", http.StatusBadRequest)
		return
	}

	comment := strings.TrimSpace(r.FormValue("content"))
	if comment == "" {
		http.Error(w, "Комментарий не может быть пустым", http.StatusBadRequest)
		return
	}

	if err := forumService.AddComment(postID, userID, comment); err != nil {
		http.Error(w, "Не удалось добавить комментарий", http.StatusInternalServerError)
		return
	}

	// Если клиент ожидает JSON (fetch)
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"post_id": postID,
		})
		return
	}

	// иначе — старое поведение с редиректом
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = fmt.Sprintf("/login#posts?posts-id=%d", postID)
	} else {
		if u, err := url.Parse(redirect); err == nil {
			u.Fragment = ""
			redirect = u.String()
		}
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// ✅ Список комментариев для поста
func (h *AuthHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	postIDStr := r.URL.Query().Get("post_id")
	if postIDStr == "" {
		h.RenderError(w, r, http.StatusBadRequest, "Ошибка 400", "Не указан ID поста")
		return
	}
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		h.RenderError(w, r, http.StatusBadRequest, "Ошибка 400", "Некорректный ID поста")
		return
	}

	forumService := service.ForumService{DB: h.DB}
	comments, err := forumService.GetCommentsByPostID(postID)
	if err != nil {
		h.RenderError(w, r, http.StatusInternalServerError, "Ошибка 500", "Не удалось получить комментарии")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func (h *AuthHandler) VoteComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.RenderError(w, r, http.StatusMethodNotAllowed, "Ошибка 405", "Метод не поддерживается")
		return
	}

	userID := middleware.CurrentUser(r, h.DB)
	if userID == 0 {
		http.Error(w, "Авторизуйтесь, чтобы голосовать", http.StatusForbidden)
		return
	}

	commentID, _ := strconv.Atoi(r.FormValue("comment_id"))
	value, _ := strconv.Atoi(r.FormValue("value"))

	if value != 1 && value != -1 {
		http.Error(w, "Некорректное значение голоса", http.StatusBadRequest)
		return
	}

	if err := h.DB.VoteComment(userID, commentID, value); err != nil {
		http.Error(w, "Ошибка голосования", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Проверка авторизации и возврат данных пользователя
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.CurrentUser(r, h.DB)
	if userID == 0 {
		writeJSONError(w, http.StatusUnauthorized, "Не авторизован")
		return
	}

	var username string
	_ = h.DB.Conn.QueryRow(`SELECT username FROM users WHERE id = ?`, userID).Scan(&username)

	writeJSONSuccessWithUser(w, "/me", username, userID)
}
