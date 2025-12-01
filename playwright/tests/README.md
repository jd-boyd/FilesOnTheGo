# Playwright Test Suite for FilesOnTheGo

This directory contains comprehensive Playwright test cases for the FilesOnTheGo application, specifically focused on testing user creation and management functionality.

## 📁 Test Files

### 1. `user-creation.spec.ts`
**Purpose**: Tests user creation, login functionality, and admin panel access
- Admin user access and navigation
- Regular user login validation
- User permissions and access control
- Session management and logout functionality

### 2. `dev-environment.spec.ts`
**Purpose**: Tests the development environment setup created by `run_dev.sh`
- Application health endpoint validation
- Test account functionality from run_dev.sh
- MinIO/S3 integration checks
- Edge cases and error handling

### 3. `user-creation-regression.spec.ts`
**Purpose**: Regression tests specifically for the user creation issue
- Tests for the original issue: "run_dev.sh says normal user exists, but admin users page only shows admin account"
- Comprehensive verification that fixes are working
- Detailed logging and debugging information
- Post-regression verification

## 🎯 Focus Area: User Creation Issue

### Original Problem
The `run_dev.sh` script was reporting successful user creation:
```
✅ Admin account created
✅ Regular user account created
```

But the admin users page only showed the admin account, not the regular user.

### Root Cause Identified
1. **Database not initialized**: Application was waiting for admin user setup
2. **Wrong collection name**: Script was creating users in `users` collection but app looks in `superusers` collection
3. **API appeared to succeed**: User creation API calls returned success but users weren't actually stored

### Fix Applied
Updated `run_dev.sh` to:
- Use application CLI for admin user creation: `./filesonthego create-admin`
- Proper sequencing: Initialize database first, then create regular users
- Wait for initialization before creating regular users

## 🚀 Running the Tests

### Quick Health Check
```bash
npx playwright test --project=chromium --grep="Application health endpoint is accessible"
```

### User Creation Regression Tests
```bash
# Test specific user creation issue
npx playwright test --project=chromium --grep="REGRESSION: Regular user should exist after run_dev.sh fix"

# Test admin panel visibility
npx playwright test --project=chromium --grep="REGRESSION: Both users should be visible in admin panel"

# Run comprehensive regression verification
npx playwright test --project=chromium --grep="POST-REGRESSION: Verify fix completeness"
```

### Full Test Suite
```bash
npx playwright test --project=chromium
```

### Development Environment Tests
```bash
npx playwright test dev-environment.spec.ts
```

## 📊 Test Results Interpretation

### Success Indicators
- ✅ **Admin login works**: Basic functionality is intact
- ✅ **Regular user login works**: User creation fix is successful
- ✅ **Both users visible in admin panel**: Complete fix verified

### Failure Indicators
- ❌ **Regular user login fails**: User creation issue persists
- ❌ **Regular user missing from admin panel**: Database/user storage issue

### Test Output Examples

#### ✅ Successful Fix
```
📊 REGRESSION TEST SUMMARY:
   Passed: 5/5
   Status: ✅ ALL TESTS PASSED - ISSUE RESOLVED

Detailed Results:
  adminCanLogIn: ✅
  regularUserCanLogIn: ✅
  adminUserInAdminPanel: ✅
  regularUserInAdminPanel: ✅
  regularUserNoAdminAccess: ✅

🎉 REGRESSION FIX VERIFICATION: SUCCESSFUL!
```

#### ❌ Issue Persists
```
📊 REGRESSION TEST SUMMARY:
   Passed: 2/5
   Status: ❌ SOME TESTS FAILED - ISSUE PERSISTS

Detailed Results:
  adminCanLogIn: ✅
  regularUserCanLogIn: ❌
  adminUserInAdminPanel: ✅
  regularUserInAdminPanel: ❌
  regularUserNoAdminAccess: ❌

🔧 REGRESSION FIX VERIFICATION: INCOMPLETE
```

## 🛠️ Debugging Features

### Screenshots
Tests automatically capture screenshots on failures:
- `admin-panel-users-{timestamp}.png` - Shows what users are visible
- `regular-user-login-failed.png` - Shows login failure state
- `debug-*.png` - Various debugging screenshots

### Console Logging
Tests provide detailed console output:
- User visibility status
- Login success/failure reasons
- Error messages and debugging information
- Email addresses found on admin page

### Error Context
- Video recordings of test runs
- Detailed error messages
- Page content analysis for debugging

## 🔧 Configuration

Tests use the configuration from `playwright.config.ts`:
- **Base URL**: `http://192.168.1.62:8090` (development environment)
- **Browser**: Chromium
- **Timeouts**: Standard Playwright defaults
- **Screenshots**: Captured on failure
- **Videos**: Recorded for failed tests

## 📝 Test Data

**Default Test Users** (from `run_dev.sh`):
- **Admin**: `admin@filesonthego.local` / `admin123`
- **Regular User**: `user@filesonthego.local` / `user123`

These credentials match what the `run_dev.sh` script should be creating.

## 🎯 Best Practices

### Before Running Tests
1. Ensure the development environment is running: `./run_dev.sh`
2. Verify application is accessible at the configured base URL
3. Check that MinIO/S3 services are properly initialized

### After Code Changes
1. Run the regression tests to verify fixes
2. Use specific test groups for targeted validation
3. Check screenshots and logs for debugging information

### Continuous Integration
These tests are designed to catch regressions in the user creation functionality and can be integrated into CI/CD pipelines to ensure the `run_dev.sh` script continues to work correctly.