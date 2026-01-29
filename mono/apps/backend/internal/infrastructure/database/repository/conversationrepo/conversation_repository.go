package conversationrepo

import (
	"context"
	"time"

	"gorm.io/gen"
	"gorm.io/gorm/clause"

	"jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/infrastructure/database/gormgen"
	"jan-server/mono/apps/backend/internal/infrastructure/database/transaction"
	"jan-server/mono/apps/backend/internal/utils/functional"
	"jan-server/mono/apps/backend/internal/utils/idgen"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

type ConversationGormRepository struct {
	db *transaction.Database
}

var _ conversation.ConversationRepository = (*ConversationGormRepository)(nil)

func NewConversationGormRepository(db *transaction.Database) conversation.ConversationRepository {
	return &ConversationGormRepository{db}
}

// Create implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) Create(ctx context.Context, conv *conversation.Conversation) error {
	model := dbschema.NewSchemaConversation(conv)
	if err := repo.db.GetQuery(ctx).Conversation.WithContext(ctx).Create(model); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create conversation")
	}
	// Update the domain object with generated ID and timestamps
	conv.ID = model.ID
	conv.CreatedAt = model.CreatedAt
	conv.UpdatedAt = model.UpdatedAt
	return nil
}

// FindByFilter implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) FindByFilter(ctx context.Context, filter conversation.ConversationFilter, pagination *query.Pagination) ([]*conversation.Conversation, error) {
	q := repo.db.GetQuery(ctx)
	sql := q.Conversation.WithContext(ctx)
	sql = repo.applyFilter(q, sql, filter)
	sql = repo.applyPagination(q, sql, pagination)

	rows, err := sql.Find()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find conversations")
	}

	result := functional.Map(rows, func(item *dbschema.Conversation) *conversation.Conversation {
		return item.EtoD()
	})
	return result, nil
}

// Count implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) Count(ctx context.Context, filter conversation.ConversationFilter) (int64, error) {
	q := repo.db.GetQuery(ctx)
	sql := q.Conversation.WithContext(ctx)
	sql = repo.applyFilter(q, sql, filter)
	return sql.Count()
}

// FindByID implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) FindByID(ctx context.Context, id uint) (*conversation.Conversation, error) {
	q := repo.db.GetQuery(ctx)
	sql := q.Conversation.WithContext(ctx)
	sql = repo.applyFilter(q, sql, conversation.ConversationFilter{ID: &id})
	result, err := sql.First()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find conversation by ID")
	}
	return result.EtoD(), nil
}

// FindByPublicID implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) FindByPublicID(ctx context.Context, publicID string) (*conversation.Conversation, error) {
	q := repo.db.GetQuery(ctx)
	sql := q.Conversation.WithContext(ctx)
	sql = repo.applyFilter(q, sql, conversation.ConversationFilter{PublicID: &publicID})
	result, err := sql.First()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find conversation by public ID")
	}
	return result.EtoD(), nil
}

// Update implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) Update(ctx context.Context, conv *conversation.Conversation) error {
	model := dbschema.NewSchemaConversation(conv)
	q := repo.db.GetQuery(ctx)

	// Use Save to update all fields
	if err := q.Conversation.WithContext(ctx).Where(q.Conversation.ID.Eq(conv.ID)).Save(model); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to update conversation")
	}

	// Update timestamps
	conv.UpdatedAt = model.UpdatedAt
	return nil
}

// Delete implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) Delete(ctx context.Context, id uint) error {
	q := repo.db.GetQuery(ctx)
	_, err := q.Conversation.WithContext(ctx).Where(q.Conversation.ID.Eq(id)).Delete()
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to delete conversation")
	}
	return nil
}

// DeleteAllByUserID implements conversation.ConversationRepository.
// It deletes all conversations for a specific user and returns the count of deleted conversations.
func (repo *ConversationGormRepository) DeleteAllByUserID(ctx context.Context, userID string) (int64, error) {
	db := repo.db.GetTx(ctx)
	result := db.Where("user_id = ?", userID).Delete(&dbschema.Conversation{})
	if result.Error != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerRepository, result.Error, "failed to delete all conversations for user")
	}
	return result.RowsAffected, nil
}

