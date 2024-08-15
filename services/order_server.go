package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"order-server/domain"
	"order-server/rabbitmq"

	// "order-server/rabbitmq"

	"gorm.io/gorm"
)

type OrderServer struct {
	db      *gorm.DB
	levelDB *LevelDBService
}

func NewOrderServer(db *gorm.DB, levelDB *LevelDBService) *OrderServer {
	return &OrderServer{
		db:      db,
		levelDB: levelDB,
	}
}

func (s *OrderServer) ProcessOrder(timeframe string) {
	signalService := NewSignalService(s.levelDB)
	producer := rabbitmq.NewOrderProducer()
	var orderFutureQueue = "order_future_queue"
	var closePositionQueue = "close_position_queue"

	orders, err := s.groupOrder(timeframe)
	if err != nil {
		return
	}

	for _, item := range orders {
		if item.Type == "EMA" {
			position, err := signalService.signalEMA(item.Symbol, item.Timeframe, item.Ema)
			if err != nil {
				fmt.Println(err)
				continue
			}

			result, err := s.queryOrder(item.Symbol, item.Type, item.Ema)
			if err != nil {
				fmt.Println(err)
				continue // Skip to the next iteration of the loop
			}

			status, err := s.queryPosition(item.Symbol, item.Ema)
			if err != nil {
				fmt.Println(err)
				continue // Skip to the next iteration of the loop
			}

			if status.Status != "" && status.Status != position.position {
				fmt.Println("hello world")
				for _, order := range *result {
					orderWithPosition := positionOrders{
						Position: position.position,
						Order:    &order,
					}

					orderJSON, err := json.Marshal(orderWithPosition)
					if err != nil {
						continue
					}

					// send to queue close position
					producer.SendTask(closePositionQueue, string(orderJSON))
				}
			}

			if position == nil {
				fmt.Println("position is nil")
				continue // Skip to the next iteration of the loop
			}

			if position.position == "" {
				fmt.Println("position.Position is empty")
				continue // Skip to the next iteration of the loop
			}

			for _, order := range *result {
				orderWithPosition := positionOrders{
					Position: position.position,
					Order:    &order,
				}

				orderJSON, err := json.Marshal(orderWithPosition)
				if err != nil {
					continue
				}
				SendLineNotify(string(orderJSON))

				producer.SendTask(orderFutureQueue, string(orderJSON))
			}
		}
	}

	// for _, item := range orders {
	// 	if item.Type == "EMA" {
	// 		position, err := signalService.signalEMA(item.Symbol, item.Timeframe, item.Ema)
	// 		if err != nil {
	// 			fmt.Println(err)
	// 			return
	// 		}
	// 		SendLineNotify(position.position)

	// 		if position != nil {
	// 			result, err := s.queryOrder(item.Symbol, item.Type, item.Ema)
	// 			if err != nil {
	// 				return
	// 			}

	// 			for _, order := range *result {
	// 				orderWithPosition := positionOrders{
	// 					Position: position.position,
	// 					Order:    &order,
	// 				}

	// 				orderJSON, err := json.Marshal(orderWithPosition)
	// 				if err != nil {
	// 					return
	// 				}
	// 				producer.SendTask(orderFutureQueue, string(orderJSON))
	// 			}
	// 		}
	// 	} else if item.Type == "CDC" {
	// 		position, err := signalService.signalCDC(item.Symbol, item.Timeframe)
	// 		if err != nil {
	// 			fmt.Println(err)
	// 			return
	// 		}

	// 		if position != nil {
	// 			result, err := s.queryOrder(item.Symbol, item.Type, item.Ema)
	// 			if err != nil {
	// 				return
	// 			}

	// 			for _, order := range *result {
	// 				orderWithPosition := positionOrders{
	// 					Position: position.position,
	// 					Order:    &order,
	// 				}

	// 				orderJSON, err := json.Marshal(orderWithPosition)
	// 				if err != nil {
	// 					return
	// 				}

	// 				producer.SendTask(orderFutureQueue, string(orderJSON))
	// 			}
	// 		}
	// 	}
	// }
}

type positionOrders struct {
	Position string         `json:"position"`
	Order    *domain.Orders `json:"order"`
}

type queryPosition struct {
	Symbol string
	Ema    int
	Status string
}

func (s *OrderServer) queryPosition(symbol string, ema int) (*queryPosition, error) {
	var order queryPosition

	// ใช้ Subquery เพื่อให้ได้ผลลัพธ์ที่ไม่ซ้ำกัน
	err := s.db.Raw(`
		SELECT symbol, ema, status
		FROM (
			SELECT symbol, ema, status
			FROM orders
			WHERE symbol = ? AND ema = ?
			GROUP BY symbol, ema, status
		) AS subquery
	`, symbol, ema).Scan(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no orders found")
		}
		return nil, fmt.Errorf("failed to query orders: %v", err)
	}

	return &order, nil
}

func (s *OrderServer) queryOrder(symbol, types string, ema ...int) (*[]domain.Orders, error) {
	var orders []domain.Orders

	query := s.db.Model(&domain.Orders{}).
		Where("symbol = ? AND type = ? AND deleted_at IS NULL", symbol, types).
		Select("symbol, quantity, leverage, ema, user_id, id")

	if len(ema) > 0 {
		query = query.Where("ema = ?", ema[0])
	}

	err := query.Find(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}

	return &orders, nil
}

type order struct {
	Symbol    string
	Ema       int
	Timeframe string
	Type      string
}

func (s *OrderServer) groupOrder(timeframe string) ([]*order, error) {
	var orders []*order

	// ทำ Group By และเลือกคอลัมน์ที่ต้องการ
	err := s.db.Model(&domain.Orders{}).
		Select("symbol, ema, timeframe, type").
		Where("timeframe = ?", timeframe).
		Group("symbol, ema, timeframe, type").
		Find(&orders).Error

	if err != nil {
		return nil, fmt.Errorf("failed to group orders: %v", err)
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("no orders found")
	}

	return orders, nil
}

func SendLineNotify(msg string) error {
	endpoint := "https://notify-api.line.me/api/notify"
	token := "41U6HJq0N1chNIjynWGCp5BEIbrABjEQX15DcUrBoSd"

	// สร้าง form data สำหรับ request
	data := url.Values{}
	data.Set("message", msg)

	// สร้าง request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// กำหนด headers
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	// ส่ง request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// ตรวจสอบสถานะการตอบกลับ
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response status: %d", resp.StatusCode)
	}

	return nil
}
