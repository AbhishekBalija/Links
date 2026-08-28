// Package auth owns login, refresh tokens, password hashing, and auth
// middleware per docs/architecture.md § Main Domain Modules.
//
// Layer structure (per docs/architecture.md § Backend Layers):
//
//	handler.go     — HTTP binding, validation, response mapping
//	service.go     — Business rules, workflow orchestration
//	repository.go  — Database queries and persistence
//	policy.go      — Resource-specific permission checks
//	model.go       — Domain/database structs
//	dto.go         — Request and response types
//	password.go    — Argon2id hashing (service-layer utility)
//	middleware.go  — Auth middleware and actor extraction
package auth