// AddItem implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) AddItem(ctx context.Context, conversationID uint, item *conversation.Item) error {
	// Verify conversation exists
	_, err := repo.FindByID(ctx, conversationID)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "conversation not found")
	}

	// Set conversation ID
	item.ConversationID = conversationID

	// Create the item
	model := dbschema.NewSchemaConversationItem(item)
	q := repo.db.GetQuery(ctx)

	if err := q.ConversationItem.WithContext(ctx).Create(model); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create conversation item")
	}

	// Update the domain object with generated ID
	item.ID = model.ID
	item.CreatedAt = model.CreatedAt

	return nil
}

// SearchItems implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) SearchItems(ctx context.Context, conversationID uint, searchQuery string) ([]*conversation.Item, error) {
	// For now, this is a simple implementation
	// In production, you'd want to use full-text search or a search engine like Elasticsearch
	q := repo.db.GetQuery(ctx)
	sql := q.ConversationItem.WithContext(ctx)
	sql = repo.applyItemFilter(q, sql, conversation.ItemFilter{
		ConversationID: &conversationID,
	})

	// Search in content JSON field (PostgreSQL JSONB search)
	// This is a basic implementation - enhance based on your database capabilities
	// Note: For proper JSON search in PostgreSQL, you might need raw SQL or custom query
	rows, err := sql.Find()

	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to search items")
	}

	result := functional.Map(rows, func(item *dbschema.ConversationItem) *conversation.Item {
		return item.EtoD()
	})

	// TODO: Implement proper full-text search filtering
	// For now, returning all items in the conversation
	return result, nil
}

// BulkAddItems implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) BulkAddItems(ctx context.Context, conversationID uint, items []*conversation.Item) error {
	if len(items) == 0 {
		return nil
	}

	// Verify conversation exists
	_, err := repo.FindByID(ctx, conversationID)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "conversation not found")
	}

	// Set conversation ID for all items
	for _, item := range items {
		item.ConversationID = conversationID
	}

	// Convert to schema models
	models := functional.Map(items, func(item *conversation.Item) *dbschema.ConversationItem {
		return dbschema.NewSchemaConversationItem(item)
	})

	// Bulk insert with manual batching to ensure ID population
	q := repo.db.GetQuery(ctx)
	batchSize := 100

	// Process in batches
	for i := 0; i < len(models); i += batchSize {
		end := i + batchSize
		if end > len(models) {
			end = len(models)
		}

		batch := models[i:end]
		if err := q.ConversationItem.WithContext(ctx).Create(batch...); err != nil {
			return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to bulk create items")
		}

		// Update domain objects with generated IDs for this batch
		for j, model := range batch {
			items[i+j].ID = model.ID
			items[i+j].CreatedAt = model.CreatedAt
		}
	}

	return nil
}

// GetItemByID implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) GetItemByID(ctx context.Context, conversationID uint, itemID uint) (*conversation.Item, error) {
	q := repo.db.GetQuery(ctx)
	sql := q.ConversationItem.WithContext(ctx)
	sql = repo.applyItemFilter(q, sql, conversation.ItemFilter{
		ID:             &itemID,
		ConversationID: &conversationID,
	})
	result, err := sql.First()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find item by ID")
	}
	return result.EtoD(), nil
}

// GetItemByPublicID implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) GetItemByPublicID(ctx context.Context, conversationID uint, publicID string) (*conversation.Item, error) {
	q := repo.db.GetQuery(ctx)
	sql := q.ConversationItem.WithContext(ctx)
	sql = repo.applyItemFilter(q, sql, conversation.ItemFilter{
		PublicID:       &publicID,
		ConversationID: &conversationID,
	})
	result, err := sql.First()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find item by public ID")
	}
	return result.EtoD(), nil
}

// GetItemByCallID implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) GetItemByCallID(ctx context.Context, conversationID uint, callID string) (*conversation.Item, error) {
	q := repo.db.GetQuery(ctx)
	// Use raw SQL since gormgen may not have the call_id field
	var result dbschema.ConversationItem
	err := q.ConversationItem.WithContext(ctx).UnderlyingDB().
		Where("conversation_id = ? AND call_id = ?", conversationID, callID).
		First(&result).Error
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find item by call ID")
	}
	return result.EtoD(), nil
}

// GetItemByCallIDAndType implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) GetItemByCallIDAndType(ctx context.Context, conversationID uint, callID string, itemType conversation.ItemType) (*conversation.Item, error) {
	q := repo.db.GetQuery(ctx)
	// Use raw SQL since gormgen may not have the call_id field
	var result dbschema.ConversationItem
	err := q.ConversationItem.WithContext(ctx).UnderlyingDB().
		Where("conversation_id = ? AND call_id = ? AND type = ?", conversationID, callID, string(itemType)).
		First(&result).Error
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find item by call ID and type")
	}
	return result.EtoD(), nil
}

