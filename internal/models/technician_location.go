package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FlexibleTime é um tipo de tempo que aceita múltiplos formatos de data
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON implementa json.Unmarshaler para FlexibleTime
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// Remove as aspas
	s := string(data)
	if s == "null" || s == `""` {
		return nil
	}
	s = s[1 : len(s)-1] // Remove quotes

	// Tenta múltiplos formatos
	formats := []string{
		time.RFC3339Nano,                // 2006-01-02T15:04:05.999999999Z07:00
		time.RFC3339,                    // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05.999999999", // Sem timezone (assume UTC)
		"2006-01-02T15:04:05.999",       // Com milissegundos
		"2006-01-02T15:04:05",           // Sem milissegundos
		"2006-01-02 15:04:05",           // Espaço ao invés de T
	}

	var parseErr error
	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			ft.Time = t
			return nil
		}
		parseErr = err
	}
	return parseErr
}

// MarshalJSON implementa json.Marshaler para FlexibleTime
func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	if ft.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(ft.Time.Format(time.RFC3339Nano))
}

// ToTime retorna o time.Time interno
func (ft *FlexibleTime) ToTime() *time.Time {
	if ft == nil || ft.Time.IsZero() {
		return nil
	}
	return &ft.Time
}

// EventType representa o tipo de evento de localização
type EventType string

const (
	EventTypeCheckin   EventType = "CHECKIN"
	EventTypeCheckout  EventType = "CHECKOUT"
	EventTypeHeartbeat EventType = "HEARTBEAT"
)

// TechnicianLocation representa um registro de localização do técnico
type TechnicianLocation struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TechnicianID string     `json:"technicianId" gorm:"type:varchar(36);not null;index:idx_tech_loc_technician_time,priority:1"`
	TicketID     *uuid.UUID `json:"ticketId,omitempty" gorm:"type:uuid;index:idx_tech_loc_ticket"`

	// Tipo de evento
	EventType EventType `json:"eventType" gorm:"type:varchar(20);not null"`

	// Coordenadas
	Latitude  float64 `json:"latitude" gorm:"type:double precision;not null"`
	Longitude float64 `json:"longitude" gorm:"type:double precision;not null"`

	// Metadados de precisão
	AccuracyM  *float64 `json:"accuracyM,omitempty" gorm:"type:double precision"`
	AltitudeM  *float64 `json:"altitudeM,omitempty" gorm:"type:double precision"`
	SpeedMps   *float64 `json:"speedMps,omitempty" gorm:"type:double precision"`
	HeadingDeg *float64 `json:"headingDeg,omitempty" gorm:"type:double precision"`

	// Provedor e dispositivo
	Provider   *string    `json:"provider,omitempty" gorm:"type:varchar(20)"`
	DeviceTime *time.Time `json:"deviceTime,omitempty" gorm:"type:timestamptz"`
	ServerTime time.Time  `json:"serverTime" gorm:"type:timestamptz;not null;default:now();index:idx_tech_loc_server_time;index:idx_tech_loc_technician_time,priority:2,sort:desc"`

	// Flags
	IsMocked      bool `json:"isMocked" gorm:"default:false"`
	IsOfflineSync bool `json:"isOfflineSync" gorm:"default:false"`

	// Audit
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`

	// Relacionamentos
	Technician *Technician `json:"technician,omitempty" gorm:"foreignKey:TechnicianID"`
}

func (TechnicianLocation) TableName() string {
	return "technician_locations"
}

// TechnicianLastLocation representa a última localização conhecida do técnico (cache)
type TechnicianLastLocation struct {
	TechnicianID string `json:"technicianId" gorm:"type:varchar(36);primary_key"`

	// Última localização
	Latitude  float64  `json:"latitude" gorm:"type:double precision;not null"`
	Longitude float64  `json:"longitude" gorm:"type:double precision;not null"`
	AccuracyM *float64 `json:"accuracyM,omitempty" gorm:"type:double precision"`

	// Contexto
	EventType      EventType  `json:"eventType" gorm:"type:varchar(20);not null"`
	TicketID       *uuid.UUID `json:"ticketId,omitempty" gorm:"type:uuid"`
	StatusSnapshot *string    `json:"statusSnapshot,omitempty" gorm:"type:varchar(30);index:idx_tech_last_loc_status"`

	// Timestamps
	DeviceTime *time.Time `json:"deviceTime,omitempty" gorm:"type:timestamptz"`
	ServerTime time.Time  `json:"serverTime" gorm:"type:timestamptz;not null"`
	UpdatedAt  time.Time  `json:"updatedAt" gorm:"autoUpdateTime"`

	// Relacionamentos
	Technician *Technician `json:"technician,omitempty" gorm:"foreignKey:TechnicianID"`
}

func (TechnicianLastLocation) TableName() string {
	return "technician_last_locations"
}

// GeoSettings representa configurações de geolocalização por escopo
type GeoSettings struct {
	ID      uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ScopeID *uuid.UUID `json:"scopeId,omitempty" gorm:"type:uuid;uniqueIndex"`

	// Configurações
	RetentionDays          int  `json:"retentionDays" gorm:"not null;default:90"`
	HeartbeatIntervalMin   int  `json:"heartbeatIntervalMin" gorm:"not null;default:5"`
	HeartbeatEnabled       bool `json:"heartbeatEnabled" gorm:"not null;default:false"`
	RequireLocationCheckin bool `json:"requireLocationCheckin" gorm:"not null;default:false"`

	// Audit
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (GeoSettings) TableName() string {
	return "geo_settings"
}

// DTOs para requests/responses

type CreateLocationRequest struct {
	TicketID   *uuid.UUID    `json:"ticketId"`
	EventType  EventType     `json:"eventType" validate:"required,oneof=CHECKIN CHECKOUT HEARTBEAT"`
	Latitude   float64       `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude  float64       `json:"longitude" validate:"required,min=-180,max=180"`
	AccuracyM  *float64      `json:"accuracyM"`
	AltitudeM  *float64      `json:"altitudeM"`
	SpeedMps   *float64      `json:"speedMps"`
	HeadingDeg *float64      `json:"headingDeg"`
	Provider   *string       `json:"provider"`
	DeviceTime *FlexibleTime `json:"deviceTime"`
	IsMocked   bool          `json:"isMocked"`
}

