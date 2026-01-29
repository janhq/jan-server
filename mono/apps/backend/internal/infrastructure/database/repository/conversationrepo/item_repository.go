package conversationrepo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	domain "jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// ItemRepository persists conversation items.
type ItemRepository struct {
	db *gorm.DB
}

var _ domain.ItemRepository = (*ItemRepository)(nil)

// NewItemGormRepository constructs the item repository implementing ItemRepository.
func NewItemGormRepository(db *gorm.DB) domain.ItemRepository {
	return &ItemRepository{db: db}
}

// Create inserts a single conversation item.
func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) error {
	entity := dbschema.NewSchemaConversationItem(item)
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create conversation item",
			err,
			"create-item-error",
		)
	}
	item.ID = entity.ID
	return nil
}

// BulkInsert stores multiple conversation items in sequence order.
func (r *ItemRepository) BulkInsert(ctx context.Context, items []domain.Item) error {
	if len(items) == 0 {
		return nil
	}

	rows := make([]dbschema.ConversationItem, 0, len(items))
	for _, item := range items {
		rows = append(rows, *dbschema.NewSchemaConversationItem(&item))
	}

	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to bulk insert conversation items",
			err,
			"7l8k9j0i-1a2b-3c4d-5e6f-7a8b9c0d1e2f",
		)
	}
	return nil
}

// BulkCreate stores multiple conversation items (pointer version).
func (r *ItemRepository) BulkCreate(ctx context.Context, items []*domain.Item) error {
	if len(items) == 0 {
		return nil
	}

	rows := make([]dbschema.ConversationItem, 0, len(items))
	for _, item := range items {
		rows = append(rows, *dbschema.NewSchemaConversationItem(item))
	}

	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to bulk create conversation items",
			err,
			"bulk-create-error",
		)
	}

	for i, row := range rows {
		items[i].ID = row.ID
	}
	return nil
}

// FindByID retrieves an item by its internal ID.
func (r *ItemRepository) FindByID(ctx context.Context, id uint) (*domain.Item, error) {
	var entity dbschema.ConversationItem
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("item not found: %d", id),
				nil,
				"find-by-id-not-found",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find item",
			err,
			"find-by-id-error",
		)
	}
	return entity.EtoD(), nil
}

// FindByPublicID retrieves an item by its public ID.
func (r *ItemRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Item, error) {
	var entity dbschema.ConversationItem
	if err := r.db.WithContext(ctx).Where("public_id = ?", publicID).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("item not found: %s", publicID),
				nil,
				"find-by-public-id-not-found",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find item",
			err,
			"find-by-public-id-error",
		)
	}
	return entity.EtoD(), nil
}

// FindByConversationID retrieves all items for a conversation.
func (r *ItemRepository) FindByConversationID(ctx context.Context, conversationID uint) ([]*domain.Item, error) {
	var entities []dbschema.ConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("sequence ASC").
		Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find items",
			err,
			"find-by-conversation-id-error",
		)
	}

	items := make([]*domain.Item, len(entities))
	for i := range entities {
		items[i] = entities[i].EtoD()
	}
	return items, nil
}

// Search performs a text search on conversation items.
func (r *ItemRepository) Search(ctx context.Context, conversationID uint, query string) ([]*domain.Item, error) {
	var entities []dbschema.ConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND legacy_content::text ILIKE ?", conversationID, "%"+query+"%").
		Order("sequence ASC").
		Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to search items",
			err,
			"search-items-error",
		)
	}

	items := make([]*domain.Item, len(entities))
	for i := range entities {
		items[i] = entities[i].EtoD()
	}
	return items, nil
}