// UpdateItem implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) UpdateItem(ctx context.Context, conversationID uint, item *conversation.Item) error {
	q := repo.db.GetQuery(ctx)
	entity := dbschema.NewSchemaConversationItem(item)
	
	_, err := q.ConversationItem.WithContext(ctx).
		Where(q.ConversationItem.ID.Eq(item.ID)).
		Where(q.ConversationItem.ConversationID.Eq(conversationID)).
		Updates(entity)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to update item")
	}
	return nil
}

// DeleteItem implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) DeleteItem(ctx context.Context, conversationID uint, itemID uint) error {
	q := repo.db.GetQuery(ctx)
	sql := q.ConversationItem.WithContext(ctx)
	sql = repo.applyItemFilter(q, sql, conversation.ItemFilter{
		ID:             &itemID,
		ConversationID: &conversationID,
	})
	_, err := sql.Delete()
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to delete item")
	}
	return nil
}

// CountItems implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) CountItems(ctx context.Context, conversationID uint, branchName string) (int, error) {
	q := repo.db.GetQuery(ctx)
	sql := q.ConversationItem.WithContext(ctx)
	
	// Apply filter with branch name for proper per-branch counting
	filter := conversation.ItemFilter{
		ConversationID: &conversationID,
	}
	if branchName != "" {
		filter.Branch = &branchName
	}
	sql = repo.applyItemFilter(q, sql, filter)

	count, err := sql.Count()
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to count items")
	}

	return int(count), nil
}

// Branch operations
// CreateBranch implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) CreateBranch(ctx context.Context, conversationID uint, branchName string, metadata *conversation.BranchMetadata) error {
	// Verify conversation exists
	_, err := repo.FindByID(ctx, conversationID)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "conversation not found")
	}

	// Create branch in database
	branch := dbschema.NewSchemaConversationBranch(conversationID, *metadata)
	q := repo.db.GetQuery(ctx)
	if err := q.ConversationBranch.WithContext(ctx).Create(branch); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create branch")
	}
	return nil
}

// GetBranch implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) GetBranch(ctx context.Context, conversationID uint, branchName string) (*conversation.BranchMetadata, error) {
	q := repo.db.GetQuery(ctx)
	branch, err := q.ConversationBranch.WithContext(ctx).
		Where(q.ConversationBranch.ConversationID.Eq(conversationID)).
		Where(q.ConversationBranch.Name.Eq(branchName)).
		First()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "branch not found")
	}
	result := branch.EtoD()
	return &result, nil
}

// ListBranches implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) ListBranches(ctx context.Context, conversationID uint) ([]*conversation.BranchMetadata, error) {
	q := repo.db.GetQuery(ctx)
	branches, err := q.ConversationBranch.WithContext(ctx).
		Where(q.ConversationBranch.ConversationID.Eq(conversationID)).
		Order(q.ConversationBranch.CreatedAt.Asc()).
		Find()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to list branches")
	}

	result := make([]*conversation.BranchMetadata, len(branches))
	for i, branch := range branches {
		meta := branch.EtoD()
		result[i] = &meta
	}
	return result, nil
}

// DeleteBranch implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) DeleteBranch(ctx context.Context, conversationID uint, branchName string) error {
	// Don't allow deleting MAIN branch
	if branchName == "MAIN" {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeValidation, "cannot delete MAIN branch", nil, "e7f8a9b0-c1d2-4e3f-4a5b-6c7d8e9f0a1b")
	}

	q := repo.db.GetQuery(ctx)
	
	// Delete all items in this branch first
	_, err := q.ConversationItem.WithContext(ctx).
		Where(q.ConversationItem.ConversationID.Eq(conversationID)).
		Where(q.ConversationItem.Branch.Eq(branchName)).
		Delete()
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to delete branch items")
	}

	// Delete the branch metadata
	_, err = q.ConversationBranch.WithContext(ctx).
		Where(q.ConversationBranch.ConversationID.Eq(conversationID)).
		Where(q.ConversationBranch.Name.Eq(branchName)).
		Delete()
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to delete branch")
	}
	return nil
}

// SetActiveBranch implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) SetActiveBranch(ctx context.Context, conversationID uint, branchName string) error {
	q := repo.db.GetQuery(ctx)
	_, err := q.Conversation.WithContext(ctx).
		Where(q.Conversation.ID.Eq(conversationID)).
		Update(q.Conversation.ActiveBranch, branchName)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to set active branch")
	}
	return nil
}

