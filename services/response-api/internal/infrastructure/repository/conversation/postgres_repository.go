package conversation

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	domain "jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/infrastructure/database/entities"
	"jan-server/services/response-api/internal/utils/platformerrors"
)

// Repository persists conversation metadata.
type Repository struct {
	db *gorm.DB
}

// NewRepository builds a conversation repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts the conversation record.
func (r *Repository) Create(ctx context.Context, conv *domain.Conversation) error {
	entity := entities.NewSchemaConversation(conv)

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create conversation",
			err,
			"2g3f4e5d-6b7c-8d9e-0f1a-2b3c4d5e6f7a",
		)
	}

	conv.ID = entity.ID
	conv.CreatedAt = entity.CreatedAt
	conv.UpdatedAt = entity.UpdatedAt
	return nil
}

// FindByPublicID fetches a conversation by its public ID.
func (r *Repository) FindByPublicID(ctx context.Context, publicID string) (*domain.Conversation, error) {
	var entity entities.Conversation
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Branches").
		Where("public_id = ?", publicID).
		First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("conversation not found: %s", publicID),
				nil,
				"3h4g5f6e-7c8d-9e0f-1a2b-3c4d5e6f7a8b",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to fetch conversation",
			err,
			"4i5h6g7f-8d9e-0f1a-2b3c-4d5e6f7a8b9c",
		)
	}

	return entity.EtoD(), nil
}

// FindByID fetches a conversation by its internal ID.
func (r *Repository) FindByID(ctx context.Context, id uint) (*domain.Conversation, error) {
	var entity entities.Conversation
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Branches").
		First(&entity, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("conversation not found: %d", id),
				nil,
				"find-by-id-not-found",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to fetch conversation",
			err,
			"find-by-id-db-error",
		)
	}
	return entity.EtoD(), nil
}

// FindByFilter fetches conversations matching the filter criteria.
func (r *Repository) FindByFilter(ctx context.Context, filter domain.ConversationFilter, pagination *domain.Pagination) ([]*domain.Conversation, error) {
	query := r.db.WithContext(ctx).Model(&entities.Conversation{})

	if filter.ID != nil {
		query = query.Where("id = ?", *filter.ID)
	}
	if filter.PublicID != nil {
		query = query.Where("public_id = ?", *filter.PublicID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.ProjectID != nil {
		query = query.Where("project_id = ?", *filter.ProjectID)
	}
	if filter.Referrer != nil {
		query = query.Where("referrer = ?", *filter.Referrer)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if pagination != nil {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
	}

	var entities []entities.Conversation
	if err := query.Preload("Items").Preload("Branches").Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find conversations",
			err,
			"find-by-filter-error",
		)
	}

	result := make([]*domain.Conversation, len(entities))
	for i := range entities {
		result[i] = entities[i].EtoD()
	}
	return result, nil
}

// Count returns the count of conversations matching the filter.
func (r *Repository) Count(ctx context.Context, filter domain.ConversationFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&entities.Conversation{})

	if filter.ID != nil {
		query = query.Where("id = ?", *filter.ID)
	}
	if filter.PublicID != nil {
		query = query.Where("public_id = ?", *filter.PublicID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.ProjectID != nil {
		query = query.Where("project_id = ?", *filter.ProjectID)
	}
	if filter.Referrer != nil {
		query = query.Where("referrer = ?", *filter.Referrer)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count conversations",
			err,
			"count-error",
		)
	}
	return count, nil
}

// Update updates a conversation record.
func (r *Repository) Update(ctx context.Context, conv *domain.Conversation) error {
	entity := entities.NewSchemaConversation(conv)
	if err := r.db.WithContext(ctx).Save(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update conversation",
			err,
			"update-error",
		)
	}
	return nil
}

// Delete removes a conversation record.
func (r *Repository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&entities.Conversation{}, id).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete conversation",
			err,
			"delete-error",
		)
	}
	return nil
}

// DeleteAllByUserID removes all conversations for a user.
func (r *Repository) DeleteAllByUserID(ctx context.Context, userID uint) (int64, error) {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.Conversation{})
	if result.Error != nil {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete conversations",
			result.Error,
			"delete-all-by-user-error",
		)
	}
	return result.RowsAffected, nil
}

// AddItem adds a single item to a conversation.
func (r *Repository) AddItem(ctx context.Context, conversationID uint, item *domain.Item) error {
	item.ConversationID = conversationID
	entity := entities.NewSchemaConversationItem(item)
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to add item",
			err,
			"add-item-error",
		)
	}
	item.ID = entity.ID
	return nil
}

// BulkAddItems adds multiple items to a conversation.
func (r *Repository) BulkAddItems(ctx context.Context, conversationID uint, items []*domain.Item) error {
	if len(items) == 0 {
		return nil
	}
	dbItems := make([]*entities.ConversationItem, len(items))
	for i, item := range items {
		item.ConversationID = conversationID
		dbItems[i] = entities.NewSchemaConversationItem(item)
	}
	if err := r.db.WithContext(ctx).Create(&dbItems).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to bulk add items",
			err,
			"bulk-add-items-error",
		)
	}
	for i, entity := range dbItems {
		items[i].ID = entity.ID
	}
	return nil
}

