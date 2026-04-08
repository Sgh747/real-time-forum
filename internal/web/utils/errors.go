package utils

import (
	"log"
	"net/http"

	"01.tomorrow-school.ai/git/dak/forum.git/internal/repo/service"
)

// RenderError рендерит standalone страницу ошибки и выставляет HTTP статус.
// Используется для 400/404/500 и других критических ошибок.
func RenderError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)

	data := service.LayoutData{
		Title:   "Ошибка",
		Error:   msg,
		Code:    code,
		Content: nil,
	}

	// Рендерим standalone forum.html
	// Важно: RenderTemplate не должен сам вызывать WriteHeader
	// и должен безопасно выполнять шаблон.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("RenderError: panic while rendering template: %v", r)
			_, _ = w.Write([]byte(msg))
		}
	}()
	RenderTemplate(w, data, "forum.html")
}

// RenderForbiddenPopup выставляет статус 403 и рендерит указанные шаблоны,
// например: "forum.html".
// Если templateFiles пуст, используются дефолтные шаблоны для списка постов + popup.
func RenderForbiddenPopup(w http.ResponseWriter, content interface{}, message string, templateFiles ...string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden) // обязательно ДО рендера

	// Дефолтные шаблоны, если не переданы
	if len(templateFiles) == 0 {
		templateFiles = []string{"forum.html"}
	}

	data := service.LayoutData{
		Title:   "Доступ запрещён",
		Error:   message,
		Code:    http.StatusForbidden,
		Content: content,
	}

	// Рендерим шаблоны. RenderTemplate не должен вызывать WriteHeader.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("RenderForbiddenPopup: panic while rendering template: %v", r)
			_, _ = w.Write([]byte(message))
		}
	}()
	RenderTemplate(w, data, templateFiles...)
}
