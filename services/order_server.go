package services

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"order-server/gRPC"
	"time"

	"gorm.io/gorm"
)

type OrderServer struct {
	gRPC.UnimplementedOrdersServiceServer
	db      *gorm.DB
	levelDB *LevelDBService
}

func NewOrderServer(db *gorm.DB, levelDB *LevelDBService) gRPC.OrdersServiceServer {
	return &OrderServer{
		db:      db,
		levelDB: levelDB,
	}
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *gRPC.OrdersDto) (*gRPC.OrderResponse, error) {
	s.ProcessOrder()

	return &gRPC.OrderResponse{
		Message:    "Order created successfully",
		StatusCode: 200,
	}, nil
}

func (s *OrderServer) ProcessOrder() {
	signalService := NewSignalService(s.levelDB)
	fmt.Println("Cron job running at", time.Now())
	orders, err := s.groupOrder()
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
				// test send line
				msg := fmt.Sprintf("EMA %s/%s",
					position.position,
					item.Symbol,
				)
				SendLineNotify(msg)
				// send order to queue
			}
		} else if item.Type == "CDC" {
			position, err := signalService.signalCDC(item.Symbol, item.Timeframe)
			if err != nil {
				fmt.Println(err)
				return
			}
			if position != nil {
				// test send line
				msg := fmt.Sprintf("CDC %s/%s",
					position.position,
					item.Symbol,
				)
				SendLineNotify(msg)
				// send order to queue
			}
		}
	}
}

type order struct {
	Symbol    string
	Ema       int
	Timeframe string
	Type      string
}

func (s *OrderServer) groupOrder() ([]*order, error) {
	orders := []*order{
		{
			Symbol:    "ETHUSDT",
			Ema:       15,
			Timeframe: "5m",
			Type:      "EMA",
		},
		{
			Symbol:    "BTCUSDT",
			Timeframe: "5m",
			Type:      "CDC",
		},
		{
			Symbol:    "DOGEUSDT",
			Timeframe: "5m",
			Ema:       15,
			Type:      "EMA",
		},
		{
			Symbol:    "XRPUSDT",
			Timeframe: "5m",
			Type:      "CDC",
		},
		{
			Symbol:    "ADAUSDT",
			Timeframe: "5m",
			Ema:       15,
			Type:      "EMA",
		},
		{
			Symbol:    "BNBUSDT",
			Timeframe: "5m",
			Ema:       15,
			Type:      "EMA",
		},
	}
	// // ทำ Group By และเลือกคอลัมน์ที่ต้องการ
	// err := s.db.Model(&domain.Orders{}).
	// 	Select("symbol, ema, timeframe, type").
	// 	Group("symbol, ema, timeframe, type").
	// 	Scan(&orders).Error

	// if err != nil {
	// 	return nil, err
	// }

	// if len(orders) == 0 {
	// 	return nil, fmt.Errorf("not found orders")
	// }

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
