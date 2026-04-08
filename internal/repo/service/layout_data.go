// internal/repo/service/layout_data.go
package service

type LayoutData struct {
	Title      string
	Error      string
	IsAuth     bool
	Username   string
	Content    interface{}
	Code       int    // для forum.html
	BgClass    string // опционально для фоновых классов
	CurrentURL string // CurrentURL содержит RequestURI текущего запроса (path + query)
	UserID     int
	Categories []Category
}
