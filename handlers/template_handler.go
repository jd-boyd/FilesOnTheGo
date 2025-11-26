package handlers

import (
	"errors"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pocketbase/pocketbase/core"
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
			"components/header.html",
			"components/breadcrumb.html",
			"components/loading.html",
			"pages/dashboard.html",
		},
		"settings": {
			"layouts/base.html",
			"layouts/app.html",
			"components/header.html",
			"components/breadcrumb.html",
			"components/loading.html",
			"pages/settings.html",
		},
		"admin": {
			"layouts/base.html",
			"layouts/app.html",
			"components/header.html",
			"components/breadcrumb.html",
			"components/loading.html",
			"pages/admin.html",
		},
		"profile": {
			"layouts/base.html",
			"layouts/app.html",
			"components/header.html",
			"components/breadcrumb.html",
			"components/loading.html",
			"pages/profile.html",
		},
	}

	// Load each template set
	for name, files := range templates {
		var tmpl *template.Template
		var err error

		if r.fsys != nil {
			// Load from fs.FS
			tmpl = template.New(name).Funcs(getTemplateFuncs())
			tmpl, err = tmpl.ParseFS(r.fsys, files...)
		} else {
			// Load from filesystem path (legacy mode)
			fullPaths := make([]string, len(files))
			for i, file := range files {
				fullPaths[i] = filepath.Join(r.baseDir, "templates", file)
			}
			tmpl = template.New(name).Funcs(getTemplateFuncs())
			tmpl, err = tmpl.ParseFiles(fullPaths...)
		}

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