type BatchLocationRequest struct {
	Locations []BatchLocationItem `json:"locations" validate:"required,min=1,max=100,dive"`
}

type BatchLocationItem struct {
	LocalID    string        `json:"localId" validate:"required"`
	TicketID   *uuid.UUID    `json:"ticketId"`
	EventType  EventType     `json:"eventType" validate:"required,oneof=CHECKIN CHECKOUT HEARTBEAT"`
	Latitude   float64       `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude  float64       `json:"longitude" validate:"required,min=-180,max=180"`
	AccuracyM  *float64      `json:"accuracyM"`
	AltitudeM  *float64      `json:"altitudeM"`
	SpeedMps   *float64      `json:"speedMps"`
	HeadingDeg *float64      `json:"headingDeg"`
	Provider   *string       `json:"provider"`
	DeviceTime *FlexibleTime `json:"deviceTime"`
	IsMocked   bool          `json:"isMocked"`
}

type BatchLocationResult struct {
	LocalID  string    `json:"localId"`
	ServerID uuid.UUID `json:"serverId,omitempty"`
	Status   string    `json:"status"` // "created", "duplicate", "error"
	Error    string    `json:"error,omitempty"`
}

type TechnicianLocationResponse struct {
	TechnicianID        string        `json:"technicianId"`
	TicketID            *uuid.UUID    `json:"ticketId,omitempty"`
	Name                string        `json:"name"`
	City                string        `json:"city,omitempty"`
	State               string        `json:"state,omitempty"`
	Street              string        `json:"street,omitempty"`
	Number              string        `json:"number,omitempty"`
	Neighborhood        string        `json:"neighborhood,omitempty"`
	AvatarURL           *string       `json:"avatarUrl,omitempty"`
	Status              string        `json:"status"`
	CurrentTicketID     *uuid.UUID    `json:"currentTicketId,omitempty"`
	CurrentTicketNumber *string       `json:"currentTicketNumber,omitempty"`
	Location            *LocationInfo `json:"location,omitempty"`
}

type LocationInfo struct {
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	AccuracyM  *float64  `json:"accuracyM,omitempty"`
	ServerTime time.Time `json:"serverTime"`
	EventType  EventType `json:"eventType"`
	MinutesAgo int       `json:"minutesAgo"`
}

type TechnicianHistoryResponse struct {
	TechnicianID   string                `json:"technicianId"`
	TechnicianName string                `json:"technicianName"`
	Period         PeriodInfo            `json:"period"`
	Summary        HistorySummary        `json:"summary"`
	Locations      []LocationHistoryItem `json:"locations"`
}

type PeriodInfo struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type HistorySummary struct {
	TotalEvents    int64 `json:"totalEvents"`
	Checkins       int64 `json:"checkins"`
	Checkouts      int64 `json:"checkouts"`
	Heartbeats     int64 `json:"heartbeats"`
	TicketsVisited int64 `json:"ticketsVisited"`
}

type LocationHistoryItem struct {
	ID           uuid.UUID  `json:"id"`
	EventType    EventType  `json:"eventType"`
	TicketID     *uuid.UUID `json:"ticketId,omitempty"`
	TicketNumber *string    `json:"ticketNumber,omitempty"`
	Latitude     float64    `json:"latitude"`
	Longitude    float64    `json:"longitude"`
	AccuracyM    *float64   `json:"accuracyM,omitempty"`
	ServerTime   time.Time  `json:"serverTime"`
}

type TicketLocationsResponse struct {
	TicketID           uuid.UUID           `json:"ticketId"`
	TicketNumber       string              `json:"ticketNumber"`
	Checkin            *CheckinoutInfo     `json:"checkin,omitempty"`
	Checkout           *CheckinoutInfo     `json:"checkout,omitempty"`
	Heartbeats         []HeartbeatInfo     `json:"heartbeats,omitempty"`
	DistanceFromClient *DistanceInfo       `json:"distanceFromClient,omitempty"`
	Sessions           []AttendanceSession `json:"sessions,omitempty"`
}

// AttendanceSession representa uma sessão de atendimento (check-in + checkout)
type AttendanceSession struct {
	ID             string          `json:"id"`
	TechnicianID   string          `json:"technicianId"`
	TechnicianName string          `json:"technicianName"`
	Checkin        *CheckinoutInfo `json:"checkin"`
	Checkout       *CheckinoutInfo `json:"checkout,omitempty"`
	Duration       *int64          `json:"durationSeconds,omitempty"`
	Status         string          `json:"status"` // "in_progress", "completed"
}

type CheckinoutInfo struct {
	TechnicianID   string    `json:"technicianId"`
	TechnicianName string    `json:"technicianName"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	AccuracyM      *float64  `json:"accuracyM,omitempty"`
	ServerTime     time.Time `json:"serverTime"`
}

