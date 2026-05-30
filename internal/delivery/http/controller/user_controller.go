package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/middleware"
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
}

func NewUserController(
	log *logrus.Logger,
	getAll usecase.GetAllUsersUseCase,
	getByID usecase.GetUserByIDUseCase,
	create usecase.CreateUserUseCase,
	update usecase.UpdateUserUseCase,
	delete usecase.DeleteUserUseCase,
) *UserController {
	return &UserController{
		Log:           log,
		GetAllUsersUC: getAll,
		GetUserByIDUC: getByID,
		CreateUserUC:  create,
		UpdateUserUC:  update,
		DeleteUserUC:  delete,
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
func (ctrl *UserController) GetAll(c *gin.Context) {
	var query dto.GetAllUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(err)
		return
	}

	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 10
	}

	filter := &entity.UserFilter{
		Page:  query.Page,
		Limit: query.Limit,
		Sort:  query.Sort,
		Role:  entity.UserRole(query.Role),
		Name:  query.Name,
		Email: query.Email,
	}

	result, err := ctrl.GetAllUsersUC.Execute(c.Request.Context(), filter)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, mapper.ToUserResponseList(result))
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
func (ctrl *UserController) GetByID(c *gin.Context) {
	var uri dto.GetUserByIDUri
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(err)
		return
	}

	user, err := ctrl.GetUserByIDUC.Execute(c.Request.Context(), uri.ID)
	if err != nil {
		c.Error(err)
		return
	}

	user.HidePassword()
	c.JSON(http.StatusOK, mapper.ToUserResponse(user))
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
func (ctrl *UserController) Create(c *gin.Context) {
	var body dto.CreateUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(err)
		return
	}

	user := &entity.User{
		Name:     body.Name,
		Email:    body.Email,
		Role:     entity.UserRole(body.Role),
		Password: body.Password,
	}

	if err := ctrl.CreateUserUC.Execute(c.Request.Context(), user); err != nil {
		c.Error(err)
		return
	}

	user.HidePassword()
	c.JSON(http.StatusCreated, mapper.ToUserResponse(user))
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
func (ctrl *UserController) Update(c *gin.Context) {
	var uri dto.UpdateUserUri
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(err)
		return
	}

	var body dto.UpdateUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(err)
		return
	}

	existing, err := ctrl.GetUserByIDUC.Execute(c.Request.Context(), uri.ID)
	if err != nil {
		c.Error(err)
		return
	}

	user := &entity.User{
		ID:       uri.ID,
		Name:     body.Name,
		Email:    body.Email,
		Role:     entity.UserRole(body.Role),
		Password: existing.Password,
	}
	if body.Password != "" {
		user.Password = body.Password
	}

	if err := ctrl.UpdateUserUC.Execute(c.Request.Context(), user); err != nil {
		c.Error(err)
		return
	}

	user.HidePassword()
	c.JSON(http.StatusOK, mapper.ToUserResponse(user))
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
func (ctrl *UserController) Delete(c *gin.Context) {
	var uri dto.DeleteUserUri
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(err)
		return
	}

	claims := middleware.GetClaims(c)
	if claims != nil && claims.UserID == uri.ID {
		c.Error(entity.ErrAdminRequired) // can't delete yourself
		return
	}

	if err := ctrl.DeleteUserUC.Execute(c.Request.Context(), uri.ID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
