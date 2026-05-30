package implementation

import (
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/kshmirko/lidar-platform-go/internal/utils/pagination"
)

var _ persistance.UserDataSource = (*UserDataSourceImpl)(nil)

type UserDataSourceImpl struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewUserDataSourceImpl(db *gorm.DB, log *logrus.Logger) *UserDataSourceImpl {
	return &UserDataSourceImpl{DB: db, Log: log}
}

func (d *UserDataSourceImpl) GetAll(ctx context.Context, filter *entity.UserFilter) ([]entity.User, int64, error) {
	var dbUsers []dbEntity.UserEntity
	var total int64

	query := d.DB.WithContext(ctx).Model(&dbEntity.UserEntity{})

	if filter.Role != "" {
		query = query.Where("role = ?", string(filter.Role))
	}
	if filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.Email != "" {
		query = query.Where("email ILIKE ?", "%"+filter.Email+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		d.Log.WithError(err).Error("UserDataSource.GetAll: count failed")
		return nil, 0, err
	}

	order := "id ASC"
	if filter.Sort == "desc" {
		order = "id DESC"
	}

	offset := pagination.Offset(filter.Page, filter.Limit)

	if err := query.Order(order).Offset(offset).Limit(filter.Limit).Find(&dbUsers).Error; err != nil {
		d.Log.WithError(err).Error("UserDataSource.GetAll: find failed")
		return nil, 0, err
	}

	users := make([]entity.User, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = entity.User{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			Role:     entity.UserRole(u.Role),
			Password: u.Password,
		}
	}

	return users, total, nil
}

func (d *UserDataSourceImpl) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	var dbUser dbEntity.UserEntity
	if err := d.DB.WithContext(ctx).First(&dbUser, id).Error; err != nil {
		return nil, err
	}

	return &entity.User{
		ID:       dbUser.ID,
		Name:     dbUser.Name,
		Email:    dbUser.Email,
		Role:     entity.UserRole(dbUser.Role),
		Password: dbUser.Password,
	}, nil
}

func (d *UserDataSourceImpl) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var dbUser dbEntity.UserEntity
	if err := d.DB.WithContext(ctx).Where("email = ?", email).First(&dbUser).Error; err != nil {
		return nil, err
	}

	return &entity.User{
		ID:       dbUser.ID,
		Name:     dbUser.Name,
		Email:    dbUser.Email,
		Role:     entity.UserRole(dbUser.Role),
		Password: dbUser.Password,
	}, nil
}

func (d *UserDataSourceImpl) Create(ctx context.Context, user *entity.User) error {
	dbUser := dbEntity.UserEntity{
		Name:     user.Name,
		Email:    user.Email,
		Role:     string(user.Role),
		Password: user.Password,
	}
	if err := d.DB.WithContext(ctx).Create(&dbUser).Error; err != nil {
		return err
	}
	user.ID = dbUser.ID
	return nil
}

func (d *UserDataSourceImpl) Update(ctx context.Context, user *entity.User) error {
	return d.DB.WithContext(ctx).Model(&dbEntity.UserEntity{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"name":     user.Name,
		"email":    user.Email,
		"role":     string(user.Role),
		"password": user.Password,
	}).Error
}

func (d *UserDataSourceImpl) Delete(ctx context.Context, id uint) error {
	return d.DB.WithContext(ctx).Delete(&dbEntity.UserEntity{}, id).Error
}