// Branch item operations
// AddItemToBranch implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) AddItemToBranch(ctx context.Context, conversationID uint, branchName string, item *conversation.Item) error {
	// Set branch on item
	item.Branch = branchName
	if branchName == "" {
		item.Branch = "MAIN"
	}
	return repo.AddItem(ctx, conversationID, item)
}

// GetBranchItems implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) GetBranchItems(ctx context.Context, conversationID uint, branchName string, pagination *query.Pagination) ([]*conversation.Item, error) {
	// Default to MAIN branch if empty
	if branchName == "" {
		branchName = "MAIN"
	}

	q := repo.db.GetQuery(ctx)
	sql := q.ConversationItem.WithContext(ctx)
	
	// Apply filter with branch name
	filter := conversation.ItemFilter{
		ConversationID: &conversationID,
		Branch:         &branchName,
	}
	sql = repo.applyItemFilter(q, sql, filter)
	sql = repo.applyItemPagination(q, sql, pagination)

	rows, err := sql.Find()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to get branch items")
	}

	return functional.Map(rows, func(item *dbschema.ConversationItem) *conversation.Item {
		return item.EtoD()
	}), nil
}

// applyItemPagination applies pagination to item queries
func (repo *ConversationGormRepository) applyItemPagination(q *gormgen.Query, sql gormgen.IConversationItemDo, p *query.Pagination) gormgen.IConversationItemDo {
	if p != nil {
		// Apply cursor-based pagination
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(q.ConversationItem.ID.Lt(*p.After))
			} else {
				sql = sql.Where(q.ConversationItem.ID.Gt(*p.After))
			}
		}

		// Apply ordering (default to ascending by ID)
		if p.Order == "desc" {
			sql = sql.Order(q.ConversationItem.ID.Desc())
		} else {
			sql = sql.Order(q.ConversationItem.ID.Asc())
		}

		// Apply limit
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
	} else {
		// Default ordering when no pagination specified
		sql = sql.Order(q.ConversationItem.ID.Asc())
	}
	return sql
}

// BulkAddItemsToBranch implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) BulkAddItemsToBranch(ctx context.Context, conversationID uint, branchName string, items []*conversation.Item) error {
	if len(items) == 0 {
		return nil
	}

	// Default to MAIN if empty
	if branchName == "" {
		branchName = "MAIN"
	}

	// Set branch on all items
	for _, item := range items {
		item.Branch = branchName
	}

	// Use existing BulkAddItems - it already handles the conversion
	return repo.BulkAddItems(ctx, conversationID, items)
}

// ForkBranch implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) ForkBranch(ctx context.Context, conversationID uint, sourceBranch, newBranch string, fromItemID string, description *string) error {
	// Get source branch items up to the fork point
	sourceItems, err := repo.GetBranchItems(ctx, conversationID, sourceBranch, nil)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to get source branch items")
	}

	// Find fork point
	forkIndex := -1
	for i, item := range sourceItems {
		if item.PublicID == fromItemID {
			forkIndex = i
			break
		}
	}

	if forkIndex == -1 && fromItemID != "" {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "fork item not found", nil, "d2e3f4a5-b6c7-4d8e-9f0a-1b2c3d4e5f6a")
	}

	// Create branch metadata
	now := time.Now()
	metadata := &conversation.BranchMetadata{
		Name:             newBranch,
		Description:      description,
		ParentBranch:     &sourceBranch,
		ForkedAt:         &now,
		ForkedFromItemID: &fromItemID,
		ItemCount:        0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := repo.CreateBranch(ctx, conversationID, newBranch, metadata); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create branch")
	}

	// Copy items up to fork point to new branch
	if forkIndex >= 0 {
		itemsToCopy := make([]*conversation.Item, forkIndex+1)
		for i := 0; i <= forkIndex; i++ {
			itemCopy := *sourceItems[i]
			itemCopy.ID = 0 // Reset ID for new insert
			// Generate new PublicID for the copied item (PublicID has unique constraint)
			newPublicID, err := idgen.GenerateSecureID("msg", 16)
			if err != nil {
				return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to generate item ID")
			}
			itemCopy.PublicID = newPublicID
			itemCopy.Branch = newBranch
			itemCopy.SequenceNumber = i + 1
			itemsToCopy[i] = &itemCopy
		}

		if err := repo.BulkAddItemsToBranch(ctx, conversationID, newBranch, itemsToCopy); err != nil {
			return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to copy items to new branch")
		}

		// Update branch item count
		q := repo.db.GetQuery(ctx)
		_, err = q.ConversationBranch.WithContext(ctx).
			Where(q.ConversationBranch.ConversationID.Eq(conversationID)).
			Where(q.ConversationBranch.Name.Eq(newBranch)).
			Update(q.ConversationBranch.ItemCount, len(itemsToCopy))
		if err != nil {
			return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to update branch item count")
		}
	}

	return nil
}

