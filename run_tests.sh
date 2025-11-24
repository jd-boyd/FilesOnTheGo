#!/bin/bash
# FilesOnTheGo Test Runner
# This script runs integration tests in a containerized environment with MinIO and the service

set -e

verbose='false'
skip='false'
test_pattern=''

print_usage() {
  echo "Usage: $0 [flags] [test_pattern]"
  echo "Flags:"
  echo " -s   Skip building container via Podman (after you've done it once, makes tests run faster)"
  echo " -v   Be more verbose"
  echo " -h   Show this help message"
  echo ""
  echo "Test patterns (optional):"
  echo " No pattern                        - Run all integration tests"
  echo " TestContainer_LoginFlow           - Run login flow tests"
  echo " TestContainer_UserRegistration    - Run user registration tests"
  echo " container                         - Run all container tests"
  echo " unit                              - Run unit tests only"
  echo ""
  echo "Examples:"
  echo " $0                                         # Run all integration tests"
  echo " $0 -s                                      # Run all tests, skip container build"
  echo " $0 TestContainer_LoginFlow                 # Run login flow tests only"
  echo " $0 container                               # Run all container tests"
  echo " $0 unit                                    # Run unit tests only (no container)"
}

while getopts 'vsh' flag; do
  case "${flag}" in
    v) verbose='true' ;;
    s) skip='true' ;;
    h) print_usage
       exit 0 ;;
    *) print_usage
       exit 1 ;;
  esac
done

# Get the test pattern from remaining arguments
shift $((OPTIND-1))
test_pattern="$1"

# Configuration
export POD_NAME=filesonthego_test_pod
export DATA_DIR=../${POD_NAME}_data
export MINIO_DATA=${DATA_DIR}/minio_data
export APP_DATA=${DATA_DIR}/app_data

# MinIO Configuration
export MINIO_ROOT_USER=filesonthego_test_admin
export MINIO_ROOT_PASSWORD=test_password_123
export MINIO_BUCKET=filesonthego-test

# FilesOnTheGo Application Configuration
export APP_PORT=8090
export APP_ENVIRONMENT=test
export APP_URL=http://localhost:8090

# S3 Configuration (connecting to MinIO)
export S3_ENDPOINT=http://localhost:9000
export S3_REGION=us-east-1
export S3_ACCESS_KEY=${MINIO_ROOT_USER}
export S3_SECRET_KEY=${MINIO_ROOT_PASSWORD}
export S3_USE_SSL=false

# Database Configuration (PocketBase)
export DB_PATH=/app/data/pb_data

# Upload and Security Configuration
export MAX_UPLOAD_SIZE=104857600  # 100MB in bytes
export JWT_SECRET=test_jwt_secret_change_in_production_12345
export PUBLIC_REGISTRATION=true
export EMAIL_VERIFICATION=false
export REQUIRE_EMAIL_AUTH=false

# Admin Account for Testing
export ADMIN_EMAIL=admin@filesonthego.test
export ADMIN_PASSWORD=admin123

echo "=== FilesOnTheGo Test Environment ==="
echo "Using DATA_DIR: ${DATA_DIR}"
echo "Using MINIO_DATA: ${MINIO_DATA}"
echo "Using APP_DATA: ${APP_DATA}"
echo "S3 Endpoint: ${S3_ENDPOINT}"
echo "App URL: ${APP_URL}"
echo "Admin Account: ${ADMIN_EMAIL}"

# Check if we should run unit tests only
if [ "$test_pattern" = "unit" ]; then
    echo "Running unit tests only (no container setup)..."
    go test ./tests/unit/... -v
    exit $?
fi

# Create data directories
mkdir -p $MINIO_DATA
mkdir -p $APP_DATA

# Clean up existing pod and create new one
echo "Setting up test Podman pod..."
podman pod rm -f $POD_NAME && /bin/true

podman pod create -p ${APP_PORT}:8090 -p 9000:9000 -p 9001:9001 -n $POD_NAME

# Start MinIO service
echo "Starting MinIO service..."
podman run -d \
       --pod $POD_NAME \
       -v $MINIO_DATA:/data \
       -e "MINIO_ROOT_USER=${MINIO_ROOT_USER}" \
       -e "MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}" \
       --name filesonthego-minio-test \
       quay.io/minio/minio server /data --console-address ":9001"

# Wait for MinIO to start
echo "Waiting for MinIO to start..."
max_attempts=30
attempt=1

