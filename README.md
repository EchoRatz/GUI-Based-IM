# GUI-Based Instant Messaging

A modern, real-time instant messaging application built with Go backend and web frontend, featuring JWT authentication and MongoDB storage.

## 🚀 Features

- **Passwordless Authentication**: Simple username-based authentication with JWT tokens
- **Real-time Messaging**: Built-in support for instant communication
- **MongoDB Integration**: Persistent storage with proper indexing
- **Docker Support**: Easy deployment with Docker Compose
- **REST API**: Clean RESTful API design
- **Health Monitoring**: Built-in health check endpoints

## 🛠️ Tech Stack

### Backend
- **Go** with Gin framework
- **MongoDB** for data persistence
- **JWT** for authentication
- **Docker** for containerization

### Frontend
- **HTML5/CSS3/JavaScript**
- Modern responsive design

## 📋 Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- MongoDB (if running locally without Docker)

## 🚀 Quick Start

### Using Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone https://github.com/EchoRatz/GUI-Based-IM.git
   cd GUI-Based-IM
   ```

2. **Start the services**
   ```bash
   docker-compose up -d
   ```

3. **Access the application**
   - Frontend: `http://localhost:3000` (or open `index.html`)
   - Backend API: `http://localhost:8080`

### Local Development

1. **Start MongoDB**
   ```bash
   docker run -d -p 27017:27017 --name mongo mongo:7.0.5
   ```

2. **Run the backend**
   ```bash
   cd backend
   go mod download
   go run .
   ```

3. **Open the frontend**
   Open `index.html` in your browser

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection string |
| `JWT_SECRET` | `dev-secret-key-change-me` | JWT signing secret (change in production!) |

### Docker Compose Override

Create a `docker-compose.override.yml` for custom configurations:

```yaml
services:
  backend:
    environment:
      - JWT_SECRET=your-super-secure-secret
      - MONGO_URI=mongodb://mongo:27017/your-db-name
```

## 📚 API Documentation

### Authentication

#### Claim Username (Sign up/Sign in)
```http
POST /claim
Content-Type: application/json

{
  "username": "alice_01"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "64f8a1b2c3d4e5f6789a0b1c",
    "username": "alice_01"
  }
}
```

#### Get Current User
```http
GET /me
Authorization: Bearer <token>
```

### Health Checks

#### Ping
```http
GET /ping
```

#### Database Health
```http
GET /health/db
```

## 🏗️ Project Structure

```
GUI-Based-IM/
├── backend/
│   ├── auth.go          # Authentication logic and JWT handling
│   ├── db.go            # MongoDB connection and utilities
│   ├── main.go          # Main server and route definitions
│   ├── go.mod           # Go module dependencies
│   ├── go.sum           # Go module checksums
│   └── Dockerfile       # Backend container configuration
├── docs/                # Documentation (future use)
├── docker-compose.yml   # Multi-container setup
├── index.html          # Frontend interface
└── README.md           # This file
```

## 🔐 Security Features

### Username Validation
- **Pattern**: `^[a-zA-Z0-9_]{3,20}$`
- **Reserved names**: admin, root, system, etc.
- **Case-insensitive**: Usernames are normalized to lowercase

### JWT Authentication
- **Algorithm**: HS256
- **Default TTL**: 24 hours
- **Claims**: user_id, username, issued_at, expires_at

## 🧪 Development

### Running Tests
```bash
cd backend
go test ./...
```

### Building
```bash
cd backend
go build -o im-backend
```

### Database Schema

#### Users Collection
```json
{
  "_id": "ObjectId",
  "username": "string (unique index)",
  "created_at": "int64 (unix timestamp)",
  "last_seen": "int64 (unix timestamp)"
}
```

## 🚀 Deployment

### Production Considerations

1. **Environment Variables**
   - Set a strong `JWT_SECRET`
   - Configure proper `MONGO_URI` with authentication
   
2. **Security**
   - Use HTTPS in production
   - Implement rate limiting
   - Add CORS configuration
   - Set up proper firewall rules

3. **Monitoring**
   - Use the `/health/db` endpoint for health checks
   - Monitor MongoDB performance
   - Set up logging and metrics

### Example Production docker-compose.yml
```yaml
services:
  mongo:
    image: mongo:7.0.5
    restart: always
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD}
    volumes:
      - mongo-data:/data/db
    networks:
      - im-network

  backend:
    build: ./backend
    restart: always
    environment:
      - MONGO_URI=mongodb://admin:${MONGO_PASSWORD}@mongo:27017/im?authSource=admin
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - mongo
    networks:
      - im-network

volumes:
  mongo-data:

networks:
  im-network:
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🐛 Known Issues

- Frontend is currently minimal and needs enhancement
- Real-time messaging features are planned but not yet implemented
- Rate limiting is not implemented

## 📞 Support

- Create an issue on GitHub
- Check existing documentation in the `docs/` folder
- Review the API endpoints using the health check endpoints

## 🛣️ Roadmap

- [ ] Real-time messaging with WebSocket
- [ ] Chat rooms and channels
- [ ] File sharing capabilities
- [ ] Enhanced frontend UI/UX
- [ ] Mobile responsiveness
- [ ] Message history and search
- [ ] User presence indicators
- [ ] Push notifications

---

**Built with ❤️ by the GUI-Based-IM team**