package modelrepo

import (
	"context"
	"errors"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/infrastructure/database/gormgen"
	"jan-server/mono/apps/backend/internal/infrastructure/database/transaction"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"

	"gorm.io/gorm"
)

type ModelCatalogGormRepository struct {
	db *transaction.Database
}

var _ domainmodel.ModelCatalogRepository = (*ModelCatalogGormRepository)(nil)

func NewModelCatalogGormRepository(db *transaction.Database) domainmodel.ModelCatalogRepository {
	return &ModelCatalogGormRepository{db: db}
}

func (repo *ModelCatalogGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IModelCatalogDo, filter domainmodel.ModelCatalogFilter) gormgen.IModelCatalogDo {
	if filter.IDs != nil && len(*filter.IDs) > 0 {
		sql = sql.Where(query.ModelCatalog.ID.In((*filter.IDs)...))
	}
	if filter.PublicID != nil {
		sql = sql.Where(query.ModelCatalog.PublicID.Eq(*filter.PublicID))
	}
	if filter.IsModerated != nil {
		sql = sql.Where(query.ModelCatalog.IsModerated.Is(*filter.IsModerated))
	}

	if filter.Active != nil {
		sql = sql.Where(query.ModelCatalog.Active.Is(*filter.Active))
	}
	if filter.Status != nil {
		sql = sql.Where(query.ModelCatalog.Status.Eq(string(*filter.Status)))
	}
	if filter.Experimental != nil {
		sql = sql.Where(query.ModelCatalog.Experimental.Is(*filter.Experimental))
	}
	if filter.RequiresFeatureFlag != nil {
		sql = sql.Where(query.ModelCatalog.RequiresFeatureFlag.Eq(*filter.RequiresFeatureFlag))
	}
	if filter.SupportsImages != nil {
		sql = sql.Where(query.ModelCatalog.SupportsImages.Is(*filter.SupportsImages))
	}
	if filter.SupportsEmbeddings != nil {
		sql = sql.Where(query.ModelCatalog.SupportsEmbeddings.Is(*filter.SupportsEmbeddings))
	}
	if filter.SupportsReasoning != nil {
		sql = sql.Where(query.ModelCatalog.SupportsReasoning.Is(*filter.SupportsReasoning))
	}
	if filter.SupportsInstruct != nil {
		sql = sql.Where(query.ModelCatalog.SupportsInstruct.Is(*filter.SupportsInstruct))
	}
	if filter.SupportsAudio != nil {
		sql = sql.Where(query.ModelCatalog.SupportsAudio.Is(*filter.SupportsAudio))
	}
	if filter.SupportsVideo != nil {
		sql = sql.Where(query.ModelCatalog.SupportsVideo.Is(*filter.SupportsVideo))
	}
	if filter.SupportsTools != nil {
		sql = sql.Where(query.ModelCatalog.SupportsTools.Is(*filter.SupportsTools))
	}
	if filter.SupportsBrowser != nil {
		sql = sql.Where(query.ModelCatalog.SupportsBrowser.Is(*filter.SupportsBrowser))
	}
	if filter.Family != nil && *filter.Family != "" {
		sql = sql.Where(query.ModelCatalog.Family.Eq(*filter.Family))
	}
	return sql
}

func (repo *ModelCatalogGormRepository) Create(ctx context.Context, catalog *domainmodel.ModelCatalog) error {
	model, err := dbschema.NewSchemaModelCatalog(catalog)
	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeValidation, "failed to convert model catalog to schema", err, "4850d796-eba9-4027-822a-8c1db9633fe0")
	}
	query := repo.db.GetQuery(ctx)
	if err := query.ModelCatalog.WithContext(ctx).Create(model); err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to create model catalog", err, "576a4099-91ff-4af7-b53a-898336b6ac94")
	}
	catalog.ID = model.ID
	catalog.CreatedAt = model.CreatedAt
	catalog.UpdatedAt = model.UpdatedAt
	catalog.Status = domainmodel.ModelCatalogStatus(model.Status)
	return nil
}

func (repo *ModelCatalogGormRepository) Update(ctx context.Context, catalog *domainmodel.ModelCatalog) error {
	model, err := dbschema.NewSchemaModelCatalog(catalog)
	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeValidation, "failed to convert model catalog to schema", err, "01276cbf-469c-4f3c-ae07-d572baf74f87")
	}
	query := repo.db.GetQuery(ctx)
	_, err = query.ModelCatalog.WithContext(ctx).Where(query.ModelCatalog.ID.Eq(model.ID)).Updates(model)
	return err

}

