package controller

import (
	"net/http"

	"github.com/labstack/echo/v5"
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
func (ctrl *AuthController) Login(c *echo.Context) error {
	var body dto.LoginRequest
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}
	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	claims, token, err := ctrl.LoginUseCase.Execute(c.Request().Context(), body.Email, body.Password)
	if err != nil {
		code := http.StatusInternalServerError
		if ce, ok := err.(interface{ StatusCode() int }); ok {
			code = ce.StatusCode()
		}
		return c.JSON(code, dto.ErrorResponse{Error: err.Error()})
	}

	resp := dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:    claims.UserID,
			Email: claims.Email,
			Role:  claims.Role,
		},
	}

	return c.JSON(http.StatusOK, resp)
}
