package implementation

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/domain/usecase"
	"github.com/physicist2018/lidar-platform-go/internal/utils/hash"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

type getAllUsersUseCaseImpl struct {
	repo repository.UserRepository
	log  *logrus.Logger
}

var _ usecase.GetAllUsersUseCase = (*getAllUsersUseCaseImpl)(nil)

func NewGetAllUsersUseCaseImpl(repo repository.UserRepository, log *logrus.Logger) *getAllUsersUseCaseImpl {
	return &getAllUsersUseCaseImpl{repo: repo, log: log}
}

func (u *getAllUsersUseCaseImpl) Execute(ctx context.Context, filter *entity.UserFilter) (*pagination.Pagination[entity.User], error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "GetAllUsersUseCase.Execute")
	defer span.End()

	start := time.Now()
	span.SetAttributes(
		attribute.Int("page", filter.Page),
		attribute.Int("limit", filter.Limit),
	)

	result, err := u.repo.FindAll(ctx, filter)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		u.log.WithFields(logrus.Fields{
			"operation": "GetAllUsersUseCase.Execute",
			"duration":  time.Since(start).String(),
			"error":     err,
		}).Error("failed to get all users")
		return nil, err
	}

	for i := range result.Data {
		result.Data[i].HidePassword()
	}

	u.log.WithFields(logrus.Fields{
		"operation": "GetAllUsersUseCase.Execute",
		"duration":  time.Since(start).String(),
		"count":     len(result.Data),
	}).Info("get all users success")

	return result, nil
}

// ---

type getUserByIDUseCaseImpl struct {
	repo repository.UserRepository
	log  *logrus.Logger
}

var _ usecase.GetUserByIDUseCase = (*getUserByIDUseCaseImpl)(nil)

func NewGetUserByIDUseCaseImpl(repo repository.UserRepository, log *logrus.Logger) *getUserByIDUseCaseImpl {
	return &getUserByIDUseCaseImpl{repo: repo, log: log}
}

func (u *getUserByIDUseCaseImpl) Execute(ctx context.Context, id uint) (*entity.User, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "GetUserByIDUseCase.Execute")
	defer span.End()

	start := time.Now()
	span.SetAttributes(attribute.Int("user_id", int(id)))

	user, err := u.repo.FindByID(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		u.log.WithFields(logrus.Fields{
			"operation": "GetUserByIDUseCase.Execute",
			"duration":  time.Since(start).String(),
			"error":     err,
		}).Error("failed to get user by id")
		return nil, err
	}

	u.log.WithFields(logrus.Fields{
		"operation": "GetUserByIDUseCase.Execute",
		"duration":  time.Since(start).String(),
	}).Info("get user by id success")

	return user, nil
}

// ---

type createUserUseCaseImpl struct {
	repo repository.UserRepository
	log  *logrus.Logger
}

var _ usecase.CreateUserUseCase = (*createUserUseCaseImpl)(nil)

func NewCreateUserUseCaseImpl(repo repository.UserRepository, log *logrus.Logger) *createUserUseCaseImpl {
	return &createUserUseCaseImpl{repo: repo, log: log}
}

func (u *createUserUseCaseImpl) Execute(ctx context.Context, user *entity.User) error {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "CreateUserUseCase.Execute")
	defer span.End()

	start := time.Now()
	span.SetAttributes(attribute.String("email", user.Email))

	// Check email uniqueness
	if _, err := u.repo.FindByEmail(ctx, user.Email); err == nil {
		span.SetStatus(codes.Error, entity.ErrEmailAlreadyExists.Error())
		return entity.ErrEmailAlreadyExists
	}

	// Hash password
	hashed, err := hash.Password(user.Password)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	user.Password = hashed

	if err := u.repo.Create(ctx, user); err != nil {
		span.SetStatus(codes.Error, err.Error())
		u.log.WithFields(logrus.Fields{
			"operation": "CreateUserUseCase.Execute",
			"duration":  time.Since(start).String(),
			"error":     err,
		}).Error("failed to create user")
		return err
	}

	u.log.WithFields(logrus.Fields{
		"operation": "CreateUserUseCase.Execute",
		"duration":  time.Since(start).String(),
	}).Info("create user success")

	return nil
}

// ---

type updateUserUseCaseImpl struct {
	repo repository.UserRepository
	log  *logrus.Logger
}

var _ usecase.UpdateUserUseCase = (*updateUserUseCaseImpl)(nil)

func NewUpdateUserUseCaseImpl(repo repository.UserRepository, log *logrus.Logger) *updateUserUseCaseImpl {
	return &updateUserUseCaseImpl{repo: repo, log: log}
}

func (u *updateUserUseCaseImpl) Execute(ctx context.Context, user *entity.User) error {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "UpdateUserUseCase.Execute")
	defer span.End()

	start := time.Now()
	span.SetAttributes(attribute.Int("user_id", int(user.ID)))

	// If the caller provided a new password, hash it
	if user.Password != "" {
		hashed, err := hash.Password(user.Password)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		user.Password = hashed
	}

	if err := u.repo.Update(ctx, user); err != nil {
		span.SetStatus(codes.Error, err.Error())
		u.log.WithFields(logrus.Fields{
			"operation": "UpdateUserUseCase.Execute",
			"duration":  time.Since(start).String(),
			"error":     err,
		}).Error("failed to update user")
		return err
	}

	u.log.WithFields(logrus.Fields{
		"operation": "UpdateUserUseCase.Execute",
		"duration":  time.Since(start).String(),
	}).Info("update user success")

	return nil
}

// ---

type deleteUserUseCaseImpl struct {
	repo repository.UserRepository
	log  *logrus.Logger
}

var _ usecase.DeleteUserUseCase = (*deleteUserUseCaseImpl)(nil)

func NewDeleteUserUseCaseImpl(repo repository.UserRepository, log *logrus.Logger) *deleteUserUseCaseImpl {
	return &deleteUserUseCaseImpl{repo: repo, log: log}
}

func (u *deleteUserUseCaseImpl) Execute(ctx context.Context, id uint) error {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "DeleteUserUseCase.Execute")
	defer span.End()

	start := time.Now()
	span.SetAttributes(attribute.Int("user_id", int(id)))

	if err := u.repo.Delete(ctx, id); err != nil {
		span.SetStatus(codes.Error, err.Error())
		u.log.WithFields(logrus.Fields{
			"operation": "DeleteUserUseCase.Execute",
			"duration":  time.Since(start).String(),
			"error":     err,
		}).Error("failed to delete user")
		return err
	}

	u.log.WithFields(logrus.Fields{
		"operation": "DeleteUserUseCase.Execute",
		"duration":  time.Since(start).String(),
	}).Info("delete user success")

	return nil
}
