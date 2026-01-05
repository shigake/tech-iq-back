package models

// City representa um município brasileiro com suas coordenadas
type City struct {
	ID         uint    `json:"id" gorm:"primaryKey;autoIncrement"`
	CodigoIBGE int     `json:"codigoIbge" gorm:"uniqueIndex;not null"`
	Nome       string  `json:"nome" gorm:"type:varchar(255);not null;index"`
	NomeNorm   string  `json:"nomeNorm" gorm:"type:varchar(255);not null;index"` // Nome normalizado (sem acentos, lowercase)
	UF         string  `json:"uf" gorm:"type:varchar(2);not null;index"`
	Latitude   float64 `json:"latitude" gorm:"type:decimal(10,6);not null"`
	Longitude  float64 `json:"longitude" gorm:"type:decimal(11,6);not null"`
	Capital    bool    `json:"capital" gorm:"default:false"`
	DDD        int     `json:"ddd" gorm:"type:int"`
}

func (City) TableName() string {
	return "cities"
}