// SwapBranchToMain implements conversation.ConversationRepository.
// It promotes the given branch to become MAIN by:
// 1. Creating a backup for the old MAIN items (if they exist)
// 2. Renaming the given branch to MAIN
// 3. Setting MAIN as the active branch
func (repo *ConversationGormRepository) SwapBranchToMain(ctx context.Context, conversationID uint, branchToPromote string) (string, error) {
	if branchToPromote == "MAIN" {
		// Already MAIN, nothing to do
		return "", nil
	}

	q := repo.db.GetQuery(ctx)

	// Generate backup name for old MAIN
	oldMainBackupName := "MAIN_" + time.Now().Format("20060102150405")

	// Check if MAIN branch record exists in the database
	mainBranch, err := q.ConversationBranch.WithContext(ctx).
		Where(q.ConversationBranch.ConversationID.Eq(conversationID)).
		Where(q.ConversationBranch.Name.Eq("MAIN")).
		First()

	if err == nil && mainBranch != nil {
		// MAIN branch record exists - rename it to backup
		_, err = q.ConversationBranch.WithContext(ctx).
			Where(q.ConversationBranch.ConversationID.Eq(conversationID)).
			Where(q.ConversationBranch.Name.Eq("MAIN")).
			Update(q.ConversationBranch.Name, oldMainBackupName)
		if err != nil {
			return "", platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to rename MAIN branch to backup")
		}
	} else {
		// MAIN branch record doesn't exist - create backup branch for existing MAIN items
		// Count existing MAIN items
		count, err := q.ConversationItem.WithContext(ctx).
			Where(q.ConversationItem.ConversationID.Eq(conversationID)).
			Where(q.ConversationItem.Branch.Eq("MAIN")).
			Count()
		if err != nil {
			return "", platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to count MAIN items")
		}

		if count > 0 {
			// Create a branch record for the backup
			now := time.Now()
			description := "Backup of original MAIN branch"
			backupBranch := &dbschema.ConversationBranch{
				ConversationID: conversationID,
				Name:           oldMainBackupName,
				Description:    &description,
				ItemCount:      int(count),
			}
			backupBranch.CreatedAt = now
			backupBranch.UpdatedAt = now

			if err := q.ConversationBranch.WithContext(ctx).Create(backupBranch); err != nil {
				return "", platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create backup branch")
			}
		} else {
			// No MAIN items exist, no backup needed
			oldMainBackupName = ""
		}
	}

	// Update all items in old MAIN to use backup name (if backup was created)
	if oldMainBackupName != "" {
		_, err = q.ConversationItem.WithContext(ctx).
			Where(q.ConversationItem.ConversationID.Eq(conversationID)).
			Where(q.ConversationItem.Branch.Eq("MAIN")).
			Update(q.ConversationItem.Branch, oldMainBackupName)
		if err != nil {
			return "", platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to update MAIN items to backup branch")
		}
	}

	// Rename the promoted branch metadata to MAIN
	_, err = q.ConversationBranch.WithContext(ctx).
		Where(q.ConversationBranch.ConversationID.Eq(conversationID)).
		Where(q.ConversationBranch.Name.Eq(branchToPromote)).
		Update(q.ConversationBranch.Name, "MAIN")
	if err != nil {
		return "", platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to rename branch to MAIN")
	}

	// Update all items in promoted branch to use MAIN
	_, err = q.ConversationItem.WithContext(ctx).
		Where(q.ConversationItem.ConversationID.Eq(conversationID)).
		Where(q.ConversationItem.Branch.Eq(branchToPromote)).
		Update(q.ConversationItem.Branch, "MAIN")
	if err != nil {
		return "", platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to update promoted items to MAIN")
	}

	// Set MAIN as active branch
	_, err = q.Conversation.WithContext(ctx).
		Where(q.Conversation.ID.Eq(conversationID)).
		Update(q.Conversation.ActiveBranch, "MAIN")
	if err != nil {
		return "", platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to set active branch to MAIN")
	}

	return oldMainBackupName, nil
}

