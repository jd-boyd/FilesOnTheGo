package handlers

import (
	"errors"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TemplateData holds common data passed to templates
type TemplateData struct {
	Title          string
	User           interface{}
	FlashMessage   string
	FlashType      string
	Breadcrumb     []BreadcrumbItem
	StorageUsed    string
	StorageQuota   string
	StoragePercent int
	HasFiles       bool
	RecentActivity []ActivityItem
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
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer(baseDir string) *TemplateRenderer {
	return &TemplateRenderer{
		templates: make(map[string]*template.Template),
		baseDir:   baseDir,
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

	// Define template files to load
	templates := map[string][]string{
		"login": {
			"templates/layouts/base.html",
			"templates/layouts/auth.html",
			"templates/pages/login.html",
		},
		"register": {
			"templates/layouts/base.html",
			"templates/layouts/auth.html",
			"templates/pages/register.html",
		},
		"dashboard": {
			"templates/layouts/base.html",
			"templates/layouts/app.html",
			"templates/components/header.html",
			"templates/components/breadcrumb.html",
			"templates/components/loading.html",
			"templates/pages/dashboard.html",
		},
	}

	// Load each template set
	for name, files := range templates {
		// Prepend base directory to file paths
		fullPaths := make([]string, len(files))
		for i, file := range files {
			fullPaths[i] = filepath.Join(r.baseDir, file)
		}

		// Parse templates with custom functions
		tmpl := template.New(name).Funcs(getTemplateFuncs())
		tmpl, err := tmpl.ParseFiles(fullPaths...)
		if err != nil {
			return err
		}

		r.templates[name] = tmpl
	}

	return nil
}

// Render renders a template with the given data
func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, exists := r.templates[name]
	if !exists {
		// Template not found, try to load it
		r.mu.RUnlock()
		if err := r.LoadTemplates(); err != nil {
			r.mu.RLock()
			return err
		}
		r.mu.RLock()
		tmpl = r.templates[name]
		if tmpl == nil {
			return errors.New("template not found: " + name)
		}
	}

	// Execute the base.html template which is the entry point
	return tmpl.ExecuteTemplate(w, "base.html", data)
}

// IsHTMXRequest checks if the request is from HTMX
func IsHTMXRequest(c *core.RequestEvent) bool {
	return c.Request.Header.Get("HX-Request") == "true"
}

// GetAuthUser extracts the authenticated user from the request context
func GetAuthUser(c *core.RequestEvent) interface{} {
	auth := c.Get("authRecord")
	return auth
}

// PrepareTemplateData creates a TemplateData struct with common fields populated
func PrepareTemplateData(c *core.RequestEvent) *TemplateData {
	return &TemplateData{
		User: GetAuthUser(c),
	}
}
