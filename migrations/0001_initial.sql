-- Immutable initial production schema. Apply with a migration runner, never manually in production.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email TEXT NOT NULL UNIQUE, name TEXT NOT NULL, password_hash TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE TABLE projects (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), owner_id UUID NOT NULL REFERENCES users(id), name TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE TABLE api_keys (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), project_id UUID NOT NULL REFERENCES projects(id), key_prefix TEXT NOT NULL, key_hash BYTEA NOT NULL UNIQUE, name TEXT NOT NULL, revoked_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), last_used_at TIMESTAMPTZ);
CREATE TABLE verification_events (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), project_id UUID NOT NULL REFERENCES projects(id), outcome TEXT NOT NULL CHECK (outcome IN ('challenge','passed','blocked')), created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE INDEX verification_events_project_created_idx ON verification_events(project_id, created_at DESC);
CREATE TABLE sessions (id BYTEA PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id), expires_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now());
