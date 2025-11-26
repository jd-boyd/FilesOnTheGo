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

## Development Environment

### Quick Development Setup

For the easiest development experience, use the provided development script:

```bash
# This script sets up MinIO, builds the app, and creates test accounts
./run_dev.sh
```

The script will:
- Start MinIO and create the required bucket
- Build and start the FilesOnTheGo application
- Create test accounts:
  - **Admin**: `admin@filesonthego.local` / `admin123`
  - **Regular User**: `user@filesonthego.local` / `user123`

Access the application at http://localhost:8090

## Running Tests

### Go Unit & Integration Tests

```bash
# All tests with detailed summary
make test

# Verbose output with full test details
make test-verbose

# Run with coverage report
make test-coverage

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run security tests only
go test -v ./tests/security/...

# Run tests for specific package
go test -v ./services/...
go test -v ./handlers/...

# Run with race detection
make race

# Run benchmarks
make benchmark
```

### Containerized Integration Tests

For comprehensive integration testing, use the `run_tests.sh` script which sets up a complete test environment with MinIO and runs tests in containers:

```bash
# Run all integration tests with full environment setup
./run_tests.sh

# Skip container rebuild for faster test runs
./run_tests.sh -s

# Run unit tests only (no containers)
./run_tests.sh unit

# Run specific integration tests
./run_tests.sh TestContainer_LoginFlow
./run_tests.sh TestContainer_UserRegistration

# Run all container-based integration tests
./run_tests.sh container

# Verbose integration testing
./run_tests.sh -v

# Show help and usage options
./run_tests.sh -h
```

**What `run_tests.sh` does:**
- Sets up isolated test environment with MinIO in containers
- Builds and deploys the FilesOnTheGo application
- Creates test database and admin account
- Runs integration tests against the live application
- Automatically cleans up test environment
- Supports test pattern matching for specific test scenarios

**Test Environment Details:**
- **Admin Account**: `admin@filesonthego.test` / `admin123`
- **S3 Endpoint**: `http://localhost:9000` (MinIO)
- **App URL**: `http://localhost:8090`
- **Test Data**: Stored in `../filesonthego_test_pod_data/`

### Playwright End-to-End Tests

The project includes comprehensive Playwright tests for user interface testing and development environment validation.

#### Prerequisites
```bash
# Install Playwright (one-time setup)
npm install
npx playwright install chromium
```

#### Running Playwright Tests

```bash
# Quick health check - verifies application is running
npx playwright test --project=chromium --grep="Application health endpoint is accessible"

# Test the core user creation functionality (main regression test)
npx playwright test --project=chromium --grep="REGRESSION: Regular user should exist after run_dev.sh fix"

# Test admin panel visibility for users
npx playwright test --project=chromium --grep="REGRESSION: Both users should be visible in admin panel"

# Run comprehensive regression verification
npx playwright test --project=chromium --grep="POST-REGRESSION: Verify fix completeness"

# Run all user creation related tests
npx playwright test user-creation.spec.ts

# Run development environment validation tests
npx playwright test dev-environment.spec.ts

# Run all regression tests
npx playwright test user-creation-regression.spec.ts

# Run all Playwright tests
npx playwright test --project=chromium

# Run with visual debugging
npx playwright test --project=chromium --headed

# Generate HTML report
npx playwright test --project=chromium --reporter=html
```

#### Test Categories

**1. User Creation Tests** (`user-creation.spec.ts`)
- Admin user access and navigation
- Regular user login validation
- User permissions and access control
- Session management functionality

**2. Development Environment Tests** (`dev-environment.spec.ts`)
- Application health endpoint validation
- Test account functionality from `run_dev.sh`
- MinIO/S3 integration checks
- Error handling and edge cases

**3. Regression Tests** (`user-creation-regression.spec.ts`)
- Tests for the specific issue: "run_dev.sh says normal user exists, but admin users page only shows admin account"
- Comprehensive verification that fixes are working
- Detailed logging and debugging information

#### Test Results Interpretation

**✅ Success Indicators:**
- All tests pass: `Passed: 5/5` - Issue is resolved
- Regular user can log in
- Both users visible in admin panel

**❌ Failure Indicators:**
- `regularUserCanLogIn: ❌` - User creation issue persists
- `regularUserInAdminPanel: ❌` - Database/user storage issue

#### Debugging Features
- Automatic screenshots on failures (saved in `test-results/`)
- Video recordings of test runs
- Detailed console logging with user visibility status
- Error context and email detection in admin panel

See [playwright/tests/README.md](playwright/tests/README.md) for detailed Playwright test documentation.

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
├── tests/            # Go test files (unit, integration, security)
├── playwright/       # Playwright E2E tests
│   ├── tests/        # Playwright test specifications
│   ├── config.ts     # Playwright configuration
│   └── README.md     # Playwright test documentation
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