func (repo *ModelCatalogGormRepository) DeleteByID(ctx context.Context, id uint) error {
	query := repo.db.GetQuery(ctx)
	_, err := query.ModelCatalog.WithContext(ctx).
		Where(query.ModelCatalog.ID.Eq(id)).
		Delete(&dbschema.ModelCatalog{})
	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to delete model catalog", err, "27d71df3-0b21-4793-8042-19d3df51ac01")
	}
	return nil
}

func (repo *ModelCatalogGormRepository) FindByID(ctx context.Context, id uint) (*domainmodel.ModelCatalog, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.ModelCatalog.WithContext(ctx).Where(query.ModelCatalog.ID.Eq(id)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "model catalog not found", err, "d97cf4f2-b638-443b-9c4b-afe1de66fe25")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find model catalog by ID", err, "dc31efe7-13e0-41df-9f20-ac366aa4d437")
	}
	catalog, err := model.EtoD()
	if err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal, "failed to convert model catalog from schema", err, "6fe551cc-e2c9-40ff-915b-5e88b84126b6")
	}
	return catalog, nil
}

func (repo *ModelCatalogGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domainmodel.ModelCatalog, error) {
	filter := domainmodel.ModelCatalogFilter{
		PublicID: &publicID,
	}
	results, err := repo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, platformerrors.NewErrorWithContext(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "model catalog not found", nil, "772954bd-28fa-46f6-9237-057a1e61f8fd", map[string]any{
			"public_id": publicID,
		})
	}
	if len(results) > 1 {
		return nil, platformerrors.NewErrorWithContext(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeTooManyRecords,
			"multiple model catalogs found with same public ID", nil, "", map[string]any{
				"public_id": publicID,
				"count":     len(results),
			})
	}
	return results[0], nil
}

func (repo *ModelCatalogGormRepository) FindByFilter(ctx context.Context, filter domainmodel.ModelCatalogFilter, p *query.Pagination) ([]*domainmodel.ModelCatalog, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ModelCatalog.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.Offset != nil && *p.Offset >= 0 {
			sql = sql.Offset(*p.Offset)
		}
		if p.Order == "desc" {
			sql = sql.Order(query.ModelCatalog.CreatedAt.Desc())
		} else {
			sql = sql.Order(query.ModelCatalog.CreatedAt.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find model catalogs by filter", err, "982b14bf-11b8-4e96-9e74-b0fb689d81c1")
	}
	result := make([]*domainmodel.ModelCatalog, 0, len(rows))
	for _, item := range rows {
		domainItem, err := item.EtoD()
		if err != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal, "failed to convert model catalog from schema", err, "4e448960-8811-4401-a385-fd658ce54816")
		}
		result = append(result, domainItem)
	}
	return result, nil
}

func (repo *ModelCatalogGormRepository) Count(ctx context.Context, filter domainmodel.ModelCatalogFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ModelCatalog.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	count, err := sql.Count()
	if err != nil {
		return 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to count model catalogs", err, "7830ad79-8377-4ae8-bf7e-3c9e1382644c")
	}
	return count, nil
}

func (repo *ModelCatalogGormRepository) BatchUpdateActive(ctx context.Context, filter domainmodel.ModelCatalogFilter, active bool) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ModelCatalog.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	result, err := sql.Update(query.ModelCatalog.Active, active)
	if err != nil {
		return 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to batch update model catalog active status", err, "e1878f1b-e6ca-4753-926a-21c41c02c201")
	}
	return result.RowsAffected, nil
}

func (repo *ModelCatalogGormRepository) FindByIDs(ctx context.Context, ids []uint) ([]*domainmodel.ModelCatalog, error) {
	if len(ids) == 0 {
		return []*domainmodel.ModelCatalog{}, nil
	}

	filter := domainmodel.ModelCatalogFilter{
		IDs: &ids,
	}
	return repo.FindByFilter(ctx, filter, nil)
}

func (repo *ModelCatalogGormRepository) FindByPublicIDs(ctx context.Context, publicIDs []string) ([]*domainmodel.ModelCatalog, error) {
	if len(publicIDs) == 0 {
		return []*domainmodel.ModelCatalog{}, nil
	}

	query := repo.db.GetQuery(ctx)
	rows, err := query.ModelCatalog.WithContext(ctx).
		Where(query.ModelCatalog.PublicID.In(publicIDs...)).
		Find()
	if err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find model catalogs by public IDs", err, "fcd6a1a6-b9bd-43f4-8f57-dbfa21a6d489")
	}

	catalogs := make([]*domainmodel.ModelCatalog, 0, len(rows))
	for _, item := range rows {
		catalog, err := item.EtoD()
		if err != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal, "failed to convert model catalog from schema", err, "3f55f3ea-b139-407d-aa43-3952881de22f")
		}
		catalogs = append(catalogs, catalog)
	}
	return catalogs, nil
}
