# Step 05: Basic HTMX UI Layout

## Overview
Create the foundational HTML templates, Tailwind CSS setup, and basic layout structure using HTMX for dynamic interactions.

## Dependencies
- Step 01: Project scaffolding (requires templates directory)

## Duration Estimate
30 minutes

## Agent Prompt

You are implementing Step 05 of the FilesOnTheGo project. Your task is to create the basic UI layout and structure using HTMX and Tailwind CSS.

### Tasks

1. **Set Up Tailwind CSS**
   - Add Tailwind CLI or CDN to the project
   - Create `static/css/tailwind.config.js`
   - Create `static/css/input.css` with Tailwind directives
   - Set up build process for CSS (npm script or Go-based)

2. **Create Base Layout (templates/layouts/base.html)**
   ```html
   - DOCTYPE and HTML structure
   - Meta tags (charset, viewport, description)
   - Tailwind CSS link
   - HTMX script (from CDN or local)
   - Custom CSS link
   - Header with navigation
   - Main content area ({% block content %})
   - Footer
   - Toast notification container
   ```

3. **Create Authentication Layout (templates/layouts/auth.html)**
   - Centered card layout for login/register
   - Minimal header
   - Background gradient or pattern
   - Form container with proper spacing

4. **Create Main Application Layout (templates/layouts/app.html)**
   - Top navigation bar with:
     - Logo/app name
     - User menu (profile, settings, logout)
     - Upload button (prominent)
   - Sidebar (optional, collapsible on mobile):
     - Storage usage indicator
     - Quick links (My Files, Shared, Trash)
   - Main content area
   - Breadcrumb navigation slot

5. **Create Components**

   **templates/components/header.html:**
   - App logo and name
   - User avatar and dropdown menu
   - Responsive mobile menu toggle

   **templates/components/breadcrumb.html:**
   - Dynamic breadcrumb navigation
   - HTMX-powered navigation
   - Home icon → folders → current location

   **templates/components/toast.html:**
   - Success/error/info notification component
   - Auto-dismiss after 5 seconds
   - Close button
   - HTMX-triggered display

   **templates/components/modal.html:**
   - Reusable modal overlay
   - Close on backdrop click
   - Close button
   - Title and content slots

   **templates/components/loading.html:**
   - Loading spinner component
   - Used as HTMX indicator
   - Overlay option for full-screen loading

6. **Create Page Templates**

   **templates/pages/login.html:**
   - Email and password inputs
   - Remember me checkbox
   - Login button
   - Link to register page
   - HTMX form submission

   **templates/pages/register.html:**
   - Email, username, password inputs
   - Password confirmation
   - Terms acceptance checkbox
   - Register button
   - Link to login page
   - HTMX form submission

   **templates/pages/dashboard.html:**
   - Empty state with upload prompt
   - Storage usage card
   - Recent files list placeholder
   - Quick actions (New Folder, Upload File)

7. **Create Static Assets**

   **static/css/custom.css:**
   - Custom styles beyond Tailwind
   - Animations
   - HTMX transitions
   - Loading states

   **static/js/app.js:**
   - Minimal JavaScript utilities
   - Toast notification function
   - File size formatting
   - Date formatting
   - Clipboard copy function

8. **Configure PocketBase Template Rendering**

   Create `handlers/template_handler.go`:
   - Set up template loading and caching
   - Create render helper function
   - Add template data helpers (current user, flash messages)
   - Implement HTMX detection (check HX-Request header)

9. **Create Auth Routes (handlers/auth_handler.go)**
   - `GET /login` - Render login page
   - `POST /login` - Handle login (HTMX-aware)
   - `GET /register` - Render register page
   - `POST /register` - Handle registration (HTMX-aware)
   - `POST /logout` - Handle logout
   - All routes should detect HTMX and respond appropriately

10. **Write Tests**

    **Template Tests (tests/unit/template_test.go):**
    - Test template loading
    - Test render function
    - Test HTMX detection
    - Test data injection

    **Handler Tests (handlers/auth_handler_test.go):**
    - Test login page rendering
    - Test registration page rendering
    - Test HTMX responses vs full page responses
    - Test authentication flow

    **UI Tests (tests/integration/ui_test.go):**
    - Test navigation works
    - Test responsive design (using headless browser)
    - Test HTMX transitions
    - Test toast notifications

### Success Criteria

- [ ] Tailwind CSS properly configured and builds
- [ ] Base layouts created and render correctly
- [ ] All components created and reusable
- [ ] Authentication pages functional
- [ ] HTMX properly integrated
- [ ] Responsive design works on mobile/tablet/desktop
- [ ] Template handler works
- [ ] Auth routes functional
- [ ] All tests pass
- [ ] Code follows CLAUDE.md guidelines
- [ ] Accessibility standards met (WCAG 2.1 AA minimum)

### Testing Commands

```bash
# Build Tailwind CSS
npm run build:css
# or
tailwindcss -i ./static/css/input.css -o ./static/css/output.css

# Run the application
go run main.go serve

# Visit in browser
# http://localhost:8090/login
# http://localhost:8090/register

# Run tests
go test ./handlers/... -v
go test ./tests/integration/... -v
```

### Design Requirements

**Color Scheme:**
- Primary: Blue (#3B82F6)
- Secondary: Gray (#6B7280)
- Success: Green (#10B981)
- Warning: Yellow (#F59E0B)
- Error: Red (#EF4444)
- Background: White/Light Gray (#F9FAFB)

**Typography:**
- Font: Inter or System UI
- Headings: Bold, larger sizes
- Body: Regular weight, readable size (16px base)

**Spacing:**
- Consistent spacing scale (Tailwind defaults)
- Generous padding in containers
- Proper margin between sections

**Responsiveness:**
- Mobile-first approach
- Breakpoints: sm (640px), md (768px), lg (1024px), xl (1280px)
- Collapsible sidebar on mobile
- Touch-friendly buttons (min 44px touch target)

### HTMX Patterns

**Form Submission:**
```html
<form hx-post="/api/login" hx-target="#content" hx-swap="innerHTML">
  <!-- form fields -->
</form>
```

**Loading States:**
```html
<button hx-get="/api/files" hx-indicator="#loading">
  Load Files
</button>
<div id="loading" class="htmx-indicator">Loading...</div>
```

**Error Handling:**
```html
<div hx-on::after-request="handleResponse(event)">
  <!-- content -->
</div>
```

### References

- DESIGN.md: User Interface Design section
- CLAUDE.md: HTMX Development Guidelines
- Tailwind CSS docs: https://tailwindcss.com/docs
- HTMX docs: https://htmx.org/docs/

### Notes

- Use semantic HTML for accessibility
- Add ARIA labels where needed
- Ensure keyboard navigation works
- Test with screen readers
- Optimize for performance (lazy loading, code splitting)
- Add meta tags for SEO
- Implement CSP headers for security
- Use HTMX extensions sparingly
