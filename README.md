# FilesOnTheGo

A self-hosted file storage and sharing service built with PocketBase (Go), S3-compatible storage, HTMX, and Tailwind CSS.

## Features

- **Self-hosted**: Full control over your data
- **S3-compatible storage**: Works with MinIO, AWS S3, Backblaze B2, and more
- **Secure file sharing**: Create shareable links with permissions and expiration
- **User management**: Built-in authentication and user quotas
- **Modern UI**: HTMX for dynamic interactions with minimal JavaScript
- **RESTful API**: PocketBase-powered API for integration

## Quick Start

### Prerequisites

- Go 1.21 or higher
- S3-compatible storage (MinIO, AWS S3, etc.)
- Git

### Installation

1. Clone the repository:
```bash
git clone https://github.com/jd-boyd/filesonthego.git
cd FilesOnTheGo
```

2. Copy the example environment file and configure:
```bash
cp .env.example .env
# Edit .env with your S3 credentials and configuration
```

3. Install dependencies:
```bash
go mod download
```

4. Build the application:
```bash
go build -o filesonthego main.go
```

5. Run the application:
```bash
./filesonthego serve
```

The application will be available at `http://localhost:8090`

## Development Setup

### Running in Development Mode

```bash
# Set environment to development
export APP_ENVIRONMENT=development

# Run directly with go
go run main.go serve
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Project Structure

```
FilesOnTheGo/
├── main.go                    # Application entry point
├── config/                    # Configuration management
├── handlers/                  # HTTP request handlers
├── services/                  # Business logic layer
├── models/                    # Data models and types
├── middleware/                # Custom middleware
├── templates/                 # HTMX HTML templates
│   ├── layouts/
│   ├── components/
│   └── pages/
├── static/                    # Static assets (CSS, JS, icons)
├── tests/                     # Test files
│   ├── integration/
│   ├── unit/
│   └── fixtures/
└── migrations/                # Database migrations
```

## Configuration

All configuration is done via environment variables. See `.env.example` for all available options.

### Required Configuration

```bash
# S3 Storage
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=filesonthego
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin

# Application
APP_URL=http://localhost:8090
```

### Optional Configuration

- `APP_PORT`: HTTP server port (default: 8090)
- `APP_ENVIRONMENT`: development or production (default: development)
- `MAX_UPLOAD_SIZE`: Maximum file upload size in bytes (default: 100MB)
- `DEFAULT_USER_QUOTA`: Storage quota per user (default: 10GB)
- `PUBLIC_REGISTRATION`: Allow public user registration (default: true)

See `.env.example` for complete configuration options.

## Setting up MinIO (Local Development)

For local development, you can use MinIO as S3-compatible storage:

```bash
# Using Docker
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  quay.io/minio/minio server /data --console-address ":9001"
```

Then create a bucket named `filesonthego` via the MinIO console at `http://localhost:9001`

## API Endpoints

### Health Check
- `GET /api/health` - Check application status

### Files (coming soon)
- `POST /api/files/upload` - Upload a file
- `GET /api/files/:id/download` - Download a file
- `GET /api/files` - List files in a directory
- `DELETE /api/files/:id` - Delete a file

### Shares (coming soon)
- `POST /api/shares` - Create a share link
- `GET /api/shares/:token` - Access shared content

See [DESIGN.md](DESIGN.md) for complete API documentation.

## Testing

This project follows Test-Driven Development (TDD) principles. All features must include tests.

### Test Coverage Requirements
- Minimum 80% coverage for all packages
- 100% coverage for security-critical code

### Running Specific Tests

```bash
# Run tests for a specific package
go test ./config/...

# Run a specific test
go test -run TestLoad_WithValidConfig ./config/

# Run with race detection
go test -race ./...
```

See [CLAUDE.md](CLAUDE.md) for detailed testing guidelines.

## Contributing

1. Read [CLAUDE.md](CLAUDE.md) for development guidelines
2. Read [DESIGN.md](DESIGN.md) for architecture details
3. Write tests for all new features
4. Follow Go best practices and project conventions
5. Ensure all tests pass before submitting

## Architecture

FilesOnTheGo uses a layered architecture:

- **Handlers**: HTTP request/response handling
- **Services**: Business logic and external integrations
- **Models**: Data structures and validation
- **Middleware**: Request processing pipeline

See [DESIGN.md](DESIGN.md) for detailed architecture documentation.

## Security

- All file operations require authentication
- Permission validation on every access
- Input sanitization to prevent path traversal
- Rate limiting on sensitive endpoints
- Secure share token generation
- Support for password-protected shares

See [CLAUDE.md](CLAUDE.md) for security guidelines.

## License

See [LICENSE](LICENSE) file for details.

## Documentation

- [DESIGN.md](DESIGN.md) - Architecture and design decisions
- [CLAUDE.md](CLAUDE.md) - AI-assisted development guidelines
- [plan/](plan/) - Implementation plan and roadmap

## Support

For issues and questions, please create an issue on GitHub.
