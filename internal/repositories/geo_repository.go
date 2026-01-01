package repositories

import (
	"tech-erp/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GeoRepository struct {
	db *gorm.DB
}

func NewGeoRepository(db *gorm.DB) *GeoRepository {
	return &GeoRepository{db: db}
}

// CreateLocation cria um novo registro de localização
func (r *GeoRepository) CreateLocation(location *models.TechnicianLocation) error {
	return r.db.Create(location).Error
}

// CreateLocations cria múltiplos registros de localização
func (r *GeoRepository) CreateLocations(locations []models.TechnicianLocation) error {
	return r.db.Create(&locations).Error
}

// UpsertLastLocation atualiza ou insere a última localização do técnico
func (r *GeoRepository) UpsertLastLocation(lastLoc *models.TechnicianLastLocation) error {
	return r.db.Save(lastLoc).Error
}

// GetLastLocation obtém a última localização de um técnico
func (r *GeoRepository) GetLastLocation(technicianID uuid.UUID) (*models.TechnicianLastLocation, error) {
	var lastLoc models.TechnicianLastLocation
	err := r.db.Preload("Technician").Where("technician_id = ?", technicianID).First(&lastLoc).Error
	if err != nil {
		return nil, err
	}
	return &lastLoc, nil
}

// GetLastLocations obtém as últimas localizações de múltiplos técnicos
func (r *GeoRepository) GetLastLocations(technicianIDs []uuid.UUID) ([]models.TechnicianLastLocation, error) {
	var locations []models.TechnicianLastLocation
	err := r.db.Preload("Technician").Where("technician_id IN ?", technicianIDs).Find(&locations).Error
	return locations, err
}

// GetAllLastLocations obtém todas as últimas localizações
func (r *GeoRepository) GetAllLastLocations(filter GeoFilter) ([]models.TechnicianLastLocation, int64, error) {
	var locations []models.TechnicianLastLocation
	var total int64

	query := r.db.Model(&models.TechnicianLastLocation{}).Preload("Technician")

	// Filtro por status
	if filter.Status != "" {
		query = query.Where("status_snapshot = ?", filter.Status)
	}

	// Filtro por tempo (apenas atualizados após)
	if filter.UpdatedSince != nil {
		query = query.Where("server_time >= ?", filter.UpdatedSince)
	}

	// Contar total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Paginação
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	// Ordenar por última atualização
	query = query.Order("server_time DESC")

	err := query.Find(&locations).Error
	return locations, total, err
}

// GetLocationHistory obtém o histórico de localizações de um técnico
func (r *GeoRepository) GetLocationHistory(technicianID uuid.UUID, filter HistoryFilter) ([]models.TechnicianLocation, int64, error) {
	var locations []models.TechnicianLocation
	var total int64

	query := r.db.Model(&models.TechnicianLocation{}).Where("technician_id = ?", technicianID)

	// Filtro por período
	if !filter.From.IsZero() {
		query = query.Where("server_time >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		query = query.Where("server_time <= ?", filter.To)
	}

	// Filtro por ticket
	if filter.TicketID != nil {
		query = query.Where("ticket_id = ?", filter.TicketID)
	}

	// Filtro por tipo de evento
	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}

	// Contar total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Paginação
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	// Ordenar por tempo
	query = query.Order("server_time ASC")

	err := query.Find(&locations).Error
	return locations, total, err
}

// GetTicketLocations obtém as localizações associadas a um ticket
func (r *GeoRepository) GetTicketLocations(ticketID uuid.UUID) ([]models.TechnicianLocation, error) {
	var locations []models.TechnicianLocation
	err := r.db.Preload("Technician").
		Where("ticket_id = ?", ticketID).
		Order("server_time ASC").
		Find(&locations).Error
	return locations, err
}

// CheckDuplicate verifica se já existe um evento similar (para deduplicação)
func (r *GeoRepository) CheckDuplicate(technicianID uuid.UUID, ticketID *uuid.UUID, eventType models.EventType, deviceTime time.Time) (bool, error) {
	var count int64
	query := r.db.Model(&models.TechnicianLocation{}).
		Where("technician_id = ?", technicianID).
		Where("event_type = ?", eventType).
		Where("device_time BETWEEN ? AND ?", deviceTime.Add(-time.Minute), deviceTime.Add(time.Minute))

	if ticketID != nil {
		query = query.Where("ticket_id = ?", ticketID)
	}

	err := query.Count(&count).Error
	return count > 0, err
}