// Item rating operations
// RateItem implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) RateItem(ctx context.Context, conversationID uint, itemID string, rating conversation.ItemRating, comment *string) error {
	q := repo.db.GetQuery(ctx)
	ratingStr := string(rating)
	now := time.Now()

	updates := map[string]interface{}{
		"rating":   ratingStr,
		"rated_at": now,
	}
	if comment != nil {
		updates["rating_comment"] = *comment
	}

	_, err := q.ConversationItem.WithContext(ctx).
		Where(q.ConversationItem.ConversationID.Eq(conversationID)).
		Where(q.ConversationItem.PublicID.Eq(itemID)).
		Updates(updates)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to rate item")
	}
	return nil
}

// GetItemRating implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) GetItemRating(ctx context.Context, conversationID uint, itemID string) (*conversation.ItemRating, error) {
	q := repo.db.GetQuery(ctx)
	item, err := q.ConversationItem.WithContext(ctx).
		Where(q.ConversationItem.ConversationID.Eq(conversationID)).
		Where(q.ConversationItem.PublicID.Eq(itemID)).
		First()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "item not found")
	}
	if item.Rating == nil {
		return nil, nil
	}
	rating := conversation.ItemRating(*item.Rating)
	return &rating, nil
}

// RemoveItemRating implements conversation.ConversationRepository.
func (repo *ConversationGormRepository) RemoveItemRating(ctx context.Context, conversationID uint, itemID string) error {
	q := repo.db.GetQuery(ctx)
	updates := map[string]interface{}{
		"rating":         nil,
		"rated_at":       nil,
		"rating_comment": nil,
	}
	_, err := q.ConversationItem.WithContext(ctx).
		Where(q.ConversationItem.ConversationID.Eq(conversationID)).
		Where(q.ConversationItem.PublicID.Eq(itemID)).
		Updates(updates)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to remove item rating")
	}
	return nil
}

// applyFilter applies filter conditions to the query
func (repo *ConversationGormRepository) applyFilter(q *gormgen.Query, sql gormgen.IConversationDo, filter conversation.ConversationFilter) gormgen.IConversationDo {
	if filter.ID != nil {
		sql = sql.Where(q.Conversation.ID.Eq(*filter.ID))
	}
	if filter.PublicID != nil {
		sql = sql.Where(q.Conversation.PublicID.Eq(*filter.PublicID))
	}
	if filter.UserID != nil {
		// Use Clauses for raw SQL condition since GORM Gen may have been generated with uint
		sql = sql.Where(gen.Cond(clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "user_id = ?", Vars: []interface{}{*filter.UserID}}}})...)
	}
	if filter.Referrer != nil && *filter.Referrer != "" {
		sql = sql.Where(q.Conversation.Referrer.Eq(*filter.Referrer))
	}
	return sql
}

// applyItemFilter applies filter conditions to the conversation item query
func (repo *ConversationGormRepository) applyItemFilter(q *gormgen.Query, sql gormgen.IConversationItemDo, filter conversation.ItemFilter) gormgen.IConversationItemDo {
	if filter.ID != nil {
		sql = sql.Where(q.ConversationItem.ID.Eq(*filter.ID))
	}
	if filter.PublicID != nil {
		sql = sql.Where(q.ConversationItem.PublicID.Eq(*filter.PublicID))
	}
	// Note: CallID filtering is done via raw SQL in GetItemByCallID since gormgen may not have the field
	if filter.ConversationID != nil {
		sql = sql.Where(q.ConversationItem.ConversationID.Eq(*filter.ConversationID))
	}
	if filter.Role != nil {
		roleStr := string(*filter.Role)
		sql = sql.Where(q.ConversationItem.Role.Eq(roleStr))
	}
	if filter.ResponseID != nil {
		sql = sql.Where(q.ConversationItem.ResponseID.Eq(*filter.ResponseID))
	}
	// Filter by branch name
	if filter.Branch != nil && *filter.Branch != "" {
		sql = sql.Where(q.ConversationItem.Branch.Eq(*filter.Branch))
	}
	return sql
}

// applyPagination applies pagination to the query
func (repo *ConversationGormRepository) applyPagination(q *gormgen.Query, sql gormgen.IConversationDo, p *query.Pagination) gormgen.IConversationDo {
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(q.Conversation.ID.Lt(*p.After))
			} else {
				sql = sql.Where(q.Conversation.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(q.Conversation.UpdatedAt.Desc())
		} else {
			sql = sql.Order(q.Conversation.UpdatedAt.Asc())
		}
	}
	return sql
}
