package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

type AuthController struct {
	Log          *logrus.Logger
	LoginUseCase usecase.LoginUseCase
}

func NewAuthController(log *logrus.Logger, loginUC usecase.LoginUseCase) *AuthController {
	return &AuthController{Log: log, LoginUseCase: loginUC}
}

// Login godoc
//
//	@Summary		Login
//	@Description	Authenticate with email and password. Returns a JWT Bearer token.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.LoginRequest	true	"Credentials"
//	@Success		200		{object}	dto.LoginResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Bad request — validation failed"
//	@Failure		401		{object}	dto.ErrorResponse	"Invalid credentials"
//	@Router			/auth/login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	var body dto.LoginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(err)
		return
	}

	claims, token, err := ctrl.LoginUseCase.Execute(c.Request.Context(), body.Email, body.Password)
	if err != nil {
		c.Error(err)
		return
	}

	resp := dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:    claims.UserID,
			Email: claims.Email,
			Role:  claims.Role,
		},
	}

	c.JSON(http.StatusOK, resp)
}