// GetItemByID retrieves an item by its internal ID.
func (r *Repository) GetItemByID(ctx context.Context, conversationID uint, itemID uint) (*domain.Item, error) {
	var entity entities.ConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND id = ?", conversationID, itemID).
		First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("item not found: %d", itemID),
				nil,
				"get-item-by-id-not-found",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get item",
			err,
			"get-item-by-id-error",
		)
	}
	return entity.EtoD(), nil
}

// GetItemByPublicID retrieves an item by its public ID.
func (r *Repository) GetItemByPublicID(ctx context.Context, conversationID uint, publicID string) (*domain.Item, error) {
	var entity entities.ConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND public_id = ?", conversationID, publicID).
		First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("item not found: %s", publicID),
				nil,
				"get-item-by-public-id-not-found",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get item",
			err,
			"get-item-by-public-id-error",
		)
	}
	return entity.EtoD(), nil
}

// GetItemByCallID retrieves an item by its call ID.
func (r *Repository) GetItemByCallID(ctx context.Context, conversationID uint, callID string) (*domain.Item, error) {
	var entity entities.ConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND call_id = ?", conversationID, callID).
		First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("item not found with call_id: %s", callID),
				nil,
				"get-item-by-call-id-not-found",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get item",
			err,
			"get-item-by-call-id-error",
		)
	}
	return entity.EtoD(), nil
}

// GetItemByCallIDAndType retrieves an item by call ID and type.
func (r *Repository) GetItemByCallIDAndType(ctx context.Context, conversationID uint, callID string, itemType domain.ItemType) (*domain.Item, error) {
	var entity entities.ConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND call_id = ? AND type = ?", conversationID, callID, itemType).
		First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("item not found with call_id: %s and type: %s", callID, itemType),
				nil,
				"get-item-by-call-id-type-not-found",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get item",
			err,
			"get-item-by-call-id-type-error",
		)
	}
	return entity.EtoD(), nil
}

// UpdateItem updates a conversation item.
func (r *Repository) UpdateItem(ctx context.Context, conversationID uint, item *domain.Item) error {
	entity := entities.NewSchemaConversationItem(item)
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND id = ?", conversationID, item.ID).
		Save(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update item",
			err,
			"update-item-error",
		)
	}
	return nil
}

// DeleteItem removes an item from a conversation.
func (r *Repository) DeleteItem(ctx context.Context, conversationID uint, itemID uint) error {
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND id = ?", conversationID, itemID).
		Delete(&entities.ConversationItem{}).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete item",
			err,
			"delete-item-error",
		)
	}
	return nil
}

// CountItems counts items in a conversation branch.
func (r *Repository) CountItems(ctx context.Context, conversationID uint, branchName string) (int, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&entities.ConversationItem{}).
		Where("conversation_id = ?", conversationID)
	if branchName != "" {
		query = query.Where("branch = ?", branchName)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count items",
			err,
			"count-items-error",
		)
	}
	return int(count), nil
}

// CreateBranch creates a new branch in a conversation.
func (r *Repository) CreateBranch(ctx context.Context, conversationID uint, branchName string, metadata *domain.BranchMetadata) error {
	entity := &entities.ConversationBranch{
		ConversationID:   conversationID,
		Name:             branchName,
		Description:      metadata.Description,
		ParentBranch:     metadata.ParentBranch,
		ForkedAt:         metadata.ForkedAt,
		ForkedFromItemID: metadata.ForkedFromItemID,
		ItemCount:        metadata.ItemCount,
	}
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create branch",
			err,
			"create-branch-error",
		)
	}
	return nil
}

// GetBranch retrieves branch metadata.
func (r *Repository) GetBranch(ctx context.Context, conversationID uint, branchName string) (*domain.BranchMetadata, error) {
	var entity entities.ConversationBranch
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND name = ?", conversationID, branchName).
		First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("branch not found: %s", branchName),
				nil,
				"get-branch-not-found",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get branch",
			err,
			"get-branch-error",
		)
	}
	return &domain.BranchMetadata{
		Name:             entity.Name,
		Description:      entity.Description,
		ParentBranch:     entity.ParentBranch,
		ForkedAt:         entity.ForkedAt,
		ForkedFromItemID: entity.ForkedFromItemID,
		ItemCount:        entity.ItemCount,
		CreatedAt:        entity.CreatedAt,
		UpdatedAt:        entity.UpdatedAt,
	}, nil
}

// ListBranches lists all branches for a conversation.
func (r *Repository) ListBranches(ctx context.Context, conversationID uint) ([]*domain.BranchMetadata, error) {
	var entities []entities.ConversationBranch
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list branches",
			err,
			"list-branches-error",
		)
	}
	result := make([]*domain.BranchMetadata, len(entities))
	for i, entity := range entities {
		result[i] = &domain.BranchMetadata{
			Name:             entity.Name,
			Description:      entity.Description,
			ParentBranch:     entity.ParentBranch,
			ForkedAt:         entity.ForkedAt,
			ForkedFromItemID: entity.ForkedFromItemID,
			ItemCount:        entity.ItemCount,
			CreatedAt:        entity.CreatedAt,
			UpdatedAt:        entity.UpdatedAt,
		}
	}
	return result, nil
}

