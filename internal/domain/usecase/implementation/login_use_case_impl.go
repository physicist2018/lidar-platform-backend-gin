package implementation

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/domain/usecase"
	"github.com/physicist2018/lidar-platform-go/internal/utils/auth"
	"github.com/physicist2018/lidar-platform-go/internal/utils/hash"
)

type loginUseCaseImpl struct {
	repo      repository.UserRepository
	jwtConfig auth.JWTConfig
	log       *logrus.Logger
}

var _ usecase.LoginUseCase = (*loginUseCaseImpl)(nil)

func NewLoginUseCaseImpl(repo repository.UserRepository, jwtConfig auth.JWTConfig, log *logrus.Logger) *loginUseCaseImpl {
	return &loginUseCaseImpl{repo: repo, jwtConfig: jwtConfig, log: log}
}

func (u *loginUseCaseImpl) Execute(ctx context.Context, email, password string) (*auth.Claims, string, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "LoginUseCase.Execute")
	defer span.End()

	start := time.Now()

	user, err := u.repo.FindByEmail(ctx, email)
	if err != nil {
		span.SetStatus(codes.Error, entity.ErrInvalidCredentials.Error())
		u.log.WithFields(logrus.Fields{
			"operation": "LoginUseCase.Execute",
			"duration":  time.Since(start).String(),
			"email":     email,
		}).Warn("login failed: user not found")
		return nil, "", entity.ErrInvalidCredentials
	}

	if !hash.CheckPassword(password, user.Password) {
		span.SetStatus(codes.Error, entity.ErrInvalidCredentials.Error())
		u.log.WithFields(logrus.Fields{
			"operation": "LoginUseCase.Execute",
			"duration":  time.Since(start).String(),
			"email":     email,
		}).Warn("login failed: wrong password")
		return nil, "", entity.ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(u.jwtConfig, user.ID, user.Email, string(user.Role))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, "", err
	}

	claims := &auth.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   string(user.Role),
	}

	u.log.WithFields(logrus.Fields{
		"operation": "LoginUseCase.Execute",
		"duration":  time.Since(start).String(),
		"user_id":   user.ID,
	}).Info("login success")

	return claims, token, nil
}
