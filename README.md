# Energy Plant Monitoring System

## Overview

The Energy Plant Monitoring System is a robust, scalable microservice designed to monitor and analyze energy production plants in real-time. It collects sensor data from various energy sources, processes events asynchronously, and provides analytics and alerting capabilities. The system supports geospatial data for plant locations and uses time-series databases for efficient historical data storage and querying.

The application follows hexagonal architecture principles, ensuring clean separation of concerns, testability, and maintainability. It integrates with Kafka for event-driven messaging, PostgreSQL with TimescaleDB for data persistence, and provides REST APIs for data access and webhook notifications for external integrations.

## Key Features

- **Real-time Event Processing**: Consumes events from Kafka topics containing sensor data from energy plants.
- **Time-series Data Storage**: Utilizes TimescaleDB for efficient storage and querying of time-series data.
- **Geospatial Support**: Integrates PostGIS for location-based queries and plant mapping.
- **RESTful API**: Provides endpoints for retrieving plant data, events, and analytics.
- **Webhook Notifications**: Sends real-time notifications to external systems when events occur.
- **Alerting System**: Telegram bot integration for error notifications and validation alerts.
- **Analytics and Reporting**: Background workers process data for analytics, continuous aggregates, and reporting.
- **Data Retention Policies**: Automatic data cleanup based on configurable retention rules.
- **Health Monitoring**: Built-in health checks for liveness and readiness probes.

## Architecture

The system is built using hexagonal (ports and adapters) architecture, which promotes loose coupling between business logic and external dependencies.

### Core Components

- **Domain Layer**: Contains business entities (Plants, Events, Analytics) and core business logic.
- **Application Layer**: Defines service interfaces (ports) for use cases like event processing and data retrieval.
- **Infrastructure Layer**: Implements adapters for external systems (Kafka, Database, HTTP clients).

### Data Flow

1. **Event Ingestion**: Sensor data from energy plants is published to Kafka topics.
2. **Event Processing**: The system consumes events, validates them, and stores them in the database.
3. **Data Storage**: Events are stored in TimescaleDB with hypertables for optimal time-series performance.
4. **Analytics Processing**: Background workers aggregate data for analytics and reporting.
5. **API Serving**: REST endpoints serve processed data to clients.
6. **Notifications**: Webhooks and Telegram alerts are triggered based on event processing and errors.

### Database Schema

The system uses a multi-schema PostgreSQL database:

- **Master Schema**: Contains plant metadata, configurations, and reference data.
- **Operational Schema**: Stores real-time event data and current status information.
- **Analytics Schema**: Houses aggregated data, reports, and historical analytics.

## Tech Stack

- **Go 1.24**: Primary programming language
- **Gin**: HTTP web framework for REST APIs
- **GORM**: ORM for database interactions
- **Kafka (confluent-kafka-go)**: Message broker for event streaming
- **PostgreSQL 15** with extensions:
  - **TimescaleDB 2.24**: Time-series database extension
  - **PostGIS 3.6**: Geospatial data support
  - **pgcrypto**: Cryptographic functions
- **Goose**: Database migration tool
- **Atlas**: Schema migration generation from GORM models
- **Docker & Docker Compose**: Containerization and local development

## Project Structure

```
├── cmd/
│   └── atlasloader/          # Entity loader for Atlas schema generation
├── db/
│   ├── Dockerfile            # PostgreSQL image with extensions
│   └── init-db.sql           # Database initialization script
├── internal/
│   ├── api/                  # Application services (Kafka consumer, etc.)
│   ├── domain/
│   │   ├── entities/         # Domain models (Plant, Event, etc.)
│   │   └── ports/            # Service and repository interfaces
│   └── infrastructure/
│       ├── adapters/         # External system adapters
│       │   ├── kafka/        # Kafka producer/consumer
│       │   ├── repositories/ # Database repositories
│       │   ├── rest/         # HTTP handlers and routing
│       │   └── webhook/      # Webhook client
│       ├── conf/             # Configuration factories
│       └── container/        # Dependency injection container
├── migrations/               # Database migration files
├── scripts/                  # Utility scripts and test data generators
└── main.go                   # Application entry point
```

