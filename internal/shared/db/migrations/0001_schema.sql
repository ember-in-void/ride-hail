-- Idempotent base schema. No BEGIN/COMMIT inside this file.

-- User roles enumeration
create table if not exists roles(value text not null primary key);
insert into roles(value) values ('PASSENGER'),('DRIVER'),('ADMIN') on conflict do nothing;

-- User status enumeration
create table if not exists user_status(value text not null primary key);
insert into user_status(value) values ('ACTIVE'),('INACTIVE'),('BANNED') on conflict do nothing;

-- Main users table
create table if not exists users (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    email varchar(100) unique not null,
    role text references roles(value) not null,
    status text references user_status(value) not null default 'ACTIVE',
    password_hash text not null,
    attrs jsonb default '{}'::jsonb
);

-- Ride status enumeration
create table if not exists ride_status(value text not null primary key);
insert into ride_status(value) values
('REQUESTED'),('MATCHED'),('EN_ROUTE'),('ARRIVED'),('IN_PROGRESS'),('COMPLETED'),('CANCELLED')
on conflict do nothing;

-- Ride type enumeration
create table if not exists vehicle_type(value text not null primary key);
insert into vehicle_type(value) values ('ECONOMY'),('PREMIUM'),('XL') on conflict do nothing;

-- Coordinates table for real-time location tracking
create table if not exists coordinates (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    entity_id uuid not null, -- driver_id or passenger_id
    entity_type varchar(20) not null check (entity_type in ('driver', 'passenger')),
    address text not null,
    latitude decimal(10,8) not null check (latitude between -90 and 90),
    longitude decimal(11,8) not null check (longitude between -180 and 180),
    fare_amount decimal(10,2) check (fare_amount >= 0),
    distance_km decimal(8,2) check (distance_km >= 0),
    duration_minutes integer check (duration_minutes >= 0),
    is_current boolean default true
);
create index if not exists idx_coordinates_entity on coordinates(entity_id, entity_type);
create index if not exists idx_coordinates_current on coordinates(entity_id, entity_type) where is_current = true;

-- Main rides table
create table if not exists rides (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    ride_number varchar(50) unique not null,
    passenger_id uuid not null references users(id),
    driver_id uuid references users(id),
    vehicle_type text references vehicle_type(value),
    status text references ride_status(value),
    priority integer default 1 check (priority between 1 and 10),
    requested_at timestamptz default now(),
    matched_at timestamptz,
    arrived_at timestamptz,
    started_at timestamptz,
    completed_at timestamptz,
    cancelled_at timestamptz,
    cancellation_reason text,
    estimated_fare decimal(10,2),
    final_fare decimal(10,2),
    pickup_coordinate_id uuid references coordinates(id),
    destination_coordinate_id uuid references coordinates(id)
);
create index if not exists idx_rides_status on rides(status);

-- Event type enumeration for audit trail
create table if not exists ride_event_type(value text not null primary key);
insert into ride_event_type(value) values
('RIDE_REQUESTED'),('DRIVER_MATCHED'),('DRIVER_ARRIVED'),
('RIDE_STARTED'),('RIDE_COMPLETED'),('RIDE_CANCELLED'),
('STATUS_CHANGED'),('LOCATION_UPDATED'),('FARE_ADJUSTED')
on conflict do nothing;

-- Event sourcing table for complete ride audit trail
create table if not exists ride_events (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    ride_id uuid references rides(id) not null,
    event_type text references ride_event_type(value),
    event_data jsonb not null
);

-- Driver domain (из ТЗ Driver & Location Service)
create table if not exists driver_status(value text not null primary key);
insert into driver_status(value) values ('OFFLINE'),('AVAILABLE'),('BUSY'),('EN_ROUTE') on conflict do nothing;

create table if not exists drivers (
    id uuid primary key references users(id),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    license_number varchar(50) unique not null,
    vehicle_type text references vehicle_type(value),
    vehicle_attrs jsonb,
    rating decimal(3,2) default 5.0 check (rating between 1.0 and 5.0),
    total_rides integer default 0 check (total_rides >= 0),
    total_earnings decimal(10,2) default 0 check (total_earnings >= 0),
    status text references driver_status(value),
    is_verified boolean default false
);
create index if not exists idx_drivers_status on drivers(status);

create table if not exists driver_sessions (
    id uuid primary key default gen_random_uuid(),
    driver_id uuid references drivers(id) not null,
    started_at timestamptz not null default now(),
    ended_at timestamptz,
    total_rides integer default 0,
    total_earnings decimal(10,2) default 0
);

create table if not exists location_history (
    id uuid primary key default gen_random_uuid(),
    coordinate_id uuid references coordinates(id),
    driver_id uuid references drivers(id),
    latitude decimal(10,8) not null check (latitude between -90 and 90),
    longitude decimal(11,8) not null check (longitude between -180 and 180),
    accuracy_meters decimal(6,2),
    speed_kmh decimal(5,2),
    heading_degrees decimal(5,2) check (heading_degrees between 0 and 360),
    recorded_at timestamptz not null default now(),
    ride_id uuid references rides(id)
);

-- Outbox для транзакционной доставки сообщений
create table if not exists outbox (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    event_type text not null,
    routing_key text not null,
    payload jsonb not null,
    status text not null default 'PENDING', -- PENDING|SENT|FAILED
    retry_count int not null default 0
);
create index if not exists idx_outbox_status on outbox(status);