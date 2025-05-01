# Firebase Firestore Forwarder Server

This is a Golang backend server that acts as a forwarder between mobile applications and Firebase Firestore. It provides a secure and controlled interface for mobile clients to interact with Firestore data.

## Features

- Authentication middleware using Firebase Auth tokens
- CRUD operations for Firestore documents
- Collection querying with pagination
- Request validation
- Swagger API documentation
- JSON API for authentication
- Docker support for easy deployment

## Prerequisites

- Go 1.19+
- Firebase project with Firestore and Authentication enabled
- Firebase service account credentials (JSON file)

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/ajipaon/firestore-admin-client.git
cd firestore-admin-client
```

### 2. Set up Firebase credentials

Download your Firebase service account JSON file from the Firebase Console:

1. Go to Firebase Console > Project Settings > Service Accounts
2. Click "Generate new private key"
3. Save the file as `firebase-credential.json` in the project root

### 3. Run locally

```bash
go mod download
go run main.go
```

By default, the server will start on port 8080. You can customize this by setting the `PORT` environment variable.

### 3.1. Development with Auto-Reload

For development, you can use Air to automatically reload the server when code changes are detected:

1. Install Air:

```bash
# Using go install
go install github.com/cosmtrek/air@latest

# Or using curl (Linux/macOS)
curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

2. Run the server with Air:

```bash
air
```

Air will watch for file changes in your project and automatically rebuild and restart the server when changes are detected. This makes development much faster as you don't need to manually restart the server after each code change.

The configuration for Air is in the `.air.toml` file in the project root.

### 3.2. Development with Docker and Auto-Reload

For development with Docker and auto-reload:

````bash
# Build the development image
docker build -t firestore-forwarder-dev -f Dockerfile.dev .
# Run the container with volume mounting for live code changes

```bash
docker run -p 8080:8080 \
  -v $(pwd):/app \
  -e PORT=8080 \
  -e FIREBASE_CRED_FILE=firebase-credential.json \
  firestore-forwarder-dev
````

### 4.1. Run Production Container from Docker Hub

You can also run the production-ready image directly from Docker Hub:

```bash
docker pull ajipaon/firestore-admin-client:latest
docker run -p 8080:8080 \
  -e PORT=8080 \
  -e FIREBASE_CRED_FILE=firebase-credential.json \
  ajipaon/firestore-admin-client:latest
```

This will start the application in a Docker container with Air watching for file changes. Any changes you make to the code will trigger an automatic rebuild and restart of the server.

### 4. Build and run with Docker

```bash
docker build -t firestore-admin-client .
docker run -p 8080:8080 -e PORT=8080 firestore-admin-client
```

## Environment Variables

- `PORT`: Server port (default: 8080)
- `FIREBASE_CRED_FILE`: Path to Firebase credentials file (default: firebase-credential.json)
- `FIREBASE_API_KEY`: Firebase Web API Key (required only for email/password authentication via the login page)

## API Documentation

The API is documented using Swagger. You can access the Swagger UI at:

```
http://localhost:8080/swagger/
```

This provides an interactive documentation where you can:

- View all available endpoints
- See request/response schemas
- Test the API directly from the browser

## API Endpoints

All API endpoints require Firebase authentication token in the Authorization header:

```
Authorization: Bearer <firebase-id-token>
```

### Authentication

- `POST /auth/login`: Login with email and password (HTML response)
- `POST /auth/login/json`: Login with email and password (JSON response)
  - Request body:
    ```json
    {
      "email": "user@example.com",
      "password": "password123"
    }
    ```
  - Response:
    ```json
    {
      "success": true,
      "message": "Authentication successful",
      "data": {
        "token": "<firebase-id-token>"
      }
    }
    ```

### Health Check

- `GET /health`: Check if the server is running (no auth required)

### Collection Operations

- `GET /api/collections/{collection}/documents`: Query documents from a collection

  - Query params:
    - `limit`: Maximum number of documents to return (default: 50)

- `POST /api/collections/{collection}/documents`: Create a new document
  - Body: JSON object with document data

### Document Operations

- `GET /api/collections/{collection}/documents/{id}`: Get a specific document
- `PUT /api/collections/{collection}/documents/{id}`: Update a document (replace)
- `PATCH /api/collections/{collection}/documents/{id}`: Update a document (merge)
- `DELETE /api/collections/{collection}/documents/{id}`: Delete a document

## Security Considerations

- All requests to the Firestore are made through the backend server using Firebase Admin SDK
- The server validates Firebase ID tokens for authentication
- The server can implement additional authorization logic beyond what Firebase provides
- Mobile clients never need direct access to Firestore
- You can add additional middleware for rate limiting, logging, etc.
