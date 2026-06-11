package controller

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/response"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

type AuthController struct {
	Log          *logrus.Logger
	LoginUseCase usecase.LoginUseCase
	Validate     *validator.Validate
}

func NewAuthController(log *logrus.Logger, loginUC usecase.LoginUseCase, validate *validator.Validate) *AuthController {
	return &AuthController{Log: log, LoginUseCase: loginUC, Validate: validate}
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
func (ctrl *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var body dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctrl.Validate.Struct(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	claims, token, err := ctrl.LoginUseCase.Execute(r.Context(), body.Email, body.Password)
	if err != nil {
		code := http.StatusInternalServerError
		if ce, ok := err.(interface{ StatusCode() int }); ok {
			code = ce.StatusCode()
		}
		response.Error(w, code, err.Error())
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

	response.JSON(w, http.StatusOK, resp)
}
