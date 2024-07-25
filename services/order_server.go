package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"order-server/domain"
	"order-server/rabbitmq"

	status "google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type OrderServer struct {
	UnimplementedOrdersServiceServer
	db      *gorm.DB
	levelDB *LevelDBService
}

func NewOrderServer(db *gorm.DB, levelDB *LevelDBService) OrdersServiceServer {
	return &OrderServer{
		db:      db,
		levelDB: levelDB,
	}
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *OrdersDto) (*OrderResponse, error) {
	var existingSymbol domain.Orders
	var orderTxQueue = "order_tx_queue"

	producer := rabbitmq.NewOrderProducer()

	// Start a new transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, status.Errorf(500, "failed to start transaction: %v", tx.Error)
	}

	// Check if the symbol already exists for the user
	err := tx.Model(&domain.Orders{}).
		Where("user_id = ? AND symbol = ? AND deleted_at IS NULL", req.UserId, req.Symbol).
		Select("symbol").
		First(&existingSymbol).Error

	// Handle database query error
	if err != nil && err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return nil, status.Errorf(400, "database query error: %v", err)
	}

	// If the symbol exists, return an error with status
	if err == nil {
		tx.Rollback()
		return nil, status.Errorf(409, "Symbol '%s' already exists for user %s", req.Symbol, req.UserId)
	}

	// convert to string
	order, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("JSON marshalling error: %v", err)
	}

	// send to msg to consumer
	producer.SendMsg(orderTxQueue, string(order))

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, status.Errorf(500, "failed to commit transaction: %v", err)
	}

	// Return successful response
	return &OrderResponse{
		Message:    "Order created successfully",
		StatusCode: 200,
	}, nil
}

func (s *OrderServer) ProcessOrder(timeframe string) {
	signalService := NewSignalService(s.levelDB)
	// producer := rabbitmq.NewOrderProducer()
	// var orderFutureQueue = "order_future_queue"

	orders, err := s.groupOrder(timeframe)
	if err != nil {
		return
	}

	for _, item := range orders {
		if item.Type == "EMA" {
			position, err := signalService.signalEMA(item.Symbol, item.Timeframe, item.Ema)
			if err != nil {
				fmt.Println(err)
				return
			}
			if position != nil {
				result, err := s.queryOrder(item.Symbol, item.Type, item.Ema)
				if err != nil {
					return
				}

				for _, order := range *result {
					orderWithPosition := positionOrders{
						Position: position.position,
						Order:    &order,
					}

					orderJSON, err := json.Marshal(orderWithPosition)
					if err != nil {
						return
					}

					SendLineNotify(string(orderJSON))

					// producer.SendMsg(orderFutureQueue, string(orderJSON))
				}
			}
		} else if item.Type == "CDC" {
			position, err := signalService.signalCDC(item.Symbol, item.Timeframe)
			if err != nil {
				fmt.Println(err)
				return
			}

			if position != nil {
				result, err := s.queryOrder(item.Symbol, item.Type, item.Ema)
				if err != nil {
					return
				}

				for _, order := range *result {
					orderWithPosition := positionOrders{
						Position: position.position,
						Order:    &order,
					}

					orderJSON, err := json.Marshal(orderWithPosition)
					if err != nil {
						return
					}

					SendLineNotify(string(orderJSON))

					// producer.SendMsg(orderFutureQueue, string(orderJSON))
				}
			}
		}
	}
}

type positionOrders struct {
	Position string         `json:"position"`
	Order    *domain.Orders `json:"order"`
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
