package controller

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/middleware"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/response"
	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/utils/mapper"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

type UserController struct {
	Log           *logrus.Logger
	GetAllUsersUC usecase.GetAllUsersUseCase
	GetUserByIDUC usecase.GetUserByIDUseCase
	CreateUserUC  usecase.CreateUserUseCase
	UpdateUserUC  usecase.UpdateUserUseCase
	DeleteUserUC  usecase.DeleteUserUseCase
	Validate      *validator.Validate
}

func NewUserController(
	log *logrus.Logger,
	getAll usecase.GetAllUsersUseCase,
	getByID usecase.GetUserByIDUseCase,
	create usecase.CreateUserUseCase,
	update usecase.UpdateUserUseCase,
	delete usecase.DeleteUserUseCase,
	validate *validator.Validate,
) *UserController {
	return &UserController{
		Log:           log,
		GetAllUsersUC: getAll,
		GetUserByIDUC: getByID,
		CreateUserUC:  create,
		UpdateUserUC:  update,
		DeleteUserUC:  delete,
		Validate:      validate,
	}
}

// GetAll godoc
//
//	@Summary		List all users
//	@Description	Returns a paginated list of users with optional filtering by role, name and email.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int		false	"Page number"	default(1)		minimum(1)
//	@Param			limit	query		int		false	"Items per page"	default(10)	minimum(1)	maximum(100)
//	@Param			sort	query		string	false	"Sort direction"	Enums(asc, desc)
//	@Param			role	query		string	false	"Filter by role"	Enums(admin, guest, manager)
//	@Param			name	query		string	false	"Filter by name (case-insensitive partial match)"
//	@Param			email	query		string	false	"Filter by email (case-insensitive partial match)"
//	@Success		200		{object}	dto.UserPaginatedResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Bad request — invalid query parameters"
//	@Failure		401		{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/users [get]
func (ctrl *UserController) GetAll(w http.ResponseWriter, r *http.Request) {
	query := dto.GetAllUsersQuery{
		Page:  1,
		Limit: 10,
	}

	q := r.URL.Query()
	if v := q.Get("page"); v != "" {
		query.Page = parseInt(v, 1)
	}
	if v := q.Get("limit"); v != "" {
		query.Limit = parseInt(v, 10)
	}
	query.Sort = q.Get("sort")
	query.Role = q.Get("role")
	query.Name = q.Get("name")
	query.Email = q.Get("email")

	filter := &entity.UserFilter{
		Page:  query.Page,
		Limit: query.Limit,
		Sort:  query.Sort,
		Role:  entity.UserRole(query.Role),
		Name:  query.Name,
		Email: query.Email,
	}

	result, err := ctrl.GetAllUsersUC.Execute(r.Context(), filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, mapper.ToUserResponseList(result))
}

// GetByID godoc
//
//	@Summary		Get user by ID
//	@Description	Returns a single user by its database ID.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		uint	true	"User ID"	minimum(1)
//	@Success		200	{object}	dto.UserResponse
//	@Failure		400	{object}	dto.ErrorResponse	"Bad request — invalid ID"
//	@Failure		401	{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	dto.ErrorResponse	"User not found"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/users/{id} [get]
func (ctrl *UserController) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := parseUint(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	user, err := ctrl.GetUserByIDUC.Execute(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	user.HidePassword()
	response.JSON(w, http.StatusOK, mapper.ToUserResponse(user))
}

// Create godoc
//
//	@Summary		Create a new user
//	@Description	Creates a new user account. Email must be unique. Password is bcrypt-hashed. **Requires admin role.**
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateUserBody	true	"User creation payload"
//	@Success		201		{object}	dto.UserResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Bad request — validation failed"
//	@Failure		401		{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	dto.ErrorResponse	"Forbidden — admin role required"
//	@Failure		409		{object}	dto.ErrorResponse	"Conflict — email already exists"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/users [post]
func (ctrl *UserController) Create(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateUserBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctrl.Validate.Struct(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user := &entity.User{
		Name:     body.Name,
		Email:    body.Email,
		Role:     entity.UserRole(body.Role),
		Password: body.Password,
	}

	if err := ctrl.CreateUserUC.Execute(r.Context(), user); err != nil {
		code := http.StatusInternalServerError
		if ce, ok := err.(interface{ StatusCode() int }); ok {
			code = ce.StatusCode()
		}
		response.Error(w, code, err.Error())
		return
	}

	user.HidePassword()
	response.JSON(w, http.StatusCreated, mapper.ToUserResponse(user))
}

// Update godoc
//
//	@Summary		Update existing user
//	@Description	Updates user fields. If password is omitted, the existing password is preserved. **Requires admin role.**
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		uint				true	"User ID"	minimum(1)
//	@Param			body	body		dto.UpdateUserBody	true	"Fields to update"
//	@Success		200		{object}	dto.UserResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Bad request — validation failed"
//	@Failure		401		{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	dto.ErrorResponse	"Forbidden — admin role required"
//	@Failure		404		{object}	dto.ErrorResponse	"User not found"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/users/{id} [put]
func (ctrl *UserController) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := parseUint(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body dto.UpdateUserBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctrl.Validate.Struct(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := ctrl.GetUserByIDUC.Execute(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	user := &entity.User{
		ID:       id,
		Name:     body.Name,
		Email:    body.Email,
		Role:     entity.UserRole(body.Role),
		Password: existing.Password,
	}
	if body.Password != "" {
		user.Password = body.Password
	}

	if err := ctrl.UpdateUserUC.Execute(r.Context(), user); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	user.HidePassword()
	response.JSON(w, http.StatusOK, mapper.ToUserResponse(user))
}

// Delete godoc
//
//	@Summary		Delete user
//	@Description	Permanently removes a user (soft-delete via GORM deleted_at). **Requires admin role.** Self-deletion is forbidden.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		uint	true	"User ID"	minimum(1)
//	@Success		204	"No Content — user deleted"
//	@Failure		400	{object}	dto.ErrorResponse	"Bad request — invalid ID"
//	@Failure		401	{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		403	{object}	dto.ErrorResponse	"Forbidden — admin role required or self-deletion"
//	@Failure		404	{object}	dto.ErrorResponse	"User not found"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/users/{id} [delete]
func (ctrl *UserController) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := parseUint(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := middleware.GetClaims(r)
	if claims != nil && claims.UserID == id {
		response.Error(w, http.StatusForbidden, entity.ErrAdminRequired.Error())
		return
	}

	if err := ctrl.DeleteUserUC.Execute(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