// FindByFilter retrieves items matching the filter criteria.
func (r *ItemRepository) FindByFilter(ctx context.Context, filter domain.ItemFilter, pagination *query.Pagination) ([]*domain.Item, error) {
	query := r.db.WithContext(ctx).Model(&dbschema.ConversationItem{})

	if filter.ID != nil {
		query = query.Where("id = ?", *filter.ID)
	}
	if filter.PublicID != nil {
		query = query.Where("public_id = ?", *filter.PublicID)
	}
	if filter.CallID != nil {
		query = query.Where("call_id = ?", *filter.CallID)
	}
	if filter.ConversationID != nil {
		query = query.Where("conversation_id = ?", *filter.ConversationID)
	}
	if filter.Role != nil {
		query = query.Where("role = ?", string(*filter.Role))
	}
	if filter.ResponseID != nil {
		query = query.Where("response_id = ?", *filter.ResponseID)
	}
	if filter.Branch != nil {
		query = query.Where("branch = ?", *filter.Branch)
	}

	query = query.Order("sequence ASC")

	if pagination != nil {
		if pagination.Offset != nil {
			query = query.Offset(*pagination.Offset)
		}
		if pagination.Limit != nil {
			query = query.Limit(*pagination.Limit)
		}
	}

	var entities []dbschema.ConversationItem
	if err := query.Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find items",
			err,
			"find-by-filter-error",
		)
	}

	items := make([]*domain.Item, len(entities))
	for i := range entities {
		items[i] = entities[i].EtoD()
	}
	return items, nil
}

// Count returns the count of items matching the filter.
func (r *ItemRepository) Count(ctx context.Context, filter domain.ItemFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&dbschema.ConversationItem{})

	if filter.ID != nil {
		query = query.Where("id = ?", *filter.ID)
	}
	if filter.PublicID != nil {
		query = query.Where("public_id = ?", *filter.PublicID)
	}
	if filter.CallID != nil {
		query = query.Where("call_id = ?", *filter.CallID)
	}
	if filter.ConversationID != nil {
		query = query.Where("conversation_id = ?", *filter.ConversationID)
	}
	if filter.Role != nil {
		query = query.Where("role = ?", string(*filter.Role))
	}
	if filter.ResponseID != nil {
		query = query.Where("response_id = ?", *filter.ResponseID)
	}
	if filter.Branch != nil {
		query = query.Where("branch = ?", *filter.Branch)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count items",
			err,
			"count-error",
		)
	}
	return count, nil
}

// ListByConversationID returns items ordered by sequence.
func (r *ItemRepository) ListByConversationID(ctx context.Context, conversationID uint) ([]domain.Item, error) {
	var entities []dbschema.ConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("sequence ASC").
		Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list conversation items",
			err,
			"8m9l0k1j-2b3c-4d5e-6f7a-8b9c0d1e2f3a",
		)
	}

	items := make([]domain.Item, 0, len(entities))
	for _, entity := range entities {
		items = append(items, *entity.EtoD())
	}
	return items, nil
}

// Update updates an existing item.
func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) error {
	entity := dbschema.NewSchemaConversationItem(item)
	if err := r.db.WithContext(ctx).Save(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update item",
			err,
			"update-error",
		)
	}
	return nil
}

// Delete removes an item by ID.
func (r *ItemRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&dbschema.ConversationItem{}, id).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete item",
			err,
			"delete-error",
		)
	}
	return nil
}

// CountByConversation counts items in a conversation.
func (r *ItemRepository) CountByConversation(ctx context.Context, conversationID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&dbschema.ConversationItem{}).
		Where("conversation_id = ?", conversationID).
		Count(&count).Error; err != nil {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count items",
			err,
			"count-by-conversation-error",
		)
	}
	return count, nil
}

// ExistsByIDAndConversation checks if an item exists in a conversation.
func (r *ItemRepository) ExistsByIDAndConversation(ctx context.Context, itemID uint, conversationID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&dbschema.ConversationItem{}).
		Where("id = ? AND conversation_id = ?", itemID, conversationID).
		Count(&count).Error; err != nil {
		return false, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to check item existence",
			err,
			"exists-by-id-error",
		)
	}
	return count > 0, nil
}
