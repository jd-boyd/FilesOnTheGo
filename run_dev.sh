#!/bin/bash
# FilesOnTheGo Development Environment Setup
# This script sets up MinIO and FilesOnTheGo service in a podman pod for local development

set -e

# MinIO Configuration
export MINIO_ROOT_USER=filesonthego_admin
export MINIO_ROOT_PASSWORD=dev_password_123
export MINIO_BUCKET=filesonthego-dev

# Pod and Data Configuration
export POD_NAME=filesonthego_dev_pod
export DATA_DIR=../${POD_NAME}_data
export MINIO_DATA=${DATA_DIR}/minio_data
export APP_DATA=${DATA_DIR}/app_data

# FilesOnTheGo Application Configuration
export APP_PORT=8090
export APP_ENVIRONMENT=development

# Add --skip-build flag support
SKIP_BUILD=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        *)
            ;;
    esac
done


# Parse command line arguments
SKIP_BUILD=false
USE_EXTERNAL_ASSETS=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --external-assets)
            USE_EXTERNAL_ASSETS=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Detect hostname for external access (can be overridden with HOST_IP environment variable)
if [ -z "$HOST_IP" ]; then
    # Try to get the primary IP address
    HOST_IP=$(hostname -I | awk '{print $1}')
    # Fallback to hostname if IP detection fails
    if [ -z "$HOST_IP" ]; then
        HOST_IP=$(hostname)
    fi
fi

export APP_URL=http://${HOST_IP}:${APP_PORT}

# S3 Configuration (connecting to MinIO)
export S3_ENDPOINT=http://${HOST_IP}:9000
export S3_REGION=us-east-1
export S3_ACCESS_KEY=${MINIO_ROOT_USER}
export S3_SECRET_KEY=${MINIO_ROOT_PASSWORD}
export S3_USE_SSL=false

# Database Configuration (SQLite via GORM)
export DB_PATH=/app/data/database.db

# Upload and Security Configuration
export MAX_UPLOAD_SIZE=104857600  # 100MB in bytes
export JWT_SECRET=dev_jwt_secret_change_in_production_12345
export PUBLIC_REGISTRATION=true
export EMAIL_VERIFICATION=false
export REQUIRE_EMAIL_AUTH=false

# Test Accounts
export ADMIN_EMAIL=admin@filesonthego.local
export ADMIN_PASSWORD=admin123
export USER_EMAIL=user@filesonthego.local
export USER_PASSWORD=user1234

echo "=== FilesOnTheGo Development Environment ==="
echo "Using DATA_DIR: ${DATA_DIR}"
echo "Using MINIO_DATA: ${MINIO_DATA}"
echo "Using APP_DATA: ${APP_DATA}"
echo "Host IP: ${HOST_IP}"
echo "S3 Endpoint: ${S3_ENDPOINT}"
echo "App URL: ${APP_URL}"
echo ""
echo "Note: Service will be accessible from other machines on the network"
echo "To use localhost only, set HOST_IP=localhost before running this script"
echo ""

if [ "$SKIP_BUILD" = "true" ]; then
    echo "Skiping container build."
else
    # Build FilesOnTheGo application container (before starting the app)
    echo "Building FilesOnTheGo container image..."
    # Use --no-cache if NOCACHE environment variable is set
    if [ "$NOCACHE" = "true" ]; then
        podman build --no-cache -t filesonthego:latest .
    else
        podman build -t filesonthego:latest .
    fi
fi

# Build the binary locally since we mount the source directory into the container
echo "Building binary locally for development mode..."
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o filesonthego main.go


# Check for and stop any existing pod
echo "Checking for existing pod..."
if podman pod exists $POD_NAME 2>/dev/null; then
    echo "Found existing pod '$POD_NAME', stopping and removing..."
    podman pod stop $POD_NAME 2>/dev/null
    podman pod rm -f $POD_NAME 2>/dev/null
    echo "✅ Cleaned up existing pod"
fi


# Create data directories
mkdir -p $MINIO_DATA
mkdir -p $APP_DATA

# Set permissions for container user (uid 1001)
# The container runs as appuser (uid 1001), so the data directory needs to be writable
podman unshare chown -R 1001:1001 $APP_DATA 2>/dev/null || \
    chmod -R 777 $APP_DATA

# Create new pod
echo "Setting up Podman pod..."
podman pod create -p ${APP_PORT}:8090 -p 9000:9000 -p 9001:9001 -n $POD_NAME

# Start MinIO service
echo "Starting MinIO service..."
podman run -d \
       --pod $POD_NAME \
       -v $MINIO_DATA:/data \
       -e "MINIO_ROOT_USER=${MINIO_ROOT_USER}" \
       -e "MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}" \
       --name filesonthego-minio-dev \
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

