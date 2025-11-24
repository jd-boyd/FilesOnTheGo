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

# Database Configuration (PocketBase)
export DB_PATH=/app/data/pb_data

# Upload and Security Configuration
export MAX_UPLOAD_SIZE=104857600  # 100MB in bytes
export JWT_SECRET=dev_jwt_secret_change_in_production_12345
export PUBLIC_REGISTRATION=true
export EMAIL_VERIFICATION=false
export REQUIRE_EMAIL_AUTH=false

# Admin Account for Testing
export ADMIN_EMAIL=admin@filesonthego.local
export ADMIN_PASSWORD=admin123

echo "=== FilesOnTheGo Development Environment ==="
echo "Using DATA_DIR: ${DATA_DIR}"
echo "Using MINIO_DATA: ${MINIO_DATA}"
echo "Using APP_DATA: ${APP_DATA}"
echo "Host IP: ${HOST_IP}"
echo "S3 Endpoint: ${S3_ENDPOINT}"
echo "App URL: ${APP_URL}"
echo "Admin Account: ${ADMIN_EMAIL}"
echo ""
echo "Note: Service will be accessible from other machines on the network"
echo "To use localhost only, set HOST_IP=localhost before running this script"

# Create data directories
mkdir -p $MINIO_DATA
mkdir -p $APP_DATA

# Set permissions for container user (uid 1001)
# The container runs as appuser (uid 1001), so the data directory needs to be writable
podman unshare chown -R 1001:1001 $APP_DATA 2>/dev/null || \
    chmod -R 777 $APP_DATA

# Clean up existing pod and create new one
echo "Setting up Podman pod..."
podman pod rm -f $POD_NAME && /bin/true

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

# Build FilesOnTheGo application container
echo "Building FilesOnTheGo container image..."
podman build -t filesonthego:latest .

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
       --name filesonthego-app-dev \
       filesonthego:latest

# Wait for application to start
echo "Waiting for FilesOnTheGo application to start..."
max_wait=30
for i in $(seq 1 $max_wait); do
    if curl -s http://${HOST_IP}:8090/api/health > /dev/null 2>&1; then
        echo "Application is ready!"
        break
    fi
    if [ $i -eq $max_wait ]; then
        echo "Warning: Application did not become ready within ${max_wait} seconds"
    fi
    sleep 1
done

# Create admin account via API
echo "Attempting to create admin account..."
curl -s -X POST http://${HOST_IP}:8090/api/collections/users/records \
     -H "Content-Type: application/json" \
     -d "{\"email\":\"${ADMIN_EMAIL}\",\"username\":\"admin\",\"password\":\"${ADMIN_PASSWORD}\",\"passwordConfirm\":\"${ADMIN_PASSWORD}\"}" \
     > /dev/null 2>&1 && echo "Admin account created successfully" || echo "Note: Admin account may already exist"

echo ""
echo "=== Development Environment Ready ==="
echo ""
echo "Access from any machine on the network:"
echo "  Application: http://${HOST_IP}:${APP_PORT}"
echo "  MinIO Console: http://${HOST_IP}:9001"
echo "  MinIO API: http://${HOST_IP}:9000"
echo ""
echo "Access from this machine (localhost):"
echo "  Application: http://localhost:${APP_PORT}"
echo "  MinIO Console: http://localhost:9001"
echo ""
echo "Admin Login Details:"
echo "  Email: ${ADMIN_EMAIL}"
echo "  Password: ${ADMIN_PASSWORD}"
echo ""
echo "MinIO Console Details:"
echo "  Username: ${MINIO_ROOT_USER}"
echo "  Password: ${MINIO_ROOT_PASSWORD}"
echo ""
echo "To view logs:"
echo "  podman logs -f filesonthego-app-dev"
echo "  podman logs -f filesonthego-minio-dev"
echo ""
echo "To stop the environment:"
echo "  podman pod stop ${POD_NAME}"
