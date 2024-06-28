package domain

import "time"

type Orders struct {
	ID        string `gorm:"primaryKey"`
	Symbol    string
	Quantity  int64
	Timeframe string
	Type      string
	Ema       *int64
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	DeletedAt *time.Time `gorm:"index"`
	Leverage  int64
	UserId    string
}