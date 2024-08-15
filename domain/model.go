package domain

import "time"

type Orders struct {
	ID        string `json:"id" gorm:"primaryKey"`
	Symbol    string `json:"symbol" gorm:"index:symbol_idx"`     // เพิ่ม index ให้กับคอลัมน์ Symbol
	Quantity  int64  `json:"quantity" gorm:"index:quantity_idx"` // เพิ่ม index ให้กับคอลัมน์ Quantity
	Timeframe string `json:"timeframe"`
	Type      string `json:"type" gorm:"index:type_idx"` // เพิ่ม index ให้กับคอลัมน์ Type
	Ema       *int64
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
	Leverage  int64      `json:"leverage" gorm:"index:leverage_idx"` // เพิ่ม index ให้กับคอลัมน์ Leverage
	UserId    string     `json:"user_id" gorm:"index:user_id_idx"`   // ระบุชื่อ key ใน JSON เป็น user_id
	Status    *string    `json:"status"`
}