# Ensure MinIO is fully ready before starting FilesOnTheGo
echo "Verifying MinIO is ready before starting application..."
max_check_attempts=10
check_attempt=1
while [ $check_attempt -le $max_check_attempts ]; do
    echo "Checking MinIO bucket readiness (attempt $check_attempt/$max_check_attempts)..."

    if podman run \
           --pod $POD_NAME \
           --entrypoint=/bin/sh \
           quay.io/minio/mc -c "\
           /usr/bin/mc alias set myminio http://localhost:9000 ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} && \
           /usr/bin/mc ls myminio/${MINIO_BUCKET}" >/dev/null 2>&1; then
        echo "✅ MinIO and bucket are ready!"
        break
    fi

    if [ $check_attempt -eq $max_check_attempts ]; then
        echo "❌ MinIO bucket not ready after $max_check_attempts attempts"
        echo "Cleaning up..."
        podman pod rm -f $POD_NAME
        exit 1
    fi

    sleep 2
    check_attempt=$((check_attempt + 1))
done

# Start FilesOnTheGo application
echo "Starting FilesOnTheGo application..."
podman run -d \
       --pod $POD_NAME \
       -v $APP_DATA:/app/data \
       -v "$(pwd)":/app:ro \
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
       -e ADMIN_EMAIL=${ADMIN_EMAIL} \
       --name filesonthego-app-dev \
       filesonthego:latest

# Wait for application to start
echo "Waiting for FilesOnTheGo application to start..."
max_wait=30
for i in $(seq 1 $max_wait); do
    if curl -s http://${HOST_IP}:${APP_PORT}/api/status > /dev/null 2>&1; then
        echo "Application is ready!"
        break
    fi
    if [ $i -eq $max_wait ]; then
        echo "Warning: Application did not become ready within ${max_wait} seconds"
    fi
    sleep 1
done

# Create test users via API
echo "Creating test user accounts..."

# Create admin user
ADMIN_RESPONSE=$(curl -s -w "%{http_code}" -X POST http://${HOST_IP}:${APP_PORT}/api/auth/register \
     -H "Content-Type: application/json" \
     -d "{\"email\":\"${ADMIN_EMAIL}\",\"username\":\"admin\",\"password\":\"${ADMIN_PASSWORD}\",\"passwordConfirm\":\"${ADMIN_PASSWORD}\"}")

ADMIN_HTTP_CODE="${ADMIN_RESPONSE: -3}"

if [ "$ADMIN_HTTP_CODE" = "200" ] || [ "$ADMIN_HTTP_CODE" = "201" ]; then
    echo "  ✅ Admin user created successfully"
elif [ "$ADMIN_HTTP_CODE" = "400" ]; then
    echo "  ℹ️  Admin user may already exist (HTTP 400)"
else
    echo "  ⚠️  Admin user creation issue (HTTP $ADMIN_HTTP_CODE)"
    echo "  Response: ${ADMIN_RESPONSE%???}"
fi

# Create regular user
USER_RESPONSE=$(curl -s -w "%{http_code}" -X POST http://${HOST_IP}:${APP_PORT}/api/auth/register \
     -H "Content-Type: application/json" \
     -d "{\"email\":\"${USER_EMAIL}\",\"username\":\"user\",\"password\":\"${USER_PASSWORD}\",\"passwordConfirm\":\"${USER_PASSWORD}\"}")

USER_HTTP_CODE="${USER_RESPONSE: -3}"

if [ "$USER_HTTP_CODE" = "200" ] || [ "$USER_HTTP_CODE" = "201" ]; then
    echo "  ✅ Regular user created successfully"
elif [ "$USER_HTTP_CODE" = "400" ]; then
    echo "  ℹ️  Regular user may already exist (HTTP 400)"
else
    echo "  ⚠️  Regular user creation issue (HTTP $USER_HTTP_CODE)"
    echo "  Response: ${USER_RESPONSE%???}"
fi

echo ""
echo "=========================================="
echo "=== Development Environment Ready ==="
echo "=========================================="
echo ""
echo "📍 Access URLs:"
echo "  Application (network): http://${HOST_IP}:${APP_PORT}"
echo "  Application (local):   http://localhost:${APP_PORT}"
echo "  MinIO Console:         http://localhost:9001"
echo ""
echo "👤 Test User Accounts:"
echo ""
echo "  Admin User:"
echo "    Email:    ${ADMIN_EMAIL}"
echo "    Password: ${ADMIN_PASSWORD}"
echo "    Access:   Full admin privileges, can manage users & settings"
echo ""
echo "  Regular User:"
echo "    Email:    ${USER_EMAIL}"
echo "    Password: ${USER_PASSWORD}"
echo "    Access:   Standard user, can upload/download files"
echo ""
echo "🗄️  MinIO Console:"
echo "    Username: ${MINIO_ROOT_USER}"
echo "    Password: ${MINIO_ROOT_PASSWORD}"
echo ""
echo "============================================"
echo ""
echo "📋 Tailing application logs..."
echo "   Press Ctrl-C to stop and cleanup the environment"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo ""
    echo "🛑 Shutting down development environment..."
    echo "   Stopping pod..."
    podman pod stop ${POD_NAME} 2>/dev/null
    echo "   Removing pod..."
    podman pod rm -f ${POD_NAME} 2>/dev/null
    echo "✅ Development environment cleaned up"
    exit 0
}

# Trap Ctrl-C and call cleanup
trap cleanup SIGINT SIGTERM

# Tail the application logs (this will block until Ctrl-C)
podman logs -f filesonthego-app-dev
