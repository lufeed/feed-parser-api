# LuFeed Parser API

A high-performance REST API built with Go for parsing RSS/Atom feeds and extracting comprehensive source information from URLs. The API provides endpoints to analyze feeds and extract metadata, making it easy to integrate feed parsing capabilities into your applications.

## Features

- 🚀 **Fast URL Parsing**: Extract feed information from any URL
- 📡 **Source Analysis**: Comprehensive source metadata extraction
- 🔐 **API Key Authentication**: Secure access with Bearer token authentication
- 📊 **Rate Limiting**: Built-in rate limiting for API protection
- 🏥 **Health Monitoring**: Health check endpoints for service monitoring
- 📝 **OpenAPI Documentation**: Complete API specification included
- 🐳 **Docker Support**: Containerized deployment ready

## Quick Start

### Installation

1. Clone the repository:
```bash
git clone https://github.com/lufeed/feed-parser-api.git
cd feed-parser-api
```

2. Install dependencies:
```bash
go mod download
```

3. Set up configuration:
```bash
export CONFIG_FILE=config-local.yml
```

4. Run the application:
```bash
go run cmd/server/main.go
```

The API will be available at `http://localhost:7654/api`

### Docker Deployment

```bash
docker build -t lufeed-parser-api .
docker run -p 7654:7654 -e CONFIG_FILE=config-prod.yml lufeed-parser-api
```

## Configuration

Create a configuration file (e.g., `config-local.yml`):

```yaml
service:
  name: lufeed-feed-parser-api
  environment: development

server:
  host: 0.0.0.0
  port: 7654
  root_path: "/api"

log:
  level: debug
  format: json

auth:
  api_keys:
    - your-api-key-1
    - your-api-key-2
```

## API Usage

### Authentication

All API endpoints (except `/ping`) require authentication using API keys:

```bash
curl -H "Authorization: Bearer your-api-key" \
     -H "Content-Type: application/json" \
     https://DOMAIN_NAME/api/v1/parsing/url
```

### Endpoints

#### Health Check
```http
GET /ping
```

**Response:**
```json
{
  "message": "pong",
  "service": "lufeed-feed-parser-api"
}
```

#### Parse URL
```http
POST /v1/parsing/url
Content-Type: application/json
Authorization: Bearer your-api-key

{
  "url": "https://example.com/feed.xml"
}
```

**Response:**
```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "title": "Tech News Daily",
    "description": "Latest technology news and updates",
    "url": "https://example.com/feed.xml",
    "image_url": "https://example.com/image.jpg",
    "published_at": "2023-12-01T10:30:00Z"
  }
}
```

#### Parse Source
```http
POST /v1/parsing/source
Content-Type: application/json
Authorization: Bearer your-api-key

{
  "url": "https://example.com"
}
```

**Response:**
```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "Tech News Site",
    "description": "A comprehensive technology news website",
    "feed_url": "https://example.com/feed.xml",
    "home_url": "https://example.com",
    "image_url": "https://example.com/logo.jpg",
    "icon_url": "https://example.com/favicon.ico"
  }
}
```

### Error Responses

```json
{
  "message": "Invalid URL format"
}
```

Common HTTP status codes:
- `200` - Success
- `400` - Bad Request (invalid URL or request body)
- `401` - Unauthorized (missing or invalid API key)
- `429` - Too Many Requests (rate limit exceeded)
- `500` - Internal Server Error

## API Documentation

Complete OpenAPI 3.0 specification is available in [`openapi.yaml`](./openapi.yaml). You can use tools like Swagger UI or Postman to explore the API interactively.

### Using with Swagger UI

```bash
# Serve the OpenAPI spec with Swagger UI
npx swagger-ui-serve openapi.yaml
```

## Development

### Project Structure

```
├── api/                    # API layer
│   ├── initialize.go      # API initialization
│   └── v1/               # Version 1 endpoints
│       ├── parsing/      # Parsing endpoints
│       └── init.go       # Route setup
├── cmd/
│   └── server/           # Application entry point
├── internal/             # Internal packages
│   ├── cache/           # Redis caching
│   ├── config/          # Configuration management
│   ├── database/        # Database connections
│   ├── logger/          # Logging utilities
│   ├── middleware/      # HTTP middleware
│   ├── models/          # Data models
│   ├── parser/          # URL/feed parsing logic
│   └── types/           # Common types
├── openapi.yaml         # API specification
└── README.md           # This file
```

### Building

```bash
go build -o bin/server cmd/server/main.go
```


### Health Checks

Use the `/ping` endpoint for health monitoring:

```bash
curl http://localhost:7654/api/ping
```

### Logging

Structured JSON logging is available with configurable levels:
- `debug` - Detailed debugging information
- `info` - General information
- `warn` - Warning messages
- `error` - Error messages

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support, please open an issue on GitHub or contact the development team.