## Local Development Setup

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- Make

### Quick Start

1. **Clone the repository and navigate to the project directory**

2. **Install development tools:**
   ```bash
   make install-dev-tools
   ```

3. **Start infrastructure services:**
   ```bash
   make docker-up
   ```

4. **Configure environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

5. **Run database migrations:**
   ```bash
   make goose-up
   ```

6. **Start the application:**
   ```bash
   make dev  # For development with hot reload
   ```

### Available Services

| Service | Port | Description |
|---------|------|-------------|
| PostgreSQL | 5432 | Main database |
| pgAdmin | 5050 | Database administration UI |
| Kafka | 9092 | Message broker |
| Kafka UI | 8080 | Kafka management UI |
| Zookeeper | 2181 | Kafka coordination service |

## API Documentation

The system provides RESTful APIs for data access and management.

### Core Endpoints

- `GET /api/v1/plants` - List energy plants
- `GET /api/v1/plants/{id}` - Get plant details
- `GET /api/v1/events` - Query events with filtering
- `POST /api/v1/webhook` - Webhook receiver for external integrations
- `GET /healthz` - Liveness probe
- `GET /readyz` - Readiness probe

### Swagger Documentation

Interactive API documentation is available at `http://localhost:9000/swagger/index.html` when running in development mode.

## Configuration

The application is configured via environment variables:

### Server Configuration
- `PORT`: Server port (default: 9000)
- `ENVIRONMENT`: Environment (dev/prod)

### Database Configuration
- `DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_NAME`, `DATABASE_USER`, `DATABASE_PASSWORD`
- `DATABASE_SCHEMA`: Target schema

### Kafka Configuration
- `LIST_KAFKA_BROKERS`: Kafka broker addresses
- `CONSUMER_GROUP`: Consumer group ID
- `CONSUMER_TOPIC`: Input topic for events
- `PRODUCER_TOPIC`: Output topic for processed events

### Notification Configuration
- `WEBHOOK_ENABLED`: Enable webhook notifications
- `WEBHOOK_URL`: Webhook endpoint URL
- `TELEGRAM_ENABLED`: Enable Telegram alerts
- `TELEGRAM_BOT_TOKEN`: Telegram bot token
- `TELEGRAM_CHAT_ID`: Telegram chat ID

## Event Processing Logic

### Event Validation
Events are validated upon consumption:
- Required fields presence (e.g., `plant_source_id`)
- UUID format validation for plant identifiers
- Plant existence verification in database

### Error Handling
Validation errors trigger:
- Logging with detailed context
- Telegram notifications (if enabled)
- Event rejection without storage

### Data Processing
Valid events are:
- Stored in operational schema
- Processed by analytics workers
- Made available via REST APIs
- Triggered webhooks for real-time notifications

## Analytics and Workers

Background workers perform:
- **Continuous Aggregates**: Automatic data summarization for performance
- **Data Retention**: Cleanup of old data based on policies
- **Alert Rules**: Evaluation of conditions for notifications
- **Geospatial Analysis**: Location-based insights and reporting

## Monitoring and Alerting

- **Health Checks**: Kubernetes-compatible liveness and readiness endpoints
- **Error Notifications**: Telegram bot integration for critical errors
- **Metrics**: Integration with Prometheus for observability
- **Tracing**: OpenTelemetry support for distributed tracing

## Deployment

The application is containerized and can be deployed using Docker:

```bash
make build
docker build -t energy-plant-monitor .
docker run -p 9000:9000 energy-plant-monitor
```

For production deployments, use the provided Docker Compose configuration or Kubernetes manifests.

## Contributing

1. Follow the hexagonal architecture patterns
2. Write tests for new functionality
3. Update API documentation for new endpoints
4. Ensure database migrations are properly versioned
5. Test with realistic data volumes using the provided scripts

## License

[Specify your license here]
