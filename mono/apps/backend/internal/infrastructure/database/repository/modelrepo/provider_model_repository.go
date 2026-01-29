package modelrepo

import (
	"context"
	"strings"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/infrastructure/database/gormgen"
	"jan-server/mono/apps/backend/internal/infrastructure/database/transaction"
)

type ProviderModelGormRepository struct {
	db *transaction.Database
}

var _ domainmodel.ProviderModelRepository = (*ProviderModelGormRepository)(nil)

func NewProviderModelGormRepository(db *transaction.Database) domainmodel.ProviderModelRepository {
	return &ProviderModelGormRepository{db: db}
}

func (repo *ProviderModelGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IProviderModelDo, filter domainmodel.ProviderModelFilter) gormgen.IProviderModelDo {
	if filter.IDs != nil && len(*filter.IDs) > 0 {
		sql = sql.Where(query.ProviderModel.ID.In((*filter.IDs)...))
	}
	if filter.PublicID != nil {
		sql = sql.Where(query.ProviderModel.PublicID.Eq(*filter.PublicID))
	}
	if filter.ProviderID != nil {
		sql = sql.Where(query.ProviderModel.ProviderID.Eq(*filter.ProviderID))
	}
	if filter.ProviderIDs != nil && len(*filter.ProviderIDs) > 0 {
		sql = sql.Where(query.ProviderModel.ProviderID.In((*filter.ProviderIDs)...))
	}
	if filter.ModelCatalogID != nil {
		sql = sql.Where(query.ProviderModel.ModelCatalogID.Eq(*filter.ModelCatalogID))
	}
	if filter.ModelPublicID != nil {
		sql = sql.Where(query.ProviderModel.ModelPublicID.Eq(*filter.ModelPublicID))
	}
	if filter.ModelPublicIDs != nil && len(*filter.ModelPublicIDs) > 0 {
		sql = sql.Where(query.ProviderModel.ModelPublicID.In((*filter.ModelPublicIDs)...))
	}
	if filter.ProviderOriginalModelID != nil {
		sql = sql.Where(query.ProviderModel.ProviderOriginalModelID.Eq(*filter.ProviderOriginalModelID))
	}
	if filter.Active != nil {
		sql = sql.Where(query.ProviderModel.Active.Is(*filter.Active))
	}
	if filter.SupportsImages != nil {
		sql = sql.LeftJoin(query.ModelCatalog, query.ModelCatalog.ID.EqCol(query.ProviderModel.ModelCatalogID))
		sql = sql.Where(query.ModelCatalog.SupportsImages.Is(*filter.SupportsImages))
	}
	if filter.SearchText != nil && strings.TrimSpace(*filter.SearchText) != "" {
		pat := "%" + strings.TrimSpace(*filter.SearchText) + "%"
		cond1 := query.ProviderModel.ModelPublicID.Like(pat)
		cond2 := query.ProviderModel.ModelDisplayName.Like(pat)
		cond3 := query.ProviderModel.ProviderOriginalModelID.Like(pat)
		cond4 := query.ProviderModel.Kind.Like(pat)
		sql = sql.Where(cond1).Or(cond2).Or(cond3).Or(cond4)
	}
	return sql
}

func (repo *ProviderModelGormRepository) Create(ctx context.Context, model *domainmodel.ProviderModel) error {
	schemaModel, err := dbschema.NewSchemaProviderModel(model)
	if err != nil {
		return err
	}
	query := repo.db.GetQuery(ctx)
	if err := query.ProviderModel.WithContext(ctx).Create(schemaModel); err != nil {
		return err
	}
	model.ID = schemaModel.ID
	model.CreatedAt = schemaModel.CreatedAt
	model.UpdatedAt = schemaModel.UpdatedAt
	return nil
}

func (repo *ProviderModelGormRepository) Update(ctx context.Context, model *domainmodel.ProviderModel) error {
	schemaModel, err := dbschema.NewSchemaProviderModel(model)
	if err != nil {
		return err
	}
	query := repo.db.GetQuery(ctx)
	_, err = query.ProviderModel.WithContext(ctx).Where(query.ProviderModel.ID.Eq(model.ID)).Updates(schemaModel)
	return err
}

func (repo *ProviderModelGormRepository) DeleteByID(ctx context.Context, id uint) error {
	query := repo.db.GetQuery(ctx)
	_, err := query.ProviderModel.WithContext(ctx).Where(query.ProviderModel.ID.Eq(id)).Delete(&dbschema.ProviderModel{})
	return err
}

func (repo *ProviderModelGormRepository) FindByID(ctx context.Context, id uint) (*domainmodel.ProviderModel, error) {
	query := repo.db.GetQuery(ctx)
	schemaModel, err := query.ProviderModel.WithContext(ctx).Where(query.ProviderModel.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return schemaModel.EtoD()
}

func (repo *ProviderModelGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domainmodel.ProviderModel, error) {
	filter := domainmodel.ProviderModelFilter{
		PublicID: &publicID,
	}
	results, err := repo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

func (repo *ProviderModelGormRepository) FindByFilter(ctx context.Context, filter domainmodel.ProviderModelFilter, p *query.Pagination) ([]*domainmodel.ProviderModel, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ProviderModel.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.Offset != nil && *p.Offset >= 0 {
			sql = sql.Offset(*p.Offset)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.ProviderModel.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.ProviderModel.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.ProviderModel.ID.Desc())
		} else {
			sql = sql.Order(query.ProviderModel.ID.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := make([]*domainmodel.ProviderModel, 0, len(rows))
	for _, item := range rows {
		domainItem, err := item.EtoD()
		if err != nil {
			return nil, err
		}
		result = append(result, domainItem)
	}
	return result, nil
}

func (repo *ProviderModelGormRepository) Count(ctx context.Context, filter domainmodel.ProviderModelFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ProviderModel.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	return sql.Count()
}

func (repo *ProviderModelGormRepository) BatchUpdateActive(ctx context.Context, filter domainmodel.ProviderModelFilter, active bool) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ProviderModel.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	result, err := sql.Update(query.ProviderModel.Active, active)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}

func (repo *ProviderModelGormRepository) BatchUpdateModelDisplayName(ctx context.Context, filter domainmodel.ProviderModelFilter, modelDisplayName string) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ProviderModel.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	result, err := sql.Update(query.ProviderModel.ModelDisplayName, modelDisplayName)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}
