package profile

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Standard error codes for API responses.
const (
	ErrCodeProfileNotFound = "PROFILE_NOT_FOUND"
	ErrCodeInvalidUserID   = "INVALID_USER_ID"
	ErrCodeInternal        = "INTERNAL_ERROR"
)

// Controller handles HTTP requests for profiles.
type Controller struct {
	service Service
}

// NewController creates a new profile controller.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// RegisterRoutes registers profile routes on the given router group.
func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	profiles := router.Group("/profile")
	{
		profiles.GET("/:userId", c.GetByUserID)
	}
}

// GetByUserID godoc
// @Summary Get user profile
// @Description Retrieve complete user profile including skills and progress by user ID
// @Tags profiles
// @Produce json
// @Param userId path string true "User ID from Kratos"
// @Success 200 {object} Profile "Profile retrieved successfully"
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 404 {object} ErrorResponse "Profile not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /profile/{userId} [get]
func (c *Controller) GetByUserID(ctx *gin.Context) {
	userID := ctx.Param("userId")

	if userID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    ErrCodeInvalidUserID,
			Message: "userId parameter is required",
		})
		return
	}

	if len(userID) > 255 {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    ErrCodeInvalidUserID,
			Message: "userId exceeds maximum length of 255 characters",
		})
		return
	}

	profile, err := c.service.GetProfile(ctx.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			ctx.JSON(http.StatusNotFound, ErrorResponse{
				Code:    ErrCodeProfileNotFound,
				Message: "profile not found",
				Details: err.Error(),
			})
		case errors.Is(err, ErrInvalidUserID):
			ctx.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    ErrCodeInvalidUserID,
				Message: "invalid user ID",
				Details: err.Error(),
			})
		default:
			ctx.JSON(http.StatusInternalServerError, ErrorResponse{
				Code:    ErrCodeInternal,
				Message: "internal server error",
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, profile)
}

// ErrorResponse represents a standard API error.
type ErrorResponse struct {
	Code    string `json:"code" example:"PROFILE_NOT_FOUND"`
	Message string `json:"message" example:"profile not found"`
	Details string `json:"details,omitempty" example:"profile not found: kratos-user-123"`
}
