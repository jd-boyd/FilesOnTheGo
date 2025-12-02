package handlers

import (
	"errors"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TemplateData holds common data passed to templates
type TemplateData struct {
	Title              string
	User               interface{}
	FlashMessage       string
	FlashType          string
	Success            string
	Error              string
	Breadcrumb         []BreadcrumbItem
	StorageUsed        string
	StorageQuota       string
	StoragePercent     int
	HasFiles           bool
	RecentActivity     []ActivityItem
	PublicRegistration bool
	Settings           map[string]interface{}
}

// BreadcrumbItem represents a breadcrumb navigation item
type BreadcrumbItem struct {
	Name string
	URL  string
}

// ActivityItem represents a recent activity entry
type ActivityItem struct {
	FileName string
	Action   string
	Time     string
}

// TemplateRenderer handles template rendering with caching
type TemplateRenderer struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
	baseDir   string
	fsys      fs.FS // filesystem for loading templates (nil means use baseDir)
}

// NewTemplateRenderer creates a new template renderer using filesystem path
func NewTemplateRenderer(baseDir string) *TemplateRenderer {
	return &TemplateRenderer{
		templates: make(map[string]*template.Template),
		baseDir:   baseDir,
	}
}

// NewTemplateRendererFromFS creates a new template renderer using an fs.FS
func NewTemplateRendererFromFS(fsys fs.FS) *TemplateRenderer {
	return &TemplateRenderer{
		templates: make(map[string]*template.Template),
		fsys:      fsys,
	}
}

// getTemplateFuncs returns custom template functions
func getTemplateFuncs() template.FuncMap {
	caser := cases.Title(language.English)
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": caser.String,
	}
}

// LoadTemplates loads all templates from the templates directory
func (r *TemplateRenderer) LoadTemplates() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Define template files to load (paths are relative to templates root when using fs.FS)
	templates := map[string][]string{
		"login": {
			"layouts/base.html",
			"layouts/auth.html",
			"pages/login.html",
		},
		"register": {
			"layouts/base.html",
			"layouts/auth.html",
			"pages/register.html",
		},
		"dashboard": {
			"layouts/base.html",
			"layouts/app.html",
			"pages/dashboard.html",
			"components/loading.html",
			"components/header.html",
			"components/breadcrumb.html",
		},
		"files": {
			"layouts/base.html",
			"layouts/app.html",
			"pages/files.html",
			"components/header.html",
		},
		"settings": {
			"layouts/base.html",
			"layouts/app.html",
			"pages/settings.html",
			"components/header.html",
		},
		"profile": {
			"layouts/base.html",
			"layouts/app.html",
			"pages/profile.html",
			"components/header.html",
		},
		"admin": {
			"layouts/base.html",
			"layouts/app.html",
			"pages/admin.html",
			"components/header.html",
		},
		"shares": {
			"layouts/base.html",
			"layouts/app.html",
			"pages/shares.html",
			"components/header.html",
		},
	}

	// Load each template set
	for name, files := range templates {
		if err := r.loadTemplate(name, files); err != nil {
			return err
		}
	}

	return nil
}

// loadTemplate loads a specific template by name
func (r *TemplateRenderer) loadTemplate(name string, files []string) error {
	var tmpl *template.Template
	var err error

	if r.fsys != nil {
		// Load from fs.FS
		tmpl, err = r.loadTemplateFromFS(files)
	} else {
		// Load from baseDir
		tmpl, err = r.loadTemplateFromDir(files)
	}

	if err != nil {
		return err
	}

	r.templates[name] = tmpl
	return nil
}

// loadTemplateFromFS loads templates from fs.FS
func (r *TemplateRenderer) loadTemplateFromFS(files []string) (*template.Template, error) {
	// Create new template with functions
	tmpl := template.New("").Funcs(getTemplateFuncs())

	// Parse each file from fs.FS
	for _, file := range files {
		content, err := fs.ReadFile(r.fsys, file)
		if err != nil {
			return nil, err
		}

		_, err = tmpl.New(file).Parse(string(content))
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

// loadTemplateFromDir loads templates from directory
func (r *TemplateRenderer) loadTemplateFromDir(files []string) (*template.Template, error) {
	// Create full paths
	fullPaths := make([]string, len(files))
	for i, file := range files {
		fullPaths[i] = filepath.Join(r.baseDir, file)
	}

	// Parse templates
	return template.New("").Funcs(getTemplateFuncs()).ParseFiles(fullPaths...)
}

// Render renders a template by name
func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}) error {
	r.mu.RLock()
	tmpl, exists := r.templates[name]
	r.mu.RUnlock()

	if !exists {
		return errors.New("template not found: " + name)
	}

	// Execute the template with the base layout
	return tmpl.ExecuteTemplate(w, "layouts/base.html", data)
}

// Helper functions for handlers

// IsHTMXRequest checks if the request is an HTMX request
func IsHTMXRequest(c *gin.Context) bool {
	return c.GetHeader("HX-Request") == "true"
}

// GetAuthUser extracts the authenticated user from the request context
func GetAuthUser(c *gin.Context) interface{} {
	claims, err := auth.GetUserClaims(c)
	if err != nil {
		return nil
	}
	return claims
}

// PrepareTemplateData creates a TemplateData struct with common fields populated
func PrepareTemplateData(c *gin.Context) *TemplateData {
	data := &TemplateData{
		Settings: make(map[string]interface{}),
	}

	// Get authenticated user
	user := GetAuthUser(c)
	if user != nil {
		data.User = user

		// Get admin status from claims
		if claims, ok := user.(*auth.JWTClaims); ok {
			data.Settings["IsAdmin"] = claims.IsAdmin
		}
	}

	return data
}
