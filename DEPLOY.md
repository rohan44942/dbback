# DBBack Deployment Guide

Split deployment: **Vercel (frontend)** + **remote Go API** + **PostgreSQL** + **S3**.

## Architecture

```
Browser -> Vercel (Next.js) -> HTTPS -> Go Controller API
                                      -> PostgreSQL (metadata + users)
                                      -> S3 (backup files)
```

## 1. PostgreSQL

Provision a managed Postgres instance (Railway, **Neon**, Supabase, RDS, or Fly Postgres).

### Neon (recommended for split deploy)

Project example: `frosty-tree-01729330` (org `org-icy-shape-84494297`).

1. Get connection string:
   ```sh
   npx neonctl connection-string frosty-tree-01729330
   ```
   Or use the Neon MCP / Console → Connection details → **Pooled connection**.

2. Save to `dbback/.env` (gitignored):
   ```
   DBBACK_DATABASE_URL=postgresql://...@...-pooler...neon.tech/neondb?sslmode=require
   ```

3. The controller loads `.env` automatically and runs migrations on startup.

Connection string example:

```
postgres://user:password@host/neondb?sslmode=require
```

The controller runs migrations automatically on startup.

## 2. Backend API

### Required environment variables

| Variable | Description |
|----------|-------------|
| `DBBACK_DATABASE_URL` | PostgreSQL connection string |
| `DBBACK_AUTH_SECRET` | Long random string for JWT signing |
| `DBBACK_CORS_ORIGIN` | Frontend URL(s), comma-separated |
| `DBBACK_ADDR` | Listen address (default `:8080`) |
| `DBBACK_STORAGE_TYPE` | `s3` or `local` |
| `DBBACK_STORAGE_BUCKET` | S3 bucket name |
| `DBBACK_STORAGE_ACCESS_KEY` | S3 access key |
| `DBBACK_STORAGE_SECRET_KEY` | S3 secret key |
| `DBBACK_STORAGE_ENDPOINT` | S3 endpoint (optional for AWS) |
| `DBBACK_ENV` | Set to `production` to enforce auth secret |
| `SLACK_WEBHOOK_URL` | Optional failure notifications |

### Railway / Fly.io

1. Deploy from `dbback/` using the Dockerfile.
2. Attach Postgres plugin or set `DBBACK_DATABASE_URL`.
3. Set env vars above.
4. Expose port 8080 with HTTPS.

### Local dev with Docker Compose

```sh
cd dbback
docker-compose -f docker-compose.dev.yml up --build
```

API: http://localhost:8080  
Health: http://localhost:8080/health

### Local dev without Docker

```powershell
# Start Postgres locally, then:
cd dbback
$env:DBBACK_DATABASE_URL = "postgres://dbback:dbback@localhost:5432/dbback?sslmode=disable"
$env:DBBACK_STORAGE_TYPE = "local"
$env:DBBACK_CORS_ORIGIN = "http://localhost:3000"
go run ./cmd/controller
```

## 3. Frontend (Netlify)

1. Set project root directory to `dbback-frontend` (or deploy the frontend-only repo).
2. Add environment variables:
   ```
   NEXT_PUBLIC_API_URL=https://your-api-domain.com/api
   NEXT_PUBLIC_GOOGLE_CLIENT_ID=YOUR_GOOGLE_OAUTH_WEB_CLIENT_ID.apps.googleusercontent.com
   ```
3. Deploy.

4. Set backend CORS + Google client ID:
   ```
   DBBACK_CORS_ORIGIN=https://dbbackup.netlify.app
   GOOGLE_CLIENT_ID=YOUR_GOOGLE_OAUTH_WEB_CLIENT_ID.apps.googleusercontent.com
   ```
   Use the **same** Web client ID on frontend and backend.

### Google Cloud Console setup

1. Create an OAuth client (type: **Web application**).
2. Authorized JavaScript origins:
   - `http://localhost:3000`
   - `https://dbbackup.netlify.app`
3. Authorized redirect URIs (GIS button flow usually only needs origins):
   - `http://localhost:3000`
   - `https://dbbackup.netlify.app`
4. Copy the Client ID into Netlify + Render env vars above.

## 4. Deploy order

1. Provision Postgres
2. Deploy backend API with `DBBACK_DATABASE_URL` and secrets
3. Verify `GET /health` returns `{"status":"ok","db":"ok"}`
4. Deploy frontend on Vercel with `NEXT_PUBLIC_API_URL`
5. Register a user via `/register`, then log in

## 5. Health check

```
GET /health
```

Response:

```json
{"status":"ok","db":"ok"}
```

Use this for load balancer and platform health probes.