// GetHistorySummary obtém resumo do histórico
func (r *GeoRepository) GetHistorySummary(technicianID uuid.UUID, from, to time.Time) (*models.HistorySummary, error) {
	var summary models.HistorySummary

	// Total de eventos
	r.db.Model(&models.TechnicianLocation{}).
		Where("technician_id = ? AND server_time BETWEEN ? AND ?", technicianID, from, to).
		Count((*int64)(&summary.TotalEvents))

	// Checkins
	r.db.Model(&models.TechnicianLocation{}).
		Where("technician_id = ? AND server_time BETWEEN ? AND ? AND event_type = ?", technicianID, from, to, models.EventTypeCheckin).
		Count((*int64)(&summary.Checkins))

	// Checkouts
	r.db.Model(&models.TechnicianLocation{}).
		Where("technician_id = ? AND server_time BETWEEN ? AND ? AND event_type = ?", technicianID, from, to, models.EventTypeCheckout).
		Count((*int64)(&summary.Checkouts))

	// Heartbeats
	r.db.Model(&models.TechnicianLocation{}).
		Where("technician_id = ? AND server_time BETWEEN ? AND ? AND event_type = ?", technicianID, from, to, models.EventTypeHeartbeat).
		Count((*int64)(&summary.Heartbeats))

	// Tickets visitados (distintos)
	r.db.Model(&models.TechnicianLocation{}).
		Where("technician_id = ? AND server_time BETWEEN ? AND ? AND ticket_id IS NOT NULL", technicianID, from, to).
		Distinct("ticket_id").
		Count((*int64)(&summary.TicketsVisited))

	return &summary, nil
}

// GetGeoSettings obtém as configurações globais ou por escopo
func (r *GeoRepository) GetGeoSettings(scopeID *uuid.UUID) (*models.GeoSettings, error) {
	var settings models.GeoSettings
	query := r.db.Model(&models.GeoSettings{})

	if scopeID != nil {
		query = query.Where("scope_id = ?", scopeID)
	} else {
		query = query.Where("scope_id IS NULL")
	}

	err := query.First(&settings).Error
	if err == gorm.ErrRecordNotFound {
		// Retornar configurações padrão
		return &models.GeoSettings{
			RetentionDays:         90,
			HeartbeatIntervalMin:  5,
			HeartbeatEnabled:      false,
			RequireLocationCheckin: false,
		}, nil
	}
	return &settings, err
}

// GetAllGeoSettings obtém todas as configurações
func (r *GeoRepository) GetAllGeoSettings() ([]models.GeoSettings, error) {
	var settings []models.GeoSettings
	err := r.db.Find(&settings).Error
	return settings, err
}

// UpsertGeoSettings atualiza ou insere configurações
func (r *GeoRepository) UpsertGeoSettings(settings *models.GeoSettings) error {
	return r.db.Save(settings).Error
}

// DeleteOldLocations remove localizações antigas baseado na retenção
func (r *GeoRepository) DeleteOldLocations(retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := r.db.Where("server_time < ?", cutoff).Delete(&models.TechnicianLocation{})
	return result.RowsAffected, result.Error
}

// CountRecentLocations conta localizações recentes (para rate limiting)
func (r *GeoRepository) CountRecentLocations(technicianID uuid.UUID, eventType models.EventType, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&models.TechnicianLocation{}).
		Where("technician_id = ? AND event_type = ? AND server_time >= ?", technicianID, eventType, since).
		Count(&count).Error
	return count, err
}

// Filtros

type GeoFilter struct {
	ScopeIDs     []uuid.UUID
	Status       string
	Query        string
	UpdatedSince *time.Time
	Limit        int
	Offset       int
}

type HistoryFilter struct {
	From      time.Time
	To        time.Time
	TicketID  *uuid.UUID
	EventType string
	Limit     int
	Offset    int
}
