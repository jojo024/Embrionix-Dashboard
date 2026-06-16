package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/gin-gonic/gin"
)

func parseUintParam(c *gin.Context, name string) uint {
	n, _ := strconv.ParseUint(c.Param(name), 10, 64)
	return uint(n)
}

// AuthHandler covers login, the current-user probe, and user management.
type AuthHandler struct {
	authSvc  *services.AuthService
	userRepo *repositories.UserRepository
}

func NewAuthHandler(authSvc *services.AuthService, userRepo *repositories.UserRepository) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, userRepo: userRepo}
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	if !h.authSvc.Enabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authentication is disabled"})
		return
	}
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, user, err := h.authSvc.Authenticate(body.Username, body.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID, "username": user.Username, "role": user.Role},
	})
}

// Me GET /api/v1/auth/me — reports auth state and the caller's role.
func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"auth_enabled": h.authSvc.Enabled(),
		"username":     c.GetString("username"),
		"role":         c.GetString("role"),
	})
}

// ListUsers GET /api/v1/users (admin)
func (h *AuthHandler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "total": len(users)})
}

// CreateUser POST /api/v1/users (admin)
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var body struct {
		Username string      `json:"username" binding:"required"`
		Password string      `json:"password" binding:"required"`
		Role     models.Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !body.Role.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be viewer, operator or admin"})
		return
	}
	hash, err := services.HashPassword(body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	user := &models.User{Username: body.Username, PasswordHash: hash, Role: body.Role, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := h.userRepo.Create(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not create user (username may be taken)"})
		return
	}
	c.JSON(http.StatusCreated, user)
}

// UpdateUser PUT /api/v1/users/:id (admin) — change role and/or password.
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	user, err := h.userRepo.FindByID(parseUintParam(c, "id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	var body struct {
		Role     *models.Role `json:"role"`
		Password *string      `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Role != nil {
		if !body.Role.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
			return
		}
		user.Role = *body.Role
	}
	if body.Password != nil && *body.Password != "" {
		hash, err := services.HashPassword(*body.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		user.PasswordHash = hash
	}
	user.UpdatedAt = time.Now()
	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

// DeleteUser DELETE /api/v1/users/:id (admin)
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	id := parseUintParam(c, "id")
	// Prevent deleting the last remaining account.
	if n, _ := h.userRepo.Count(); n <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last remaining user"})
		return
	}
	if err := h.userRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
