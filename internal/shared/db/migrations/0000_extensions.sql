-- Must run before any schema relying on gen_random_uuid() and PostGIS geography/geometry functions
create extension if not exists pgcrypto;
create extension if not exists postgis;