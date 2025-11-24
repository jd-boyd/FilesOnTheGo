# FilesOnTheGo

Self-hosted file storage with S3 backend, built on PocketBase.

## What is this?

FilesOnTheGo is a file sharing service you run yourself. Think Nextcloud or Google Drive, but simpler. Files get stored in S3-compatible storage (MinIO, AWS, Backblaze, etc.) while PocketBase handles users, auth, and metadata.

The frontend uses HTMX and Tailwind—minimal JavaScript, server-rendered pages.

## Getting Started

You'll need:
- Go 1.21+
- S3-compatible storage (MinIO works great for local dev)

```bash
git clone https://github.com/jd-boyd/filesonthego.git
cd FilesOnTheGo

cp .env.example .env
# Edit .env with your S3 credentials

go mod download
go build -o filesonthego main.go
./filesonthego serve
```

Open http://localhost:8090

## Local Development with MinIO

For local dev, spin up MinIO:

```bash
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  quay.io/minio/minio server /data --console-address ":9001"
```

Create a bucket called `filesonthego` at http://localhost:9001

## Configuration

Set these in your `.env`:

```bash
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=filesonthego
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
APP_URL=http://localhost:8090
```

Optional settings (with defaults):
- `APP_PORT` - HTTP port (8090)
- `MAX_UPLOAD_SIZE` - Max file size in bytes (100MB)
- `DEFAULT_USER_QUOTA` - Storage per user (10GB)
- `PUBLIC_REGISTRATION` - Allow signups (true)

See `.env.example` for everything.

## Running Tests

```bash
# All tests with summary
make test

# Verbose output
make test-verbose

# With coverage
make test-coverage

# Specific package
go test -v ./services/...
```

## Project Layout

```
FilesOnTheGo/
├── main.go           # Entry point
├── config/           # Config loading
├── handlers/         # HTTP handlers
├── services/         # Business logic
├── models/           # Data types
├── middleware/       # Request middleware
├── templates/        # HTMX templates
├── static/           # CSS, JS, icons
├── tests/            # Test files
└── migrations/       # DB migrations
```

## API

Current endpoints:
- `GET /api/health` - Health check

Coming soon:
- `POST /api/files/upload` - Upload
- `GET /api/files/:id/download` - Download
- `GET /api/files` - List files
- `DELETE /api/files/:id` - Delete
- `POST /api/shares` - Create share link
- `GET /api/shares/:token` - Access share

See [DESIGN.md](DESIGN.md) for the full API spec.

## Docs

- [DESIGN.md](DESIGN.md) - Architecture and API design
- [CLAUDE.md](CLAUDE.md) - Development guidelines
- [plan/](plan/) - Implementation roadmap

## License

See [LICENSE](LICENSE).
