# Image AI Backend

Go + PostgreSQL backend for the multi-user, multi-app image generation site.

## Database

```sql
CREATE DATABASE "image-ai";
```

Default local connection:

```txt
postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable
```

Run migrations:

```powershell
cd backend
$env:DATABASE_URL="postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable"
go run ./cmd/migrate
```

Start API:

```powershell
cd backend
$env:DATABASE_URL="postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable"
$env:JWT_SECRET="local-dev-secret"
go run ./cmd/api
```

## Main Endpoints

- `GET /healthz`
- `POST /api/v1/auth/send-code`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/forgot-password`
- `GET /api/v1/auth/me`
- `GET /api/v1/apps`
- `GET /api/v1/apps/{id}`
- `POST /api/v1/generations`
- `GET /api/v1/generations`
- `GET /api/v1/generations/{id}`

Generation is queued and completed with placeholder output in v1. Replace the worker section with a real image provider later.
