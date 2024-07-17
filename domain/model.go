package domain

import "time"

type Orders struct {
	ID        string `gorm:"primaryKey"`
	Symbol    string `gorm:"index:symbol_idx"`   // เพิ่ม index ให้กับคอลัมน์ Symbol
	Quantity  int64  `gorm:"index:quantity_idx"` // เพิ่ม index ให้กับคอลัมน์ Quantity
	Timeframe string
	Type      string `gorm:"index:type_idx"` // เพิ่ม index ให้กับคอลัมน์ Type
	Ema       *int64
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime"`
	DeletedAt *time.Time `gorm:"index"`
	Leverage  int64      `gorm:"index:leverage_idx"`               // เพิ่ม index ให้กับคอลัมน์ Leverage
	UserId    string     `json:"user_id" gorm:"index:user_id_idx"` // ระบุชื่อ key ใน JSON เป็น user_id
	Status    *string
}
