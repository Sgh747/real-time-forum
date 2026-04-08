package utils

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

// RenderTemplate — универсальный рендер для страниц (теперь только forum.html).
func RenderTemplate(w http.ResponseWriter, data interface{}, files ...string) {
	if len(files) == 0 {
		fmt.Fprint(w, "Неверный вызов RenderTemplate: нет шаблонов")
		return
	}
	renderStandalone(w, data, files)
}

// renderStandalone рендерит forum.html или другие standalone страницы.
func renderStandalone(w http.ResponseWriter, data interface{}, files []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	toParse := make([]string, 0, len(files))
	for _, f := range files {
		if filepath.IsAbs(f) || filepath.Dir(f) != "." {
			toParse = append(toParse, f)
		} else {
			toParse = append(toParse, filepath.Join("internal/web/templates", f))
		}
	}

	tmpl, err := template.ParseFiles(toParse...)
	if err != nil {
		log.Printf("renderStandalone: parse error: %v", err)
		fmt.Fprint(w, "Ошибка парсинга шаблона: "+err.Error())
		return
	}

	rootName := filepath.Base(files[0])
	if err := tmpl.ExecuteTemplate(w, rootName, data); err != nil {
		log.Printf("renderStandalone: execute error: %v", err)
		fmt.Fprint(w, "Ошибка рендера шаблона: "+err.Error())
	}
}
