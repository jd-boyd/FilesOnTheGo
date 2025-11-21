# Step 01: Project Scaffolding and PocketBase Setup

## Overview
Set up the foundational project structure for FilesOnTheGo, initialize Go modules, configure PocketBase, and create the basic directory structure.

## Dependencies
- None (this is the first step)

## Duration Estimate
30 minutes

## Agent Prompt

You are implementing Step 01 of the FilesOnTheGo project. Your task is to create the complete project scaffolding and PocketBase setup.

### Tasks

1. **Initialize Go Module**
   - Create `go.mod` with module name `github.com/jd-boyd/filesonthego`
   - Set Go version to 1.21 or higher

2. **Add Initial Dependencies**
   - PocketBase: `github.com/pocketbase/pocketbase`
   - AWS SDK for Go (S3): `github.com/aws/aws-sdk-go-v2/service/s3`
   - AWS SDK config: `github.com/aws/aws-sdk-go-v2/config`
   - AWS SDK credentials: `github.com/aws/aws-sdk-go-v2/credentials`
   - Testify for testing: `github.com/stretchr/testify`
   - UUID generation: `github.com/google/uuid`
   - Zerolog for logging: `github.com/rs/zerolog`

3. **Create Directory Structure**
   ```
   FilesOnTheGo/
   ├── main.go
   ├── handlers/
   ├── services/
   ├── models/
   ├── middleware/
   ├── templates/
   │   ├── layouts/
   │   ├── components/
   │   └── pages/
   ├── static/
   │   ├── css/
   │   └── js/
   ├── tests/
   │   ├── integration/
   │   ├── unit/
   │   └── fixtures/
   ├── migrations/
   └── config/
   ```

4. **Create main.go**
   - Initialize PocketBase app
   - Set up basic configuration loading from environment variables
   - Configure logging with zerolog
   - Add placeholder for S3 configuration
   - Set up graceful shutdown
   - Include comprehensive comments

5. **Create config/config.go**
   - Define configuration struct with fields for:
     - S3 endpoint, region, bucket, access key, secret key
     - Application port, environment, URL
     - Database path
     - Max upload size
     - JWT secret
     - Feature flags (public registration, email verification)
     - Default user quota
   - Implement loading from environment variables
   - Include validation logic

6. **Create .env.example**
   - Include all configuration variables from DESIGN.md
   - Add helpful comments for each variable
   - Set reasonable development defaults

7. **Update .gitignore**
   - Add pb_data/
   - Add .env
   - Add coverage files
   - Add build artifacts

8. **Create README.md**
   - Project description
   - Quick start instructions
   - Development setup
   - Running tests
   - Link to DESIGN.md and CLAUDE.md

9. **Write Tests**
   - Create `config/config_test.go` with tests for configuration loading
   - Test environment variable parsing
   - Test validation logic
   - Achieve 80%+ coverage

### Success Criteria

- [ ] Go module initialized with all dependencies
- [ ] Complete directory structure created
- [ ] main.go successfully compiles and runs
- [ ] Configuration system works and is tested
- [ ] Environment variables properly loaded
- [ ] Tests pass with `go test ./...`
- [ ] Code follows CLAUDE.md guidelines
- [ ] Documentation is clear and complete

### Testing Commands

```bash
# Initialize and download dependencies
go mod tidy

# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Build the application
go build -o filesonthego main.go

# Run the application (should start without errors)
./filesonthego serve
```

### Expected Output

After completion:
1. Project compiles without errors
2. PocketBase starts successfully on port 8090
3. All tests pass
4. Configuration loads from environment variables
5. Logging is properly configured

### References

- DESIGN.md: Configuration Management section
- CLAUDE.md: Project Structure and Development Workflow sections
- PocketBase docs: https://pocketbase.io/docs/

### Notes

- Use structured logging (zerolog) from the start
- Follow Go best practices for project layout
- Ensure all paths are configurable via environment variables
- Add health check endpoint (`/api/health`)
- This step is critical - all other steps depend on this foundation
