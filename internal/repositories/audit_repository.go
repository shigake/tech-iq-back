package repositories

import (
	"encoding/json"
	"reflect"

	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

// AuditRepository handles audit log database operations
type AuditRepository interface {
	Create(log *models.AuditLog) error
	FindByID(id string) (*models.AuditLog, error)
	FindAll(filter models.AuditLogFilter) (*models.AuditLogResponse, error)
	FindByEntity(entityType, entityID string, page, size int) ([]models.AuditLog, int64, error)
	FindByUser(userID string, page, size int) ([]models.AuditLog, int64, error)
	GetStats(filter models.AuditLogFilter) (*models.AuditStats, error)
	GetDistinctEntityTypes() ([]string, error)
	GetDistinctUsers() ([]models.UserAuditStat, error)

	// Hierarchy-aware queries (respects user access to entities)
	FindAllWithHierarchy(userID string, accessibleTechnicianIDs []string, filter models.AuditLogFilter) (*models.AuditLogResponse, error)
}

type auditRepository struct {
	db *gorm.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

// Create inserts a new audit log entry
func (r *auditRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

// FindByID retrieves an audit log by ID
func (r *auditRepository) FindByID(id string) (*models.AuditLog, error) {
	var log models.AuditLog
	err := r.db.Preload("User").Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// FindAll retrieves audit logs with filters
func (r *auditRepository) FindAll(filter models.AuditLogFilter) (*models.AuditLogResponse, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{})

	// Apply filters
	query = r.applyFilters(query, filter)

	// Count total
	query.Count(&total)

	// Pagination
	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	offset := filter.Page * filter.Size

	// Get logs
	err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(filter.Size).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / filter.Size
	if int(total)%filter.Size > 0 {
		totalPages++
	}

	return &models.AuditLogResponse{
		Items:      logs,
		Total:      total,
		Page:       filter.Page,
		Size:       filter.Size,
		TotalPages: totalPages,
	}, nil
}

// FindAllWithHierarchy retrieves audit logs respecting hierarchy access
func (r *auditRepository) FindAllWithHierarchy(userID string, accessibleTechnicianIDs []string, filter models.AuditLogFilter) (*models.AuditLogResponse, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{})

	// Apply filters
	query = r.applyFilters(query, filter)

	// Apply hierarchy filter - user can see:
	// 1. Logs of entities they have access to (technicians in their hierarchy)
	// 2. Logs of their own actions
	// 3. Non-technician logs if they have appropriate permissions
	if len(accessibleTechnicianIDs) > 0 {
		query = query.Where(
			"(entity_type = 'technician' AND entity_id IN ?) OR user_id = ? OR entity_type != 'technician'",
			accessibleTechnicianIDs, userID,
		)
	} else {
		// User only sees their own actions
		query = query.Where("user_id = ?", userID)
	}

	// Count total
	query.Count(&total)

	// Pagination
	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	offset := filter.Page * filter.Size

	// Get logs
	err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(filter.Size).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / filter.Size
	if int(total)%filter.Size > 0 {
		totalPages++
	}

	return &models.AuditLogResponse{
		Items:      logs,
		Total:      total,
		Page:       filter.Page,
		Size:       filter.Size,
		TotalPages: totalPages,
	}, nil
}

// FindByEntity retrieves audit logs for a specific entity
func (r *auditRepository) FindByEntity(entityType, entityID string, page, size int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{}).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID)

	query.Count(&total)

	offset := page * size
	err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindByUser retrieves audit logs for actions by a specific user
func (r *auditRepository) FindByUser(userID string, page, size int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{}).Where("user_id = ?", userID)

	query.Count(&total)

	offset := page * size
	err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetStats retrieves audit statistics
func (r *auditRepository) GetStats(filter models.AuditLogFilter) (*models.AuditStats, error) {
	var stats models.AuditStats
	var totalLogs int64

	query := r.db.Model(&models.AuditLog{})
	query = r.applyFilters(query, filter)
	query.Count(&totalLogs)
	stats.TotalLogs = totalLogs

	// Logs by action
	var actionStats []struct {
		Action string
		Count  int64
	}
	r.db.Model(&models.AuditLog{}).
		Select("action, count(*) as count").
		Group("action").
		Find(&actionStats)

	stats.LogsByAction = make(map[string]int64)
	for _, s := range actionStats {
		stats.LogsByAction[s.Action] = s.Count
	}

	// Logs by entity type
	var entityStats []struct {
		EntityType string
		Count      int64
	}
	r.db.Model(&models.AuditLog{}).
		Select("entity_type, count(*) as count").
		Group("entity_type").
		Find(&entityStats)

	stats.LogsByEntity = make(map[string]int64)
	for _, s := range entityStats {
		stats.LogsByEntity[s.EntityType] = s.Count
	}

	// Logs by user (top 10)
	var userStats []models.UserAuditStat
	r.db.Model(&models.AuditLog{}).
		Select("user_id, user_name, user_email, count(*) as count").
		Group("user_id, user_name, user_email").
		Order("count DESC").
		Limit(10).
		Find(&userStats)
	stats.LogsByUser = userStats

	// Recent activity (last 10)
	var recentLogs []models.AuditLog
	r.db.Model(&models.AuditLog{}).
		Preload("User").
		Order("created_at DESC").
		Limit(10).
		Find(&recentLogs)
	stats.RecentActivity = recentLogs

	return &stats, nil
}

// GetDistinctEntityTypes returns all distinct entity types in audit logs
func (r *auditRepository) GetDistinctEntityTypes() ([]string, error) {
	var types []string
	err := r.db.Model(&models.AuditLog{}).
		Distinct("entity_type").
		Pluck("entity_type", &types).Error
	return types, err
}

// GetDistinctUsers returns all users with audit log entries
func (r *auditRepository) GetDistinctUsers() ([]models.UserAuditStat, error) {
	var users []models.UserAuditStat
	err := r.db.Model(&models.AuditLog{}).
		Select("user_id, user_name, user_email, count(*) as count").
		Group("user_id, user_name, user_email").
		Order("user_name ASC").
		Find(&users).Error
	return users, err
}

// applyFilters applies common filters to the query
func (r *auditRepository) applyFilters(query *gorm.DB, filter models.AuditLogFilter) *gorm.DB {
	if filter.EntityType != "" {
		query = query.Where("entity_type = ?", filter.EntityType)
	}
	if filter.EntityID != "" {
		query = query.Where("entity_id = ?", filter.EntityID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", filter.EndDate)
	}
	if filter.SearchQuery != "" {
		searchTerm := "%" + filter.SearchQuery + "%"
		query = query.Where(
			"entity_name ILIKE ? OR user_name ILIKE ? OR description ILIKE ?",
			searchTerm, searchTerm, searchTerm,
		)
	}
	return query
}

// ========== Helper Functions for Creating Audit Logs ==========

// AuditService provides helper methods for creating audit logs
type AuditService struct {
	repo AuditRepository
}

// NewAuditService creates a new audit service
func NewAuditService(repo AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// LogCreate logs a create action
func (s *AuditService) LogCreate(userID, userName, userEmail, entityType, entityID, entityName, ipAddress, userAgent string, newValue interface{}) error {
	newValueJSON, _ := json.Marshal(newValue)

	log := &models.AuditLog{
		Action:      models.AuditActionCreate,
		EntityType:  entityType,
		EntityID:    entityID,
		EntityName:  entityName,
		UserID:      userID,
		UserName:    userName,
		UserEmail:   userEmail,
		NewValue:    newValueJSON,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Description: "Registro criado",
	}
	return s.repo.Create(log)
}

// LogUpdate logs an update action
func (s *AuditService) LogUpdate(userID, userName, userEmail, entityType, entityID, entityName, ipAddress, userAgent string, oldValue, newValue interface{}) error {
	oldValueJSON, _ := json.Marshal(oldValue)
	newValueJSON, _ := json.Marshal(newValue)
	changes := s.computeChanges(oldValue, newValue)
	changesJSON, _ := json.Marshal(changes)

	log := &models.AuditLog{
		Action:      models.AuditActionUpdate,
		EntityType:  entityType,
		EntityID:    entityID,
		EntityName:  entityName,
		UserID:      userID,
		UserName:    userName,
		UserEmail:   userEmail,
		OldValue:    oldValueJSON,
		NewValue:    newValueJSON,
		Changes:     changesJSON,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Description: "Registro atualizado",
	}
	return s.repo.Create(log)
}

// LogDelete logs a delete action
func (s *AuditService) LogDelete(userID, userName, userEmail, entityType, entityID, entityName, ipAddress, userAgent string, oldValue interface{}) error {
	oldValueJSON, _ := json.Marshal(oldValue)

	log := &models.AuditLog{
		Action:      models.AuditActionDelete,
		EntityType:  entityType,
		EntityID:    entityID,
		EntityName:  entityName,
		UserID:      userID,
		UserName:    userName,
		UserEmail:   userEmail,
		OldValue:    oldValueJSON,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Description: "Registro excluído",
	}
	return s.repo.Create(log)
}

// LogStatusChange logs a status change (e.g., technician inactivated)
func (s *AuditService) LogStatusChange(userID, userName, userEmail, entityType, entityID, entityName, ipAddress, userAgent, oldStatus, newStatus string) error {
	changes := map[string]interface{}{
		"status": map[string]string{
			"old": oldStatus,
			"new": newStatus,
		},
	}
	changesJSON, _ := json.Marshal(changes)

	description := "Status alterado de " + oldStatus + " para " + newStatus

	log := &models.AuditLog{
		Action:      models.AuditActionUpdate,
		EntityType:  entityType,
		EntityID:    entityID,
		EntityName:  entityName,
		UserID:      userID,
		UserName:    userName,
		UserEmail:   userEmail,
		Changes:     changesJSON,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Description: description,
	}
	return s.repo.Create(log)
}

// computeChanges computes the differences between old and new values
func (s *AuditService) computeChanges(oldValue, newValue interface{}) map[string]interface{} {
	changes := make(map[string]interface{})

	oldMap := structToMap(oldValue)
	newMap := structToMap(newValue)

	for key, newVal := range newMap {
		if oldVal, exists := oldMap[key]; exists {
			if !reflect.DeepEqual(oldVal, newVal) {
				changes[key] = map[string]interface{}{
					"old": oldVal,
					"new": newVal,
				}
			}
		} else {
			changes[key] = map[string]interface{}{
				"old": nil,
				"new": newVal,
			}
		}
	}

	return changes
}

// structToMap converts a struct to a map
func structToMap(v interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	data, err := json.Marshal(v)
	if err != nil {
		return result
	}
	json.Unmarshal(data, &result)
	return result
}
