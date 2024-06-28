package services

import "github.com/syndtr/goleveldb/leveldb"

// LevelDBService เป็นโครงสร้างที่เก็บ instance ของ LevelDB
type LevelDBService struct {
	DB *leveldb.DB
}

// NewLevelDBService สร้าง instance ใหม่ของ LevelDBService และเปิดฐานข้อมูล
func NewLevelDBService(path string) (*LevelDBService, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &LevelDBService{
		DB: db,
	}, nil
}

// Close ปิดฐานข้อมูล LevelDB
func (service *LevelDBService) Close() error {
	return service.DB.Close()
}

// Create เพิ่มข้อมูลใหม่ในฐานข้อมูล
func (service *LevelDBService) Create(key, value string) error {
	return service.DB.Put([]byte(key), []byte(value), nil)
}

// Read อ่านข้อมูลจากฐานข้อมูล
func (service *LevelDBService) Read(key string) (string, error) {
	data, err := service.DB.Get([]byte(key), nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Update แก้ไขข้อมูลในฐานข้อมูล
func (service *LevelDBService) Update(key, newValue string) error {
	return service.DB.Put([]byte(key), []byte(newValue), nil)
}

// Delete ลบข้อมูลออกจากฐานข้อมูล
func (service *LevelDBService) Delete(key string) error {
	return service.DB.Delete([]byte(key), nil)
}