type HeartbeatInfo struct {
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	ServerTime time.Time `json:"serverTime"`
}

type DistanceInfo struct {
	CheckinMeters     *float64 `json:"checkinMeters,omitempty"`
	CheckoutMeters    *float64 `json:"checkoutMeters,omitempty"`
	ClientAddress     *string  `json:"clientAddress,omitempty"`
	ClientCoordinates *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"clientCoordinates,omitempty"`
}

type GeoSettingsResponse struct {
	Global GeoSettingsInfo    `json:"global"`
	Scopes []ScopeGeoSettings `json:"scopes,omitempty"`
}

type GeoSettingsInfo struct {
	RetentionDays          int  `json:"retentionDays"`
	HeartbeatIntervalMin   int  `json:"heartbeatIntervalMin"`
	HeartbeatEnabled       bool `json:"heartbeatEnabled"`
	RequireLocationCheckin bool `json:"requireLocationCheckin"`
}

type ScopeGeoSettings struct {
	ScopeID                uuid.UUID `json:"scopeId"`
	ScopeName              string    `json:"scopeName"`
	RetentionDays          int       `json:"retentionDays"`
	HeartbeatIntervalMin   int       `json:"heartbeatIntervalMin"`
	HeartbeatEnabled       bool      `json:"heartbeatEnabled"`
	RequireLocationCheckin bool      `json:"requireLocationCheckin"`
}

type UpdateGeoSettingsRequest struct {
	ScopeID                *uuid.UUID `json:"scopeId"`
	RetentionDays          *int       `json:"retentionDays"`
	HeartbeatIntervalMin   *int       `json:"heartbeatIntervalMin"`
	HeartbeatEnabled       *bool      `json:"heartbeatEnabled"`
	RequireLocationCheckin *bool      `json:"requireLocationCheckin"`
}

// GeoClusterItem representa um ponto no mapa: cluster (vários técnicos) ou técnico individual
type GeoClusterItem struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Count int     `json:"count"`
	// Preenchido somente quando IsExact=true (zoom >= 12, técnico individual)
	TechnicianID string `json:"technicianId,omitempty"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	HasRealLoc   bool   `json:"hasRealLocation,omitempty"`
	MinutesAgo   int    `json:"minutesAgo,omitempty"`
}

// GeoClustersResponse é a resposta do endpoint de clusters
type GeoClustersResponse struct {
	Clusters   []GeoClusterItem `json:"clusters"`
	TotalCount int              `json:"totalCount"`
	Zoom       int              `json:"zoom"`
	// IsExact = true quando cada item é um técnico individual (zoom >= 12)
	IsExact bool `json:"isExact"`
}
