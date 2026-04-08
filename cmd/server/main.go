package main

import (
	"log"
	"net/http"
	"strconv"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/middleware"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/sqlite"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/web/handlers"
	"01.tomorrow-school.ai/git/dak/forum.git/internal/web/utils"
)

func main() {
	// Подключаем миграции
	migrations := []string{
		"migrations/001_init.sql",
		"migrations/002_votes.sql",
		"migrations/003_categories.sql",
		"migrations/004_messages.sql",
		"migrations/005_rooms.sql",
		"migrations/006_invites.sql",
		"migrations/007_tags.sql",
		"migrations/008_comment_votes.sql",
	}

	db, err := sqlite.Open("real-time-forum.db", migrations)
	if err != nil {
		log.Fatalf("Ошибка открытия базы: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// Хэндлеры
	authHandler := &handlers.AuthHandler{DB: db}
	chatHandler := handlers.NewChatHandler(db)

	// Авторизация
	mux.HandleFunc("/register", authHandler.Register)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/logout", authHandler.Logout)
	mux.HandleFunc("/me", authHandler.Me)

	// Посты
	mux.HandleFunc("/posts", authHandler.ListPosts) // список постов
	mux.HandleFunc("/vote", authHandler.Vote)
	mux.HandleFunc("/delete-post", authHandler.DeletePost)
	mux.HandleFunc("/update-post", authHandler.UpdatePost)
	mux.HandleFunc("/create-post", authHandler.CreatePost) // создание поста
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			authHandler.RenderError(w, r, http.StatusBadRequest, "Ошибка", "Не указан ID поста")
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			authHandler.RenderError(w, r, http.StatusBadRequest, "Ошибка", "Некорректный ID поста")
			return
		}
		authHandler.ShowPost(w, r, id)
	})

	// Комментарии
	mux.HandleFunc("/comments", authHandler.ListComments)                              // GET для получения
	mux.HandleFunc("/add-comment", middleware.RequireAuth(db, authHandler.AddComment)) // POST для добавления
	mux.HandleFunc("/vote-comment", authHandler.VoteComment)

	// WebSocket чат
	mux.HandleFunc("/ws/chat", chatHandler.HandleConnections)
	go chatHandler.HandleMessages()

	// SPA — forum.html
	mux.HandleFunc("/forum", func(w http.ResponseWriter, r *http.Request) {
		data := authHandler.MakeLayoutData(r, nil, "Форум", "")
		utils.RenderTemplate(w, data, "forum.html")
	})

	// Статика
	fs := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	fsTemplates := http.FileServer(http.Dir("internal/web/templates"))
	mux.Handle("/templates/", http.StripPrefix("/templates/", fsTemplates))

	addr := ":8080"
	log.Printf("Сервер запущен: http://localhost%s/forum#home", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
