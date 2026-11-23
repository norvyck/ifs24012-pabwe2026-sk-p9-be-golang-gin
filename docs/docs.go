package docs

import (
	"encoding/json"
	"strings"

	"github.com/swaggo/swag"
)

const docTemplate = `{
  "swagger": "2.0",
  "info": {
    "description": "REST API untuk platform Delcom Blog Multi-Tier & AI Generator dengan Integrasi Search Engine.",
    "title": "Delcom Blog REST API",
    "version": "1.0.0"
  },
  "host": "{{HOST}}",
  "basePath": "/",
  "schemes": ["{{SCHEME}}"],
  "securityDefinitions": {
    "BearerAuth": {
      "type": "apiKey",
      "name": "Authorization",
      "in": "header",
      "description": "Masukkan token dengan format: Bearer <token>"
    }
  },
  "paths": {
    "/api/public/categories": {
      "get": {
        "tags": ["Public"],
        "summary": "Daftar kategori artikel blog",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/public/posts": {
      "get": {
        "tags": ["Public"],
        "summary": "Daftar artikel terbit dengan pencarian dan filter kategori",
        "parameters": [
          { "name": "q", "in": "query", "type": "string", "description": "Kata kunci pencarian" },
          { "name": "category", "in": "query", "type": "string", "description": "Filter kategori" }
        ],
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/public/posts/{slug}": {
      "get": {
        "tags": ["Public"],
        "summary": "Detail artikel publik berdasarkan slug",
        "parameters": [
          { "name": "slug", "in": "path", "required": true, "type": "string" }
        ],
        "responses": { "200": { "description": "OK" }, "404": { "description": "Not Found" } }
      }
    },
    "/api/auth/register": {
      "post": {
        "tags": ["Auth"],
        "summary": "Registrasi akun penulis baru",
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "required": ["username", "email", "password"],
              "properties": {
                "username": { "type": "string", "example": "penulis1" },
                "email": { "type": "string", "example": "penulis1@delcom.org" },
                "password": { "type": "string", "example": "password123" }
              }
            }
          }
        ],
        "responses": { "201": { "description": "Created" } }
      }
    },
    "/api/auth/login": {
      "post": {
        "tags": ["Auth"],
        "summary": "Login pengguna dan penerbitan JWT",
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "required": ["username", "password"],
              "properties": {
                "username": { "type": "string", "example": "admin" },
                "password": { "type": "string", "example": "admin" }
              }
            }
          }
        ],
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/auth/logout": {
      "post": {
        "tags": ["Auth"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Logout sesi pengguna aktif",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/auth/profile": {
      "get": {
        "tags": ["Auth"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Ambil profil penulis login",
        "responses": { "200": { "description": "OK" } }
      },
      "put": {
        "tags": ["Auth"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Perbarui profil penulis login",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/posts": {
      "get": {
        "tags": ["Posts"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Daftar artikel milik penulis login",
        "responses": { "200": { "description": "OK" } }
      },
      "post": {
        "tags": ["Posts"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Buat artikel baru",
        "responses": { "201": { "description": "Created" } }
      }
    },
    "/api/posts/stats": {
      "get": {
        "tags": ["Posts"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Ringkasan 3 kartu statistik artikel penulis",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/posts/{id}": {
      "get": {
        "tags": ["Posts"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Detail artikel penulis berdasarkan ID",
        "parameters": [{ "name": "id", "in": "path", "required": true, "type": "integer" }],
        "responses": { "200": { "description": "OK" } }
      },
      "put": {
        "tags": ["Posts"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Perbarui artikel penulis",
        "parameters": [{ "name": "id", "in": "path", "required": true, "type": "integer" }],
        "responses": { "200": { "description": "OK" } }
      },
      "delete": {
        "tags": ["Posts"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Hapus artikel penulis",
        "parameters": [{ "name": "id", "in": "path", "required": true, "type": "integer" }],
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/posts/{id}/toggle-publish": {
      "patch": {
        "tags": ["Posts"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Toggle status publikasi artikel (Draf <-> Terbit)",
        "parameters": [{ "name": "id", "in": "path", "required": true, "type": "integer" }],
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/posts/ai-generate": {
      "post": {
        "tags": ["AI"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Generate artikel blog menggunakan AI dan live Search Engine",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/admin/users": {
      "get": {
        "tags": ["Admin"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Daftar seluruh pengguna sistem dan statistik artikel",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/admin/posts": {
      "get": {
        "tags": ["Admin"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Monitoring seluruh artikel semua penulis",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/admin/users/{id}/posts": {
      "get": {
        "tags": ["Admin"],
        "security": [{ "BearerAuth": [] }],
        "summary": "Daftar artikel milik pengguna tertentu",
        "parameters": [{ "name": "id", "in": "path", "required": true, "type": "integer" }],
        "responses": { "200": { "description": "OK" } }
      }
    }
  }
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1.0.0",
	Host:             "localhost:8081",
	BasePath:         "/",
	Schemes:          []string{"http"},
	Title:            "Delcom Blog REST API",
	Description:      "REST API untuk platform Delcom Blog Multi-Tier & AI Generator dengan Integrasi Search Engine.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  GetSwaggerJSON("localhost:8081", "http"),
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}

func GetSwaggerJSON(host, scheme string) string {
	if strings.TrimSpace(host) == "" {
		host = "localhost:8081"
	}
	if strings.TrimSpace(scheme) == "" {
		scheme = "http"
	}
	spec := strings.ReplaceAll(docTemplate, "{{HOST}}", host)
	spec = strings.ReplaceAll(spec, "{{SCHEME}}", scheme)
	return spec
}

func ParseSwaggerMap(host, scheme string) map[string]any {
	var out map[string]any
	_ = json.Unmarshal([]byte(GetSwaggerJSON(host, scheme)), &out)
	return out
}
