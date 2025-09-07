# Planning Poker

A full-stack Planning Poker application for agile teams, built with Go (backend) and Next.js/React (frontend).

## Features

- Real-time collaborative estimation using WebSockets
- User roles: owner, participant, spectator
- Modern UI with Next.js and React
- Dockerized for easy deployment

## Project Structure

```
planning-poker/
├── cmd/api/                # Go backend entrypoint
├── internal/               # Go backend logic
│   └── planningpoker/      # Core planning poker logic
├── frontend/
│   └── planning-poker-front/ # Next.js frontend app
├── Dockerfile              # Backend Dockerfile
├── docker-compose.yml      # Multi-service orchestration
├── Makefile                # Common dev commands
├── example.env             # Example environment variables
└── README.md
```

## Prerequisites

- Docker & Docker Compose
- (For local dev) Go 1.24+, Node.js 18+, npm

## Running with Docker

To build and run both backend and frontend:

```bash
docker-compose up --build
```

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080

## Local Development

### Backend

```bash
make init      # Install Go and frontend dependencies
make run       # Run backend server
make test      # Run Go tests
```

### Frontend

```bash
make run-frontend
```

## Environment Variables

See `example.env` and `frontend/planning-poker-front/example.env` for configuration.

## Testing

- Backend: `make test`
- Frontend: Use your preferred React/Next.js testing tools

## Building for Production

- Backend: `make build` or use the provided Dockerfile
- Frontend: `npm run build` in the frontend directory or use the provided Dockerfile

## License

MIT