// DeleteBranch deletes a branch from a conversation.
func (r *Repository) DeleteBranch(ctx context.Context, conversationID uint, branchName string) error {
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND name = ?", conversationID, branchName).
		Delete(&entities.ConversationBranch{}).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete branch",
			err,
			"delete-branch-error",
		)
	}
	return nil
}

// SetActiveBranch sets the active branch for a conversation.
func (r *Repository) SetActiveBranch(ctx context.Context, conversationID uint, branchName string) error {
	if err := r.db.WithContext(ctx).Model(&entities.Conversation{}).
		Where("id = ?", conversationID).
		Update("active_branch", branchName).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to set active branch",
			err,
			"set-active-branch-error",
		)
	}
	return nil
}

// AddItemToBranch adds an item to a specific branch.
func (r *Repository) AddItemToBranch(ctx context.Context, conversationID uint, branchName string, item *domain.Item) error {
	item.ConversationID = conversationID
	item.Branch = branchName
	return r.AddItem(ctx, conversationID, item)
}

// GetBranchItems retrieves items from a specific branch.
func (r *Repository) GetBranchItems(ctx context.Context, conversationID uint, branchName string, pagination *domain.Pagination) ([]*domain.Item, error) {
	query := r.db.WithContext(ctx).
		Where("conversation_id = ? AND branch = ?", conversationID, branchName).
		Order("sequence ASC")

	if pagination != nil {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
	}

	var entities []entities.ConversationItem
	if err := query.Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get branch items",
			err,
			"get-branch-items-error",
		)
	}

	result := make([]*domain.Item, len(entities))
	for i := range entities {
		result[i] = entities[i].EtoD()
	}
	return result, nil
}

// BulkAddItemsToBranch adds multiple items to a branch.
func (r *Repository) BulkAddItemsToBranch(ctx context.Context, conversationID uint, branchName string, items []*domain.Item) error {
	for _, item := range items {
		item.Branch = branchName
	}
	return r.BulkAddItems(ctx, conversationID, items)
}

// ForkBranch creates a new branch from an existing one at a specific item.
func (r *Repository) ForkBranch(ctx context.Context, conversationID uint, sourceBranch, newBranch string, fromItemID string, description *string) error {
	// TODO: Implement fork logic - copy items from source to new branch up to fromItemID
	return platformerrors.NewError(
		ctx,
		platformerrors.LayerRepository,
		platformerrors.ErrorTypeNotImplemented,
		"fork branch not implemented",
		nil,
		"fork-branch-not-implemented",
	)
}

// SwapBranchToMain swaps a branch with MAIN.
func (r *Repository) SwapBranchToMain(ctx context.Context, conversationID uint, branchToPromote string) (string, error) {
	// TODO: Implement swap logic
	return "", platformerrors.NewError(
		ctx,
		platformerrors.LayerRepository,
		platformerrors.ErrorTypeNotImplemented,
		"swap branch to main not implemented",
		nil,
		"swap-branch-not-implemented",
	)
}

// RateItem adds a rating to an item.
func (r *Repository) RateItem(ctx context.Context, conversationID uint, itemID string, rating domain.ItemRating, comment *string) error {
	updates := map[string]interface{}{
		"rating":         string(rating),
		"rating_comment": comment,
	}
	if err := r.db.WithContext(ctx).Model(&entities.ConversationItem{}).
		Where("conversation_id = ? AND public_id = ?", conversationID, itemID).
		Updates(updates).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to rate item",
			err,
			"rate-item-error",
		)
	}
	return nil
}

// GetItemRating retrieves the rating for an item.
func (r *Repository) GetItemRating(ctx context.Context, conversationID uint, itemID string) (*domain.ItemRating, error) {
	var entity entities.ConversationItem
	if err := r.db.WithContext(ctx).
		Select("rating").
		Where("conversation_id = ? AND public_id = ?", conversationID, itemID).
		First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get item rating",
			err,
			"get-item-rating-error",
		)
	}
	if entity.Rating == nil {
		return nil, nil
	}
	rating := domain.ItemRating(*entity.Rating)
	return &rating, nil
}

// RemoveItemRating removes the rating from an item.
func (r *Repository) RemoveItemRating(ctx context.Context, conversationID uint, itemID string) error {
	updates := map[string]interface{}{
		"rating":         nil,
		"rating_comment": nil,
		"rated_at":       nil,
	}
	if err := r.db.WithContext(ctx).Model(&entities.ConversationItem{}).
		Where("conversation_id = ? AND public_id = ?", conversationID, itemID).
		Updates(updates).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to remove item rating",
			err,
			"remove-item-rating-error",
		)
	}
	return nil
}
