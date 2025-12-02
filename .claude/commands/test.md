# test - Run Tests

Run test suite for the FilesOnTheGo project. By default, this command runs unit tests which are the core test suite for the project.

## What it does

- Runs the unit test suite using `make test-unit`
- Provides comprehensive test output and statistics
- Tests all core components: models, services, handlers, and templates
- Excludes integration tests that require database/external services

## Usage

```
/test
```

This will execute the unit test suite and show you:
- ✅ Overall test results (pass/fail counts)
- 📊 Execution time and coverage statistics
- 📦 Detailed breakdown by test package
- 🚨 Any test failures with error details

## Expected Output

```
🧪 Test Summary
============================================================
📊 Overall Results: 421 tests, 420 passed, 1 skipped
⏱️  Total Time: 1.07s
✅ Unit tests completed successfully!
```

## Additional Test Commands

For more specific testing, use:
- `/test:unit` - Run unit tests only (same as /test)
- `/test:integration` - Run integration tests
- `/test:security` - Run security tests
- `/test:coverage` - Generate coverage report

## What Gets Tested

The test suite covers:
- **Models**: Data structures, validation, and database models (201 tests)
- **Services**: Business logic, S3 operations, permissions (139 tests)
- **Configuration**: Settings and environment handling (18 tests)
- **Templates**: HTML template rendering and assets (13 tests)
- **UI Components**: Frontend interactions (44 tests)

Total: **415+ tests** covering all core functionality

## Troubleshooting

If tests fail:
1. Check dependencies: `go mod tidy`
2. Fix formatting: `make fmt`
3. Verify imports: handlers → handlers_gin migration
4. Check template files exist in assets/

## Related Files

- `Makefile` - Contains test commands and targets
- `tests/unit/` - Unit test implementation files
- `CLAUDE.md` - Testing guidelines and requirements