---
name: go-openapi-docs
description: Triggers whenever a new Go API endpoint, handler, or router group is created, modified, or refactored. Ensures the latest version of OpenAPI/Swagger specs are generated or updated.
---

# Go OpenAPI Documentation Standard

You are an expert Go backend engineer. Every time you implement, modify, or extend a Go endpoint, you must strictly document it using the latest OpenAPI specification standard via standard code comments or declarative annotations.

## Execution Rules

1. **Detect Framework:** Identify if the project is using `Gin`, `Fiber`, `Echo`, or standard `net/http`.
2. **Declarative Annotation:** Generate appropriate documentation tags directly above the handler function (e.g., using `swag` / `swaggo` conventions for Go).
3. **Spec Completeness:** Always include:
   - `@Summary` and `@Description`
   - `@Tags` (grouped logically)
   - `@Accept json` and `@Produce json`
   - `@Param` mapping path, query, or body structures
   - `@Success` and `@Failure` responses pointing to exact Go structs
4. **Validation:** If the workspace contains a script to regenerate the swagger files (e.g., `swag init`), run it in the terminal to verify the documentation compiles without errors.

## Example Pattern

```go
// @Summary      Create a new user
// @Description  Takes a JSON payload to register a new system user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user  body      models.CreateUserRequest  true  "User Payload"
// @Success      201   {object}  models.UserResponse
// @Failure      400   {object}  models.ErrorResponse
// @Router       /users [post]
func (h *Handler) CreateUser(c *gin.Context) { ... }
