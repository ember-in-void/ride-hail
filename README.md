# 🚗 Ride-Hailing System

> Production-ready microservices-based ride-hailing system with real-time communication, geospatial matching, and event-driven architecture.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql)](https://postgresql.org)
[![RabbitMQ](https://img.shields.io/badge/RabbitMQ-3.13-FF6600?style=flat&logo=rabbitmq)](https://rabbitmq.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 📋 Table of Contents

- [Overview](#-overview)
- [Architecture](#️-architecture)
- [📚 Documentation](#-developer-documentation)
- [Quick Start](#-quick-start)
- [System Verification](#-system-verification)
- [API Documentation](#-api-documentation)
- [WebSocket](#-websocket)
- [RabbitMQ](#-rabbitmq)
- [Database](#️-database)
- [Testing](#-testing)
- [Troubleshooting](#-troubleshooting)

---

## 🌟 Overview

**Ride-Hailing System** is a production-ready backend for a taxi/rideshare platform, built using modern architectural patterns and technologies.

### Key Features

✨ **Real-time Communication**
- WebSocket connections for passengers and drivers
- Instant matching notifications
- Live driver location tracking

🗺️ **Geospatial Matching**
- PostGIS for finding nearest drivers
- Radius search (5km) with ST_DWithin
- Optimization through GIST indexes

📨 **Event-Driven Architecture**
- RabbitMQ for asynchronous communication
- Topic and Fanout exchanges
- Automatic retry and error handling

🏗️ **Clean Architecture**
- Hexagonal Pattern (Ports & Adapters)
- SOLID principles
- Complete framework independence

---

## 📚 Developer Documentation

### 🎓 For Beginners

If you're new to the project, we recommend studying the documentation in this order:

1. **[ARCHITECTURE_FLOW.md](docs/ARCHITECTURE_FLOW.md)** (450+ lines) ⭐ **START HERE!**
   - 🏠 Clean Architecture house metaphor
   - 📊 Visual data flow diagrams
   - 👣 Step-by-step code execution (7 steps)
   - 🛡️ Error protection mechanisms
   - 📖 Technical terms glossary
   - 💡 Debugging tips

2. **[DIAGRAMS.md](docs/DIAGRAMS.md)** (300+ lines)
   - 🔄 Sequence Diagram (Mermaid)
   - 🏗️ Component Diagram
   - 📈 Data Flow Diagram
   - ⚠️ Error Handling Flow
   - 🛡️ Race Condition Prevention
   - 🗄️ Database Schema

3. **Code Comments** (500+ lines)
   - All critical files have detailed comments
   - Explanations of "what", "why", and "how"
   - Usage examples
   - Step-by-step breakdowns (STEP 1, STEP 2, ...)

### 🏗️ For Experienced Developers

4. **[CODE_STANDARDS.md](docs/CODE_STANDARDS.md)** (400+ lines)
   - 🎯 Clean Architecture principles
   - 📝 SOLID in practice
   - ⚠️ Error handling
   - 🧪 Test examples
   - ✅ Pre-commit checklist

5. **[DOCUMENTATION_SUMMARY.md](docs/DOCUMENTATION_SUMMARY.md)**
   - 📊 Documentation statistics
   - ✅ What has been done
   - 🚀 Next steps

### 📖 Additional Documentation

- **[docs_architecture.md](docs/docs_architecture.md)** - Technical architecture
- **[admin_api.md](docs/admin_api.md)** - Admin API
- **[INTEGRATION.md](docs/INTEGRATION.md)** - Component integration
- **[reglament.md](docs/reglament.md)** - Project regulations

---


### Technology Stack

**Backend:**
- Go 1.24+
- PostgreSQL 16 + PostGIS
- RabbitMQ 3.13
- gorilla/websocket
- pgx/v5
- golang-jwt/jwt/v5

**Infrastructure:**
- Docker & Docker Compose
- Health checks
- Graceful shutdown

---

## 🏗️ Architecture

### Microservices

```
┌─────────────────────────────────────────────────────────┐
│                   RIDE-HAILING SYSTEM                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ Ride Service │  │Driver Service│  │Admin Service │ │
│  │   :3000      │  │    :3001     │  │    :3002     │ │
│  │              │  │              │  │              │ │
│  │ • Rides      │  │ • Drivers    │  │ • Users      │ │
│  │ • Passengers │  │ • Location   │  │ • Overview   │ │
│  │ • WebSocket  │  │ • Matching   │  │ • Analytics  │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘ │
│         │                 │                  │         │
│         └─────────────────┴──────────────────┘         │
│                           │                            │
│         ┌─────────────────┴─────────────────┐         │
│         │                                   │         │
│    ┌────▼─────┐                      ┌─────▼──────┐  │
│    │PostgreSQL│                      │  RabbitMQ  │  │
│    │  :5432   │                      │   :5672    │  │
│    │ + PostGIS│                      │ Management │  │
│    └──────────┘                      │   :15672   │  │
│                                      └────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### System Components

#### 1. **Ride Service** (port 3000)
- Ride management
- WebSocket for passengers
- RabbitMQ consumers (location updates, driver responses)
- REST API for ride creation

#### 2. **Driver Service** (port 3001)
- Driver management
- WebSocket for drivers
- PostGIS matching algorithm
- RabbitMQ consumers (ride requests)
- Location tracking

#### 3. **Admin Service** (port 3002)
- Administrative dashboard
- User management
- Statistics and analytics

#### 4. **PostgreSQL + PostGIS**
- Main database
- Geospatial queries
- Storage for rides, drivers, coordinates

#### 5. **RabbitMQ**
- Message broker
- 3 exchanges: ride_topic, driver_topic, location_fanout
- Event-driven communication between services

---

## 🚀 Quick Start

### Prerequisites

- Go 1.24+ ([installation](https://golang.org/dl/))
- Docker and Docker Compose ([installation](https://docs.docker.com/get-docker/))
- jq for test scripts (optional)

### Step 1: Clone the Repository

```bash
git clone https://github.com/ember-in-void/ride-hail.git
cd ride-hail
```

### Step 2: Start Infrastructure

```bash
cd deployments
docker compose up -d
```

This will start:
- ✅ PostgreSQL on port 5432
- ✅ RabbitMQ on ports 5672 (AMQP) and 15672 (Management UI)

Check status:
```bash
docker compose ps

# Should be running:
# ridehail-postgres
# ridehail-rabbitmq
```

### Step 3: Build the Project

```bash
cd ..  # return to project root
go build -o bin/ridehail ./main.go
```

Verify build:
```bash
ls -lh bin/ridehail
# Should create an executable file ~16MB
```

### Step 4: Start Services

Open **3 terminals** and start each service:

**Terminal 1 - Ride Service:**
```bash
./bin/ridehail
# Will start on port 3000
```

**Terminal 2 - Driver Service:**
```bash
SERVICE_MODE=driver ./bin/ridehail
# Will start on port 3001
```

**Terminal 3 - Admin Service:**
```bash
SERVICE_MODE=admin ./bin/ridehail
# Will start on port 3002
```

### Step 5: Verify Operation

```bash
# Check health of all services
curl http://localhost:3000/health  # Ride Service
curl http://localhost:3001/health  # Driver Service
curl http://localhost:3004/health  # Admin Service

# Expected response from each:
# {"status":"ok","service":"ride"}
# {"status":"ok","service":"driver"}
# {"status":"ok","service":"admin"}
```

---

## ✅ System Verification

### 1. Infrastructure Testing

#### PostgreSQL
```bash
# Connect to database
docker exec -it ridehail-postgres psql -U ridehail_user -d ridehail_db

# Check PostGIS
ridehail_db=# SELECT PostGIS_version();
# Should show PostGIS version

# Check tables
ridehail_db=# \dt
# Table list: users, drivers, rides, coordinates, location_history

# Check indexes
ridehail_db=# \di
# Should have idx_coordinates_geography (GIST)

# Exit
ridehail_db=# \q
```

#### RabbitMQ
```bash
# Open Management UI in browser
# http://localhost:15672
# Login: guest / Password: guest

# Check exchanges:
# - ride_topic (type: topic)
# - driver_topic (type: topic)
# - location_fanout (type: fanout)

# Check queues:
# - driver_matching
# - ride_service_driver_responses
# - ride_service_locations
```

### 2. WebSocket Connection Testing

```bash
# Run automatic test
chmod +x scripts/test-websocket.sh
./scripts/test-websocket.sh
```

Expected output:
```
========================================
Testing WebSocket Connections
========================================

[1/2] Testing Ride Service WebSocket...
  ✓ Connection successful (HTTP 101 Switching Protocols)

[2/2] Testing Driver Service WebSocket...
  ✓ Connection successful (HTTP 101 Switching Protocols)

========================================
✅ All WebSocket tests passed!
========================================
```

### 3. Driver API Testing

```bash
# Run full Driver Service test
chmod +x scripts/test-driver-api.sh
./scripts/test-driver-api.sh
```

This test verifies:
1. ✅ Driver creation via Admin API
2. ✅ GoOnline endpoint
3. ✅ UpdateLocation with PostGIS
4. ✅ Location publishing to RabbitMQ
5. ✅ GoOffline endpoint

### 4. End-to-End Full Flow Test

```bash
# Run E2E test
chmod +x scripts/test-e2e-ride-flow.sh
./scripts/test-e2e-ride-flow.sh
```

This test verifies the complete cycle:
1. ✅ JWT token generation for passenger and driver
2. ✅ User creation in database
3. ✅ Driver goes online
4. ✅ Driver updates location (Moscow: 55.7558, 37.6173)
5. ✅ Passenger creates ride (Red Square → Kremlin)
6. ✅ Ride Service publishes to RabbitMQ
7. → Driver Service finds driver with PostGIS (5km radius)
8. → Driver receives offer via WebSocket
9. → Driver responds via WebSocket
10. → Ride Service receives response
11. → Passenger receives notification

### 5. 🎬 Demo: Full Ride Cycle (Beautiful Output)

**New beautiful demo script with colored output and detailed logging!**

```bash
# Run beautiful full cycle demonstration
chmod +x scripts/demo-full-ride-cycle.sh
./scripts/demo-full-ride-cycle.sh
```

**What the demo shows:**

```
🚗 RIDE-HAILING SYSTEM - FULL CYCLE DEMONSTRATION 🚗

STEP 0:  ✓ Checking availability of all services
STEP 1:  ✓ Generating test UUIDs and data
STEP 2:  ✓ Creating JWT tokens (ADMIN, PASSENGER, DRIVER)
STEP 3:  👤 Creating passenger and 🚗 driver via Admin API
STEP 4:  🚗 Driver goes online (status → AVAILABLE)
STEP 5:  📍 Updating driver location (Almaty Central Park)
STEP 6:  👤 Passenger creates ride (Central Park → Kok-Tobe)
         🚀 RabbitMQ: ride.request.ECONOMY → driver_matching queue
         📊 PostGIS: ST_DWithin(5km) - driver search
STEP 7:  🚗 Driver receives and accepts offer
         🚀 RabbitMQ: driver.response → ride_service_driver_responses
STEP 8:  ⏱ Driver starts ride (status → IN_PROGRESS)
STEP 9:  📍 Movement simulation with location updates:
         • 43.235, 76.885 - Moving towards destination (25.5 km/h)
         • 43.230, 76.870 - Halfway there (35.2 km/h)
         • 43.225, 76.860 - Almost arrived (28.7 km/h)
         • 43.222, 76.851 - Arriving at destination (15.3 km/h)
STEP 10: 💰 Driver completes ride
         Distance: 5.2 km | Duration: 18 min
STEP 11: 📊 Checking Admin Dashboard (metrics and active rides)

✓ ALL STEPS SUCCESSFULLY COMPLETED!
```

**Demo script features:**
- 🎨 Beautiful colored output with emojis
- 📝 Detailed logging of each step
- ⚡ Automatic service availability checks
- 🔍 Output of all created UUIDs for debugging
- 📊 Final table with test results
- 🎯 Real driver movement simulation
- ✅ Verification of all system components

**Components tested:**
- JWT Authentication (3 roles)
- Admin Service (user creation, metrics)
- Driver Service (lifecycle, location, PostGIS)
- Ride Service (ride creation, RabbitMQ)
- RabbitMQ (3 exchanges, all queues)
- PostGIS (ST_DWithin geosearch within 5km radius)
- WebSocket simulation (ride offers & responses)

---

## 📡 API Documentation

### Ride Service (http://localhost:3000)

#### Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Health check | No |
| POST | `/rides` | Create ride | JWT (PASSENGER/ADMIN) |
| GET | `/ws` | WebSocket for passengers | JWT |

#### POST /rides - Create Ride

**Request:**
```bash
curl -X POST http://localhost:3000/rides \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 55.7558,
    "pickup_lng": 37.6173,
    "pickup_address": "Red Square, Moscow",
    "destination_lat": 55.7522,
    "destination_lng": 37.6156,
    "destination_address": "Kremlin, Moscow",
    "priority": 5
  }'
```

**Response (201 Created):**
```json
{
  "ride_id": "abc-123-def-456",
  "ride_number": "R-20251031-001",
  "status": "PENDING",
  "estimated_fare": 250.50,
  "pickup_address": "Red Square, Moscow",
  "destination_address": "Kremlin, Moscow"
}
```

### Driver Service (http://localhost:3001)

#### Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Health check | No |
| POST | `/drivers/{id}/online` | Выход онлайн | JWT (DRIVER) |
| POST | `/drivers/{id}/offline` | Выход оффлайн | JWT (DRIVER) |
| POST | `/drivers/{id}/location` | Обновить локацию | JWT (DRIVER) |
| POST | `/drivers/{id}/start` | Начать поездку | JWT (DRIVER) |
| POST | `/drivers/{id}/complete` | Завершить поездку | JWT (DRIVER) |
| GET | `/ws` | WebSocket для водителей | JWT |

#### POST /drivers/{id}/online - Go Online

**Request:**
```bash
curl -X POST http://localhost:3001/drivers/driver-123/online \
  -H "Authorization: Bearer YOUR_DRIVER_JWT_TOKEN" \
  -H "Content-Type: application/json"
```

**Response (200 OK):**
```json
{
  "driver_id": "driver-123",
  "status": "AVAILABLE",
  "is_online": true,
  "timestamp": "2025-10-31T12:00:00Z"
}
```

#### POST /drivers/{id}/location - Update Location

**Request:**
```bash
curl -X POST http://localhost:3001/drivers/driver-123/location \
  -H "Authorization: Bearer YOUR_DRIVER_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 55.7558,
    "longitude": 37.6173,
    "accuracy_meters": 10.0,
    "speed_kmh": 45.5,
    "heading_degrees": 180
  }'
```

**Response (200 OK):**
```json
{
  "success": true,
  "coordinate_id": "coord-789",
  "message": "Location updated successfully"
}
```

**What happens:**
1. Location is saved in PostgreSQL with PostGIS
2. Published to RabbitMQ exchange `location_fanout`
3. All subscribers (Ride Service) receive the update
4. Passengers receive notification via WebSocket

### Admin Service (http://localhost:3004)

#### Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Health check | No |
| POST | `/admin/users` | Создать пользователя | JWT (ADMIN) |
| GET | `/admin/users` | Список пользователей | JWT (ADMIN) |
| GET | `/admin/overview` | System overview | JWT (ADMIN) |
| GET | `/admin/rides/active` | Active rides | JWT (ADMIN) |

#### POST /admin/users - Create User

**Request:**
```bash
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "user-123",
    "email": "driver@example.com",
    "role": "DRIVER",
    "phone": "+79991234567"
  }'
```

**Response (201 Created):**
```json
{
  "id": "user-123",
  "email": "driver@example.com",
  "role": "DRIVER",
  "created_at": "2025-10-31T12:00:00Z"
}
```

---

## 🔐 JWT Authentication

### Token Generation

```bash
# Passenger
go run cmd/generate-jwt/main.go \
  --user-id "passenger-123" \
  --role "PASSENGER" \
  --ttl "24h"

# Driver
go run cmd/generate-jwt/main.go \
  --user-id "driver-456" \
  --role "DRIVER" \
  --ttl "24h"

# Administrator
go run cmd/generate-jwt/main.go \
  --user-id "admin-1" \
  --role "ADMIN" \
  --ttl "24h"
```

### Token Usage

```bash
# Save token
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Use in requests
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/rides
```

### Roles and Access

| Role | Access |
|------|--------|
| **PASSENGER** | Ride Service (ride creation, WebSocket) |
| **DRIVER** | Driver Service (status/location management, WebSocket) |
| **ADMIN** | Admin Service (user management, analytics) |

---

## 🔌 WebSocket

### Ride Service WebSocket (Passengers)

**Connection:**
```
ws://localhost:3000/ws?token=YOUR_JWT_TOKEN
```

**Incoming messages (from server):**

1. **Ride Status Update**
```json
{
  "type": "ride_status",
  "ride_id": "abc-123",
  "status": "DRIVER_ASSIGNED",
  "driver_id": "driver-456"
}
```

2. **Driver Location Update**
```json
{
  "type": "driver_location",
  "ride_id": "abc-123",
  "latitude": 55.7558,
  "longitude": 37.6173,
  "heading": 180,
  "speed": 45.5
}
```

3. **Match Notification**
```json
{
  "type": "ride_matched",
  "ride_id": "abc-123",
  "driver_id": "driver-456",
  "estimated_arrival_minutes": 5,
  "driver_info": {
    "name": "John Doe",
    "rating": 4.8,
    "vehicle": {
      "make": "Toyota",
      "model": "Camry",
      "color": "Black",
      "plate": "A123BC77"
    }
  }
}
```

### Driver Service WebSocket (Drivers)

**Connection:**
```
ws://localhost:3001/ws?token=YOUR_DRIVER_JWT_TOKEN
```

**Incoming messages (from server):**

1. **Ride Offer**
```json
{
  "type": "ride_offer",
  "ride_id": "abc-123",
  "pickup_location": {
    "lat": 55.7558,
    "lng": 37.6173,
    "address": "Red Square"
  },
  "destination_location": {
    "lat": 55.7522,
    "lng": 37.6156,
    "address": "Kremlin"
  },
  "estimated_fare": 250.50,
  "distance_km": 2.5
}
```

**Outgoing messages (from client):**

1. **Accept Ride**
```json
{
  "type": "ride_response",
  "ride_id": "abc-123",
  "accepted": true,
  "current_location": {
    "latitude": 55.7600,
    "longitude": 37.6200
  }
}
```

2. **Location Update**
```json
{
  "type": "location_update",
  "latitude": 55.7558,
  "longitude": 37.6173,
  "accuracy_meters": 10.0,
  "speed_kmh": 45.5,
  "heading_degrees": 180
}
```

### WebSocket Testing

```bash
# Automatic test
./scripts/test-websocket.sh

# Manual testing with websocat
# Installation: cargo install websocat

# Passenger
websocat "ws://localhost:3000/ws?token=$PASSENGER_TOKEN"

# Driver
websocat "ws://localhost:3001/ws?token=$DRIVER_TOKEN"
```

---

## 📨 RabbitMQ

### Topology

```
Exchanges:
├─ ride_topic (topic)
│  ├─ Routing: ride.request.*
│  └─ Queue: driver_matching
│
├─ driver_topic (topic)
│  ├─ Routing: driver.response.*
│  └─ Queue: ride_service_driver_responses
│
└─ location_fanout (fanout)
   ├─ Queue: ride_service_locations
   └─ Queue: driver_service_locations (optional)
```

### Message Flows

#### 1. Ride Request Flow
```
POST /rides
    ↓
Ride Service → ride_topic (ride.request.{ride_id})
    ↓
driver_matching queue
    ↓
Driver Service Consumer
    ↓
PostGIS: ST_DWithin(5km) + ST_Distance
    ↓
WebSocket → Driver (ride offer)
```

#### 2. Driver Response Flow
```
WebSocket ← Driver (ride_response)
    ↓
Driver Service
    ↓
driver_topic (driver.response.{ride_id})
    ↓
ride_service_driver_responses queue
    ↓
Ride Service Consumer
    ↓
WebSocket → Passenger (match notification)
```

#### 3. Location Update Flow
```
POST /drivers/{id}/location
    ↓
Driver Service
    ↓
location_fanout (broadcast)
    ↓
├─ ride_service_locations queue
│  ↓
│  Ride Service Consumer
│  ↓
│  WebSocket → Passenger
└─ (other subscribers)
```

### RabbitMQ Verification

```bash
# Open Management UI
# http://localhost:15672 (guest/guest)

# Check exchanges
curl -u guest:guest http://localhost:15672/api/exchanges

# Check queues
curl -u guest:guest http://localhost:15672/api/queues

# Check bindings
curl -u guest:guest http://localhost:15672/api/bindings
```

---

## 🗄️ Database

### Schema Overview

```sql
-- Users (all types: PASSENGER, DRIVER, ADMIN)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Drivers
CREATE TABLE drivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    vehicle_info JSONB,
    rating DECIMAL(3,2),
    status VARCHAR(50) DEFAULT 'OFFLINE',
    is_online BOOLEAN DEFAULT false
);

-- Rides
CREATE TABLE rides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_number VARCHAR(50) UNIQUE,
    passenger_id UUID REFERENCES users(id),
    driver_id UUID REFERENCES drivers(id),
    status VARCHAR(50) NOT NULL,
    vehicle_type VARCHAR(50),
    pickup_location GEOGRAPHY(POINT, 4326),
    destination_location GEOGRAPHY(POINT, 4326),
    pickup_address TEXT,
    destination_address TEXT,
    estimated_fare DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Driver Coordinates (PostGIS)
CREATE TABLE driver_coordinates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID REFERENCES drivers(id),
    location GEOGRAPHY(POINT, 4326),
    accuracy_meters DECIMAL(10,2),
    speed_kmh DECIMAL(10,2),
    heading_degrees DECIMAL(5,2),
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Spatial Index
CREATE INDEX idx_driver_coordinates_location 
ON driver_coordinates USING GIST (location);
```

### PostGIS Query Examples

#### 1. Find drivers within 5km radius

```sql
SELECT 
    d.id,
    d.user_id,
    dc.location,
    ST_Distance(
        dc.location,
        ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography
    ) as distance_meters
FROM drivers d
INNER JOIN LATERAL (
    SELECT location
    FROM driver_coordinates
    WHERE driver_id = d.id
    ORDER BY recorded_at DESC
    LIMIT 1
) dc ON true
WHERE d.is_online = true
  AND d.status = 'AVAILABLE'
  AND d.vehicle_info->>'type' = 'ECONOMY'
  AND ST_DWithin(
    dc.location,
    ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography,
    5000  -- 5 км в метрах
  )
ORDER BY distance_meters ASC
LIMIT 10;
```

**What happens:**
- `ST_DWithin` - fast radius check (uses spatial index)
- `ST_Distance` - precise distance calculation for sorting
- `LATERAL JOIN` - getting the latest coordinate for each driver
- `GEOGRAPHY` - automatic Earth curvature accounting

#### 2. Ride history with distance

```sql
SELECT 
    r.ride_number,
    r.pickup_address,
    r.destination_address,
    r.estimated_fare,
    ST_Distance(
        r.pickup_location,
        r.destination_location
    ) / 1000.0 as distance_km,
    r.created_at
FROM rides r
WHERE r.passenger_id = 'passenger-123'
ORDER BY r.created_at DESC
LIMIT 10;
```

#### 3. Active drivers on map (GeoJSON)

```sql
SELECT jsonb_build_object(
    'type', 'FeatureCollection',
    'features', jsonb_agg(
        jsonb_build_object(
            'type', 'Feature',
            'geometry', ST_AsGeoJSON(dc.location)::jsonb,
            'properties', jsonb_build_object(
                'driver_id', d.id,
                'status', d.status,
                'vehicle_type', d.vehicle_info->>'type',
                'rating', d.rating
            )
        )
    )
) as geojson
FROM drivers d
INNER JOIN LATERAL (
    SELECT location
    FROM driver_coordinates
    WHERE driver_id = d.id
    ORDER BY recorded_at DESC
    LIMIT 1
) dc ON true
WHERE d.is_online = true;
```

### Database Maintenance

```bash
# Connect to PostgreSQL
docker exec -it ride-postgres psql -U postgres -d ridehail

# Check extensions
SELECT * FROM pg_extension WHERE extname IN ('uuid-ossp', 'postgis');

# Table statistics
SELECT 
    schemaname,
    tablename,
    n_live_tup as rows,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_stat_user_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

# Check spatial index
SELECT 
    indexname, 
    indexdef 
FROM pg_indexes 
WHERE tablename = 'driver_coordinates';

# Analyze GIST index performance
EXPLAIN ANALYZE
SELECT *
FROM driver_coordinates
WHERE ST_DWithin(
    location,
    ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography,
    5000
);
```

---

## 🧪 Testing

### 1. Infrastructure Tests

```bash
# PostgreSQL
docker exec ride-postgres pg_isready -U postgres

# PostgreSQL + PostGIS
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT PostGIS_Version();"

# RabbitMQ
curl -u guest:guest http://localhost:15672/api/overview

# Exchanges и queues
curl -u guest:guest http://localhost:15672/api/exchanges | jq
curl -u guest:guest http://localhost:15672/api/queues | jq
```

### 2. Health Checks

```bash
# Все сервисы
curl http://localhost:3000/health  # Ride Service
curl http://localhost:3001/health  # Driver Service
curl http://localhost:3004/health  # Admin Service
```

### 3. Unit Tests

```bash
# Run all tests
go test ./... -v

# Тесты с покрытием
go test ./... -cover -coverprofile=coverage.out

# HTML отчет
go tool cover -html=coverage.out -o coverage.html

# Тесты конкретного модуля
go test ./internal/ride/... -v
go test ./internal/driver/... -v
go test ./internal/admin/... -v
```

### 4. Integration Tests

```bash
# Полный E2E тест
./scripts/test-e2e-ride-flow.sh

# Тест с подробным выводом
bash -x ./scripts/test-e2e-ride-flow.sh

# Тест driver API
./scripts/test-driver-flow.sh

# Тест admin API
./scripts/test-admin-api.sh
```

### 5. Manual Testing Workflow

#### Step 1: Create Users

```bash
# Generate admin token
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
  --user-id "admin-1" \
  --role "ADMIN" \
  --ttl "24h" | grep "JWT:" | cut -d' ' -f2)

# Create passenger
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "passenger-test-123",
    "email": "passenger@test.com",
    "role": "PASSENGER",
    "phone": "+79991234567"
  }'

# Create driver
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "driver-test-456",
    "email": "driver@test.com",
    "role": "DRIVER",
    "phone": "+79997654321"
  }'
```

#### Step 2: Driver Goes Online

```bash
# Generate driver token
DRIVER_TOKEN=$(go run cmd/generate-jwt/main.go \
  --user-id "driver-test-456" \
  --role "DRIVER" \
  --ttl "24h" | grep "JWT:" | cut -d' ' -f2)

# Go online
curl -X POST http://localhost:3001/drivers/driver-test-456/online \
  -H "Authorization: Bearer $DRIVER_TOKEN"

# Update location (Moscow, center)
curl -X POST http://localhost:3001/drivers/driver-test-456/location \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 55.7558,
    "longitude": 37.6173,
    "accuracy_meters": 10.0,
    "speed_kmh": 0.0,
    "heading_degrees": 0
  }'
```

#### Step 3: Passenger Creates Ride

```bash
# Generate passenger token
PASSENGER_TOKEN=$(go run cmd/generate-jwt/main.go \
  --user-id "passenger-test-123" \
  --role "PASSENGER" \
  --ttl "24h" | grep "JWT:" | cut -d' ' -f2)

# Create ride
RIDE_RESPONSE=$(curl -X POST http://localhost:3000/rides \
  -H "Authorization: Bearer $PASSENGER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 55.7558,
    "pickup_lng": 37.6173,
    "pickup_address": "Red Square, Moscow",
    "destination_lat": 55.7522,
    "destination_lng": 37.6156,
    "destination_address": "Kremlin, Moscow",
    "priority": 5
  }')

echo $RIDE_RESPONSE | jq
RIDE_ID=$(echo $RIDE_RESPONSE | jq -r '.ride_id')
```

#### Step 4: Check RabbitMQ

```bash
# Check messages in driver_matching queue
curl -u guest:guest \
  "http://localhost:15672/api/queues/%2F/driver_matching" | jq

# Get message (non-destructive peek)
curl -u guest:guest \
  -X POST "http://localhost:15672/api/queues/%2F/driver_matching/get" \
  -H "Content-Type: application/json" \
  -d '{"count":1,"ackmode":"ack_requeue_true","encoding":"auto"}' | jq
```

#### Step 5: WebSocket Testing

```bash
# Install websocat (if not installed)
# cargo install websocat

# Connect as driver
websocat "ws://localhost:3001/ws?token=$DRIVER_TOKEN"

# In another terminal - connect as passenger
websocat "ws://localhost:3000/ws?token=$PASSENGER_TOKEN"

# Create ride and observe events in both WebSockets
```

### 6. Performance Testing

```bash
# Установить Apache Bench
sudo apt-get install apache2-utils

# Тест health endpoint
ab -n 1000 -c 10 http://localhost:3000/health

# Тест создания поездок (с JWT)
ab -n 100 -c 5 \
  -H "Authorization: Bearer $PASSENGER_TOKEN" \
  -p ride-payload.json \
  -T application/json \
  http://localhost:3000/rides
```

### 7. Load Testing с k6

```javascript
// load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
};

export default function () {
  let token = 'YOUR_JWT_TOKEN';
  
  let payload = JSON.stringify({
    vehicle_type: 'ECONOMY',
    pickup_lat: 55.7558,
    pickup_lng: 37.6173,
    pickup_address: 'Red Square',
    destination_lat: 55.7522,
    destination_lng: 37.6156,
    destination_address: 'Kremlin',
  });

  let params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  };

  let res = http.post('http://localhost:3000/rides', payload, params);
  
  check(res, {
    'status is 201': (r) => r.status === 201,
    'ride_id present': (r) => JSON.parse(r.body).ride_id !== undefined,
  });

  sleep(1);
}
```

```bash
# Run k6
k6 run load-test.js
```

---

## 📊 Monitoring

### Metrics Endpoints

```bash
# Health checks
curl http://localhost:3000/health | jq
curl http://localhost:3001/health | jq
curl http://localhost:3004/health | jq

# RabbitMQ metrics
curl -u guest:guest http://localhost:15672/api/overview | jq '.queue_totals'

# Database stats
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT COUNT(*) FROM rides WHERE status = 'PENDING';"

docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT COUNT(*) FROM drivers WHERE is_online = true;"
```

### Logs

```bash
# Docker compose logs
docker-compose -f deployments/docker-compose.yml logs -f

# Specific service logs
docker-compose -f deployments/docker-compose.yml logs -f ride-postgres
docker-compose -f deployments/docker-compose.yml logs -f ride-rabbitmq

# Application logs (если запущено через go run)
# Логи пишутся в stdout
```

---

## 🚀 Deployment

### Production Build

```bash
# Сборка бинарника
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o bin/ridehail-linux-amd64 \
  ./main.go

# Binary size
ls -lh bin/ridehail-linux-amd64

# Upx compression (optional)
upx --best --lzma bin/ridehail-linux-amd64
```

### Docker Build

```bash
# Build image
docker build -f deployments/Dockerfile -t ridehail:latest .

# Check size
docker images ridehail:latest

# Run container
docker run -d \
  --name ridehail-app \
  -p 3000:3000 \
  -p 3001:3001 \
  -p 3002:3002 \
  -e DB_HOST=postgres \
  -e RABBITMQ_HOST=rabbitmq \
  ridehail:latest
```

### Environment Variables

```bash
# Database
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=ridehail

# RabbitMQ
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT=5672
export RABBITMQ_USER=guest
export RABBITMQ_PASSWORD=guest

# JWT
export JWT_SECRET=your-super-secret-key-change-in-production
export JWT_ISSUER=ride-hail-system

# Service Ports
export RIDE_SERVICE_PORT=3000
export DRIVER_SERVICE_PORT=3001
export ADMIN_SERVICE_PORT=3002

# Logging
export LOG_LEVEL=info  # debug, info, warn, error
```

---

## 📝 Documentation

```bash
# Подключиться к PostgreSQL
make db-shell

# Или вручную
docker exec -it ridehail-postgres psql -U ridehail_user -d ridehail_db

# Проверить созданные поездки
SELECT * FROM rides ORDER BY created_at DESC LIMIT 5;
```

## 🐰 RabbitMQ

```bash
# Open Management UI
# http://localhost:15672
# Login: guest / guest

# Check queues
# Exchanges: ride_topic, driver_topic, location_fanout
# Queues: ride.requested, ride.matched, ride.completed, etc.
```

## 🛠️ Useful Commands

```bash
# Show all available commands
make help

# Run tests
make test

# Linter
make lint

# Clean artifacts
make clean

# Rebuild Docker images
make docker-build

# Restart services
make docker-restart
```

## 📝 Project Structure

```
ride-hail/
├── cmd/                      # CLI утилиты
│   └── generate-jwt/         # Генератор JWT токенов
├── config/                   # Конфигурационные файлы
│   ├── db.yaml
│   ├── mq.yaml
│   ├── service.yaml
│   ├── ws.yaml
│   └── jwt.yaml
├── deployments/              # Docker файлы
│   ├── Dockerfile
│   └── docker-compose.yml
├── internal/
│   ├── ride/                 # Ride Service
│   │   ├── domain/
│   │   ├── application/
│   │   ├── adapter/
│   │   └── bootstrap/
│   ├── driver/               # Driver Service
│   ├── admin/                # Admin Service
│   └── shared/               # Общая инфраструктура
│       ├── config/
│       ├── logger/
│       ├── db/
│       ├── mq/
│       ├── ws/
│       └── auth/
├── main.go                   # Точка входа
├── Makefile
└── README.md
```

## 🧪 Testing

### Unit Tests

```bash
make test
```

### Driver Service Testing ⭐

Full documentation: [TESTING_GUIDE.md](TESTING_GUIDE.md)

```bash
# 1. Create test driver
./scripts/setup-test-driver.sh

# 2. Run full testing (8 tests)
export DRIVER_ID="your-driver-id"
./scripts/test-driver-api.sh
```

Available scripts:
- `setup-test-driver.sh` - create test driver
- `generate-driver-token.sh` - generate JWT token
- `test-driver-api.sh` - automatic API testing (8 tests)
- `test-driver-workflow.sh` - complete driver workflow
- `driver-api-helpers.sh` - interactive functions

### Integration Tests

```bash
# Start services
make docker-up

# Run tests
./scripts/integration-test.sh
```

## 📊 Monitoring

### Metrics Endpoints

```bash
# Health checks
curl http://localhost:3000/health | jq
curl http://localhost:3001/health | jq
curl http://localhost:3004/health | jq

# RabbitMQ metrics
curl -u guest:guest http://localhost:15672/api/overview | jq '.queue_totals'

# Database stats
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT COUNT(*) FROM rides WHERE status = 'PENDING';"

docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT COUNT(*) FROM drivers WHERE is_online = true;"
```

### Logs

```bash
# Docker compose logs
docker-compose -f deployments/docker-compose.yml logs -f

# Specific service logs
docker-compose -f deployments/docker-compose.yml logs -f ride-postgres
docker-compose -f deployments/docker-compose.yml logs -f ride-rabbitmq

# Application logs (если запущено через go run)
# Логи пишутся в stdout с JSON structured logging
```

---

## 🚀 Deployment

### Production Build

```bash
# Сборка бинарника
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o bin/ridehail-linux-amd64 \
  ./main.go

# Binary size
ls -lh bin/ridehail-linux-amd64

# Upx compression (optional)
upx --best --lzma bin/ridehail-linux-amd64
```

### Docker Build

```bash
# Build image
docker build -f deployments/Dockerfile -t ridehail:latest .

# Check size
docker images ridehail:latest

# Run container
docker run -d \
  --name ridehail-app \
  -p 3000:3000 \
  -p 3001:3001 \
  -p 3002:3002 \
  -e DB_HOST=postgres \
  -e RABBITMQ_HOST=rabbitmq \
  ridehail:latest
```

### Environment Variables

```bash
# Database
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=ridehail

# RabbitMQ
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT=5672
export RABBITMQ_USER=guest
export RABBITMQ_PASSWORD=guest

# JWT
export JWT_SECRET=your-super-secret-key-change-in-production
export JWT_ISSUER=ride-hail-system

# Service Ports
export RIDE_SERVICE_PORT=3000
export DRIVER_SERVICE_PORT=3001
export ADMIN_SERVICE_PORT=3002

# Logging
export LOG_LEVEL=info  # debug, info, warn, error
```

---

## 📝 Documentation

### Architecture Documents

- **[IMPLEMENTATION_GUIDE.md](docs/IMPLEMENTATION_GUIDE.md)** - Полное руководство по архитектуре
  - Детальное описание каждого компонента
  - Диаграммы потоков данных
  - PostGIS query examples
  - RabbitMQ топология с примерами
  - WebSocket протоколы

- **[PROJECT_COMPLETION.md](PROJECT_COMPLETION.md)** - Отчет о завершении проекта
  - Чеклисты всех компонентов
  - Технические метрики
  - Результаты тестирования
  - Список достижений

- **[FINAL_SUMMARY.md](FINAL_SUMMARY.md)** - Краткий обзор проекта
  - Quick start guide
  - Ключевые метрики
  - Основные достижения
  - Навигация по документации

### API Documentation

- **[docs/admin_api.md](docs/admin_api.md)** - Admin Service API reference
  - Все endpoints с примерами
  - Модели данных
  - Примеры curl запросов

- **[docs/INTEGRATION.md](docs/INTEGRATION.md)** - Руководство по интеграции
  - WebSocket протоколы
  - RabbitMQ message formats
  - JWT authentication flow

### Code Examples

```bash
# Примеры использования в директории scripts/
./scripts/test-e2e-ride-flow.sh      # E2E тест полного flow
./scripts/test-admin-api.sh           # Testing Admin API
./scripts/generate-admin-token.sh     # Генерация admin токена
```

---

## 🏗️ Architecture Patterns

### Clean Architecture (Hexagonal)

```
internal/
├─ ride/
│  ├─ domain/           # Бизнес-логика, entities
│  ├─ application/      # Use cases, ports
│  │  ├─ ports/
│  │  │  ├─ in/        # Входящие порты (интерфейсы use cases)
│  │  │  └─ out/       # Исходящие порты (репозитории)
│  │  └─ usecase/      # Реализация use cases
│  ├─ adapter/         # Адаптеры
│  │  ├─ in/           # Входящие адаптеры
│  │  │  ├─ transport/ # HTTP handlers
│  │  │  ├─ in_ws/     # WebSocket handlers
│  │  │  └─ in_amqp/   # RabbitMQ consumers
│  │  └─ out/          # Исходящие адаптеры (DB, MQ producers)
│  └─ bootstrap/       # Dependency injection
```

**Principles:**
- ✅ **Dependency Inversion** - domain doesn't depend on external libraries
- ✅ **Ports & Adapters** - clear boundaries between layers
- ✅ **Use Cases** - business logic is isolated
- ✅ **Testability** - easy to mock dependencies

### Event-Driven Architecture

**Asynchronous communication via RabbitMQ:**

1. **Topic Exchange** - маршрутизация по routing key
   - `ride_topic`: `ride.request.*`
   - `driver_topic`: `driver.response.*`

2. **Fanout Exchange** - broadcast всем подписчикам
   - `location_fanout`: обновления локации водителя

3. **Dead Letter Queues** - обработка ошибок
   - Retry механизм с экспоненциальной задержкой
   - Monitoring failed messages

**Advantages:**
- 🔄 **Loose Coupling** - services are independent
- 📈 **Scalability** - horizontal scaling
- 🛡️ **Resilience** - fault tolerance through queues
- 📊 **Auditability** - all events are logged

### Geospatial Architecture (PostGIS)

**Query optimization:**

```sql
-- 1. Spatial Index (GIST)
CREATE INDEX idx_driver_coordinates_location 
ON driver_coordinates USING GIST (location);

-- 2. Two-step query optimization
-- Step 1: ST_DWithin (fast filtering by index)
-- Step 2: ST_Distance (precise distance for top-N)

-- 3. LATERAL JOIN for latest location
SELECT d.*, dc.location
FROM drivers d
INNER JOIN LATERAL (
    SELECT location
    FROM driver_coordinates
    WHERE driver_id = d.id
    ORDER BY recorded_at DESC
    LIMIT 1
) dc ON true;
```

**Performance:**
- ⚡ GIST index: O(log n) search vs O(n) table scan
- 🎯 ST_DWithin использует bounding box для быстрой фильтрации
- 📍 GEOGRAPHY type автоматически учитывает кривизну Земли

---

## 🔧 Troubleshooting

### Common Issues

#### 1. RabbitMQ Connection Failed

**Symptoms:**
```
Failed to connect to RabbitMQ: dial tcp: connection refused
```

**Solution:**
```bash
# Check status
docker-compose -f deployments/docker-compose.yml ps

# Перезапустить RabbitMQ
docker-compose -f deployments/docker-compose.yml restart ride-rabbitmq

# Check logs
docker-compose -f deployments/docker-compose.yml logs ride-rabbitmq

# Check ports
netstat -tlnp | grep 5672
```

#### 2. PostgreSQL Connection Failed

**Symptoms:**
```
Error connecting to database: connection refused
```

**Solution:**
```bash
# Check status
docker exec ride-postgres pg_isready -U postgres

# Check connection
docker exec -it ride-postgres psql -U postgres -d ridehail -c "\conninfo"

# Recreate DB (CAUTION!)
docker-compose -f deployments/docker-compose.yml down -v
docker-compose -f deployments/docker-compose.yml up -d
```

#### 3. PostGIS Extension Missing

**Symptoms:**
```
ERROR: type "geography" does not exist
```

**Solution:**
```bash
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "CREATE EXTENSION IF NOT EXISTS postgis;"

# Verify
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT PostGIS_Version();"
```

#### 4. JWT Token Invalid

**Symptoms:**
```json
{"error": "unauthorized", "message": "invalid token"}
```

**Solution:**
```bash
# Check secret in config/jwt.yaml
cat config/jwt.yaml

# Generate new token
go run cmd/generate-jwt/main.go \
  --user-id "test-123" \
  --role "PASSENGER" \
  --ttl "24h"

# Verify token
go run cmd/verify-jwt/main.go --token "YOUR_TOKEN"
```

#### 5. WebSocket Connection Failed

**Symptoms:**
```
WebSocket handshake failed: 401 Unauthorized
```

**Solution:**
```bash
# Check token in URL
ws://localhost:3000/ws?token=YOUR_JWT_TOKEN

# Check role (PASSENGER for /rides, DRIVER for /drivers)

# Test connection with curl
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
  "http://localhost:3000/ws?token=$TOKEN"
```

#### 6. Driver Matching Not Working

**Symptoms:**
- Ride is created, but driver doesn't receive notification

**Diagnostics:**
```bash
# 1. Check that driver is online
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT id, is_online, status FROM drivers;"

# 2. Check driver location
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT driver_id, ST_AsText(location), recorded_at 
      FROM driver_coordinates 
      ORDER BY recorded_at DESC 
      LIMIT 5;"

# 3. Check driver_matching queue
curl -u guest:guest \
  http://localhost:15672/api/queues/%2F/driver_matching | jq

# 4. Check Driver Service logs
docker-compose -f deployments/docker-compose.yml logs driver-service

# 5. Test PostGIS query manually
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT d.id, 
      ST_Distance(
        dc.location,
        ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography
      ) as distance_meters
      FROM drivers d
      INNER JOIN LATERAL (
        SELECT location FROM driver_coordinates 
        WHERE driver_id = d.id 
        ORDER BY recorded_at DESC LIMIT 1
      ) dc ON true
      WHERE d.is_online = true
        AND ST_DWithin(
          dc.location,
          ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography,
          5000
        )
      ORDER BY distance_meters ASC;"
```

#### 7. High Memory Usage

**Solution:**
```bash
# Check memory usage
docker stats

# Limit memory for containers
# In docker-compose.yml add:
services:
  ride-postgres:
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M

# Clean unused images
docker system prune -a
```

#### 8. Docker Buildx Error

If you get error `fork/exec .../docker-buildx: no such file or directory`:

```bash
# Use regular docker build instead of buildx
docker build -f deployments/Dockerfile -t ride-hail .
```

#### 9. Ports Busy

```bash
# Check occupied ports
sudo lsof -i :3000
sudo lsof -i :5432

# Kill process on port
sudo kill -9 $(sudo lsof -t -i:3000)

# Change ports in docker-compose.yml
```

#### 10. Migration Issues

```bash
# Recreate DB (deletes all data!)
docker-compose -f deployments/docker-compose.yml down -v
docker-compose -f deployments/docker-compose.yml up -d

# Or manually
docker exec -it ride-postgres psql -U postgres -c "DROP DATABASE IF EXISTS ridehail;"
docker exec -it ride-postgres psql -U postgres -c "CREATE DATABASE ridehail;"
```

### Debug Mode

```bash
# Run with debug logs
export LOG_LEVEL=debug
go run main.go

# Trace SQL queries (PostgreSQL)
export DB_LOG_LEVEL=debug

# Trace RabbitMQ messages
export RABBITMQ_LOG_LEVEL=debug
```

---

## 🤝 Contributing

### Development Workflow

1. **Fork the repository**
2. **Create feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Make changes**
   - Follow Clean Architecture
   - Add unit tests
   - Update documentation
4. **Run tests**
   ```bash
   go test ./... -v
   ./scripts/test-e2e-ride-flow.sh
   ```
5. **Commit changes**
   ```bash
   git commit -m "feat: add amazing feature"
   ```
6. **Push to branch**
   ```bash
   git push origin feature/amazing-feature
   ```
7. **Open Pull Request**

### Code Style

```bash
# Formatting
go fmt ./...

# Linting
golangci-lint run

# Imports organization
goimports -w .
```

### Commit Convention

```
feat: новая функция
fix: исправление бага
docs: изменения в документации
refactor: рефакторинг кода
test: добавление тестов
chore: обновление зависимостей, конфигурации
```

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

---

## 👥 Authors

- **Adam** - Initial work and architecture

---

## 🙏 Acknowledgments

- **Go Community** - amazing ecosystem
- **PostGIS Team** - powerful geospatial extension
- **RabbitMQ Team** - reliable messaging
- **Clean Architecture** - Robert C. Martin
- **Hexagonal Architecture** - Alistair Cockburn

---

## 📞 Support

### Issues

If you found a bug or want to suggest an improvement:
1. Check [Troubleshooting](#-troubleshooting)
2. Open an issue on GitHub
3. Describe the problem with examples

### Questions

For questions about the project:
- Create a discussion on GitHub
- Specify version of Go, PostgreSQL, RabbitMQ
- Attach logs and configuration

---

## 🎯 Roadmap

### Version 2.0 (Planned)

- [ ] **Payment Integration**
  - Stripe/PayPal integration
  - Fare calculation algorithm
  - Transaction history

- [ ] **Advanced Matching**
  - Machine learning for ETA prediction
  - Traffic data integration
  - Dynamic pricing

- [ ] **Mobile Apps**
  - React Native passenger app
  - React Native driver app
  - Push notifications

- [ ] **Analytics Dashboard**
  - Real-time metrics (Grafana)
  - Business intelligence
  - Driver performance tracking

- [ ] **Advanced Features**
  - Ride scheduling
  - Ride sharing (pooling)
  - Multi-stop rides
  - Favorite locations

- [ ] **Operations**
  - Kubernetes deployment
  - CI/CD pipeline (GitHub Actions)
  - Automated testing
  - Load testing framework

---

**⭐ If you find this project useful, please star it on GitHub!**