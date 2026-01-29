package userrepo

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"jan-server/mono/apps/backend/internal/domain/user"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// OIDCUserGormRepository implements user.OIDCRepository for OIDC users.
type OIDCUserGormRepository struct {
	db *gorm.DB
}

var _ user.OIDCRepository = (*OIDCUserGormRepository)(nil)

// NewUserGormRepository creates a new OIDC user repository.
func NewUserGormRepository(db *gorm.DB) user.OIDCRepository {
	return &OIDCUserGormRepository{db: db}
}

func (repo *OIDCUserGormRepository) FindByIssuerAndSubject(ctx context.Context, issuer, subject string) (*user.OIDCUser, error) {
	var entity dbschema.User
	err := repo.db.WithContext(ctx).
		Where("issuer = ? AND subject = ?", issuer, subject).
		First(&entity).
		Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find user by issuer and subject",
			err,
			"b2a7c2d5-53b2-44a3-8f8f-927f94e9a4db",
		)
	}
	return entity.EtoD(), nil
}

func (repo *OIDCUserGormRepository) FindOIDCByID(ctx context.Context, id uint) (*user.OIDCUser, error) {
	var entity dbschema.User
	err := repo.db.WithContext(ctx).
		Where("id = ?", id).
		First(&entity).
		Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find user by ID",
			err,
			"a9d3f8e4-21c7-4f5b-9a2e-6d8f9e1a2b3c",
		)
	}
	return entity.EtoD(), nil
}

func (repo *OIDCUserGormRepository) UpsertOIDC(ctx context.Context, usr *user.OIDCUser) (*user.OIDCUser, error) {
	// Prepare schema model from domain user
	schemaUser := dbschema.NewSchemaUser(usr)

	assignments := map[string]any{
		"auth_provider": schemaUser.AuthProvider,
		"username":      schemaUser.Username,
		"email":         schemaUser.Email,
		"name":          schemaUser.Name,
		"picture":       schemaUser.Picture,
		"updated_at":    gorm.Expr("NOW()"),
	}

	if err := repo.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "issuer"}, {Name: "subject"}},
			DoUpdates: clause.Assignments(assignments),
		}).
		Create(schemaUser).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to upsert user",
			err,
			"3b31d2bd-3260-4233-b0c8-09909fa0f154",
		)
	}

	// Retrieve the persisted user to capture ID and timestamps
	// Use a background context with timeout for the reload to avoid context cancellation
	// if the client disconnects after the upsert succeeds
	reloadCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var persisted dbschema.User
	if err := repo.db.WithContext(reloadCtx).
		Where("issuer = ? AND subject = ?", schemaUser.Issuer, schemaUser.Subject).
		First(&persisted).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to reload upserted user",
			err,
			"f71f98cb-3154-4ad2-9076-7e58628a4098",
		)
	}

	domainUser := persisted.EtoD()
	return domainUser, nil
}
