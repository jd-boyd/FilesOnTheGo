#!/bin/bash
# FilesOnTheGo Development Environment Setup
# This script sets up MinIO and FilesOnTheGo service in a podman pod for local development

set -e
set -x

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
echo "S3 Endpoint: ${S3_ENDPOINT}"
echo "App URL: ${APP_URL}"
echo "Admin Account: ${ADMIN_EMAIL}"

# Create data directories
mkdir -p $MINIO_DATA
mkdir -p $APP_DATA

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
sleep 3

# Create MinIO bucket and set access policy
echo "Setting up MinIO bucket..."
podman run \
       --pod $POD_NAME \
       --entrypoint=/bin/sh \
       quay.io/minio/mc -c "\
      /usr/bin/mc alias set myminio http://localhost:9000 ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} && \
      /usr/bin/mc mb --ignore-existing myminio/${MINIO_BUCKET} && \
      /usr/bin/mc anonymous set download myminio/${MINIO_BUCKET} && \
      echo 'MinIO bucket setup complete'"

# Verify MinIO setup
podman run \
       --pod $POD_NAME \
       --entrypoint=/bin/sh \
       quay.io/minio/mc -c "\
      /usr/bin/mc alias set myminio http://localhost:9000 ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} && \
      /usr/bin/mc ls myminio/ && \
      echo 'MinIO verification complete'"

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
sleep 10

# Create admin account using PocketBase CLI if it exists
echo "Attempting to create admin account..."
podman run \
       --pod $POD_NAME \
       -v $APP_DATA:/app/data \
       --entrypoint=/bin/sh \
       filesonthego:latest -c "\
      /app/filesonthego admin create ${ADMIN_EMAIL} ${ADMIN_PASSWORD} 2>/dev/null && \
      echo 'Admin account created successfully' || \
      echo 'Note: Admin account creation failed - you may need to create it through the web interface'"

echo ""
echo "=== Development Environment Ready ==="
echo "FilesOnTheGo Application: http://localhost:${APP_PORT}"
echo "MinIO Console: http://localhost:9001"
echo "MinIO API: http://localhost:9000"
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
