# Delcom Blog — Backend (Golang Gin REST API)

Backend REST API untuk platform **Delcom Blog Multi-Tier & AI Generator dengan Integrasi Search Engine** menggunakan arsitektur **Clean Architecture & Feature-First** (Golang 1.27.4, Gin, GORM, PostgreSQL 17, Swagger/OpenAPI).

## Cara Menjalankan

```bash
cp .env.example .env
go run cmd/server/main.go
```

Server berjalan pada `http://localhost:8081`.
Dokumentasi interaktif Swagger UI tersedia pada `http://localhost:8081/swagger/index.html`.

## Pengujian Unit (100% Statement Coverage)

```bash
go test -v -cover ./internal/... ./docs/...
```

## Build Binary

```bash
go build -o server.exe cmd/server/main.go
.\server.exe
```
