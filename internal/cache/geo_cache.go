package cache

import (
	"encoding/json"
	"fmt"
	"time"
)

// Geo Cache Keys
const (
	// Key para armazenar todos os técnicos com localização
	// Estrutura: Hash com technicianId -> TechnicianGeoData (JSON)
	AllTechniciansGeoKey = "geo:technicians:all"
	
	// Key para armazenar última localização de um técnico específico
	TechnicianLastLocationKeyPrefix = "geo:technician:last:"
	
	// Key para sinalizar que o cache foi carregado
	GeoCacheLoadedKey = "geo:cache:loaded"
)

// Geo Cache TTLs
const (
	// TTL para o cache geral de todos os técnicos
	AllTechniciansGeoTTL = 30 * time.Minute
	
	// TTL para última localização individual (mais curto para dados em tempo real)
	TechnicianLastLocationTTL = 5 * time.Minute
	
	// TTL para flag de cache carregado
	GeoCacheLoadedTTL = 24 * time.Hour
)

// TechnicianGeoData representa os dados de um técnico para o mapa
type TechnicianGeoData struct {
	TechnicianID   string   `json:"technicianId"`
	Name           string   `json:"name"`
	City           string   `json:"city"`
	State          string   `json:"state"`
	Status         string   `json:"status"`
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	AccuracyM      *float64 `json:"accuracyM,omitempty"`
	EventType      string   `json:"eventType,omitempty"`
	LastUpdateTime *int64   `json:"lastUpdateTime,omitempty"` // Unix timestamp
	HasRealLocation bool    `json:"hasRealLocation"` // true se tem localização real do app
}

// TechnicianLastLocationKey retorna a chave Redis para última localização de um técnico
func TechnicianLastLocationKey(technicianID string) string {
	return fmt.Sprintf("%s%s", TechnicianLastLocationKeyPrefix, technicianID)
}

// GeoService cache methods - a serem adicionados ao RedisClient

// SetAllTechniciansGeo armazena todos os técnicos no cache usando Hash
func (r *RedisClient) SetAllTechniciansGeo(technicians []TechnicianGeoData) error {
	if len(technicians) == 0 {
		return nil
	}
	
	// Usar pipeline para performance
	pipe := r.client.Pipeline()
	
	// Deletar hash anterior
	pipe.Del(r.ctx, AllTechniciansGeoKey)
	
	// Adicionar cada técnico como field no hash
	for _, tech := range technicians {
		jsonValue, err := json.Marshal(tech)
		if err != nil {
			continue
		}
		pipe.HSet(r.ctx, AllTechniciansGeoKey, tech.TechnicianID, jsonValue)
	}
	
	// Definir TTL
	pipe.Expire(r.ctx, AllTechniciansGeoKey, AllTechniciansGeoTTL)
	
	// Marcar cache como carregado
	pipe.Set(r.ctx, GeoCacheLoadedKey, "1", GeoCacheLoadedTTL)
	
	_, err := pipe.Exec(r.ctx)
	return err
}

// GetAllTechniciansGeo retorna todos os técnicos do cache
func (r *RedisClient) GetAllTechniciansGeo() ([]TechnicianGeoData, error) {
	result, err := r.client.HGetAll(r.ctx, AllTechniciansGeoKey).Result()
	if err != nil {
		return nil, err
	}
	
	technicians := make([]TechnicianGeoData, 0, len(result))
	for _, jsonValue := range result {
		var tech TechnicianGeoData
		if err := json.Unmarshal([]byte(jsonValue), &tech); err != nil {
			continue
		}
		technicians = append(technicians, tech)
	}
	
	return technicians, nil
}

// UpdateTechnicianLocation atualiza a localização de um técnico específico no cache
func (r *RedisClient) UpdateTechnicianLocation(tech TechnicianGeoData) error {
	jsonValue, err := json.Marshal(tech)
	if err != nil {
		return err
	}
	
	// Atualizar no hash geral
	return r.client.HSet(r.ctx, AllTechniciansGeoKey, tech.TechnicianID, jsonValue).Err()
}

// GetTechnicianGeo retorna um técnico específico do cache
func (r *RedisClient) GetTechnicianGeo(technicianID string) (*TechnicianGeoData, error) {
	jsonValue, err := r.client.HGet(r.ctx, AllTechniciansGeoKey, technicianID).Result()
	if err != nil {
		return nil, err
	}
	
	var tech TechnicianGeoData
	if err := json.Unmarshal([]byte(jsonValue), &tech); err != nil {
		return nil, err
	}
	
	return &tech, nil
}

// IsGeoCacheLoaded verifica se o cache de geo foi carregado
func (r *RedisClient) IsGeoCacheLoaded() bool {
	exists, err := r.Exists(GeoCacheLoadedKey)
	if err != nil {
		return false
	}
	return exists
}

// GetGeoCacheCount retorna a quantidade de técnicos no cache
func (r *RedisClient) GetGeoCacheCount() (int64, error) {
	return r.client.HLen(r.ctx, AllTechniciansGeoKey).Result()
}