while [ $attempt -le $max_attempts ]; do
    echo "Checking MinIO health (attempt $attempt/$max_attempts)..."

    # Check if MinIO is responding to HTTP requests
    if podman run --rm --pod $POD_NAME \
       --entrypoint=/bin/sh \
       quay.io/minio/mc -c "mc config host add myminio http://localhost:9000 ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD}" >/dev/null 2>&1; then
        echo "MinIO is ready!"
        break
    fi

    if [ $attempt -eq $max_attempts ]; then
        echo "Error: MinIO failed to start after $max_attempts attempts"
        echo "Cleaning up..."
        podman pod rm -f $POD_NAME
        exit 1
    fi

    echo "MinIO not ready yet, waiting 2 seconds..."
    sleep 2
    attempt=$((attempt + 1))
done

# Create MinIO bucket and set access policy
echo "Setting up MinIO bucket..."
if podman run \
       --pod $POD_NAME \
       --entrypoint=/bin/sh \
       quay.io/minio/mc -c "\
      /usr/bin/mc alias set myminio http://localhost:9000 ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} && \
      /usr/bin/mc mb --ignore-existing myminio/${MINIO_BUCKET} && \
      /usr/bin/mc anonymous set download myminio/${MINIO_BUCKET} && \
      echo 'MinIO bucket setup complete'"; then
    echo "✅ MinIO bucket setup completed successfully"
else
    echo "❌ Failed to set up MinIO bucket"
    echo "Cleaning up..."
    podman pod rm -f $POD_NAME
    exit 1
fi

if [ "$skip" == 'true' ] ; then
    echo "Skipping container build..."
else
    echo "Building FilesOnTheGo container image..."
    podman build -t filesonthego:test ./
fi

# Start FilesOnTheGo application
echo "Starting FilesOnTheGo application..."
podman run -d \
       --pod $POD_NAME \
       -v $APP_DATA:/app/data \
       -e APP_PORT=${APP_PORT} \
       -e APP_ENVIRONMENT=${APP_ENVIRONMENT} \
       -e APP_URL=${APP_URL} \
       -e S3_ENDPOINT=${S3_ENDPOINT} \
       -e S3_REGION=${S3_REGION} \
       -e S3_BUCKET=${MINIO_BUCKET} \
       -e S3_ACCESS_KEY=${S3_ACCESS_KEY} \
       -e S3_SECRET_KEY=${S3_SECRET_KEY} \
       -e S3_USE_SSL=${S3_USE_SSL} \
       -e DB_PATH=${DB_PATH} \
       -e MAX_UPLOAD_SIZE=${MAX_UPLOAD_SIZE} \
       -e JWT_SECRET=${JWT_SECRET} \
       -e PUBLIC_REGISTRATION=${PUBLIC_REGISTRATION} \
       -e EMAIL_VERIFICATION=${EMAIL_VERIFICATION} \
       -e REQUIRE_EMAIL_AUTH=${REQUIRE_EMAIL_AUTH} \
       --name filesonthego-app-test \
       filesonthego:test

# Wait for application to start
echo "Waiting for FilesOnTheGo application to start..."
sleep 10

# Create admin account using PocketBase CLI if it exists
echo "Attempting to create admin account..."
podman run \
       --pod $POD_NAME \
       -v $APP_DATA:/app/data \
       --entrypoint=/bin/sh \
       filesonthego:test -c "\
      /app/filesonthego admin create ${ADMIN_EMAIL} ${ADMIN_PASSWORD} 2>/dev/null && \
      echo 'Admin account created successfully' || \
      echo 'Note: Admin account creation failed - may already exist'"

echo "Test environment is ready!"

# Run the integration tests
echo "Starting integration tests..."

# Set test environment variables for Go tests
export CGO_ENABLED=0
export GOOS=linux

if [ -n "$test_pattern" ] && [ "$test_pattern" != "container" ]; then
    echo "Running tests matching pattern: $test_pattern"
    go test -tags=container ./tests/integration/container_test.go -v -run "$test_pattern" -timeout=5m
else
    echo "Running all container integration tests"
    go test -tags=container ./tests/integration/... -v -timeout=5m
fi

# Capture the exit code from the test run
TEST_EXIT_CODE=$?

echo "Cleaning up test environment..."
podman pod rm -f $POD_NAME

# Exit with the test exit code
exit $TEST_EXIT_CODE