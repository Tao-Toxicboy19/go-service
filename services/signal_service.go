package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type SignalService struct {
	levelDB *LevelDBService
}

func NewSignalService(db *LevelDBService) *SignalService {
	return &SignalService{
		levelDB: db,
	}
}

type EmaRes struct {
	ema       float64
	lastPrice float64
}

func (s *SignalService) calculateEMA(data []float64, ema int) (*EmaRes, error) {

	if len(data) < int(ema) {
		return nil, fmt.Errorf("not enough data points for EMA calculation")
	}

	k := 2.0 / float64(ema+1)
	var emaValue float64

	// Initialize EMA with the average of the first `ema` data points
	for i := 0; i < int(ema); i++ {
		emaValue += float64(data[i])
	}
	emaValue /= float64(ema)

	// Calculate EMA for the rest of the data points
	for i := int(ema); i < len(data); i++ {
		emaValue = (float64(data[i])-emaValue)*k + emaValue
	}

	return &EmaRes{
		ema:       emaValue,
		lastPrice: data[len(data)-1],
	}, nil
}

type Candle struct {
	Timestamp                int64   `json:"timestamp"`
	Open                     float64 `json:"open"`
	High                     float64 `json:"high"`
	Low                      float64 `json:"low"`
	Close                    float64 `json:"close"`
	Volume                   float64 `json:"volume"`
	CloseTimestamp           int64   `json:"closeTimestamp"`
	QuoteAssetVolume         float64 `json:"quoteAssetVolume"`
	NumberOfTrades           int64   `json:"numberOfTrades"`
	TakerBuyBaseAssetVolume  float64 `json:"takerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume float64 `json:"takerBuyQuoteAssetVolume"`
	Ignore                   string  `json:"ignore"`
}

func fetch(symbol, timeframe string, limit int) ([]float64, error) {
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=%d", symbol, timeframe, limit)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching data: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var rawCandles [][]interface{}
	err = json.Unmarshal(body, &rawCandles)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	closePrices := make([]float64, len(rawCandles))
	for i, rawCandle := range rawCandles {
		if len(rawCandle) < 12 {
			return nil, fmt.Errorf("invalid candle data format")
		}

		closePrice, err := strconv.ParseFloat(rawCandle[4].(string), 64)
		if err != nil {
			return nil, fmt.Errorf("error converting close price to float64: %v", err)
		}
		closePrices[i] = closePrice
	}

	return closePrices, nil
}

type Position struct {
	position string
}

func (s *SignalService) signalEMA(symbol, timeframe string, ema int) (*Position, error) {
	key := fmt.Sprintf("%s/%s/EMA", symbol, timeframe)

	result, err := fetch(symbol, timeframe, ema*2)
	if err != nil {
		return nil, err
	}

	prevEMA, err := s.calculateEMA(result[:len(result)-1], ema)
	if err != nil {
		return nil, err
	}

	posi, err := s.saveSymbol(symbol, key, ema)
	if err != nil {
		return nil, err
	}

	if posi == nil {
		fmt.Println("posi is nil")
		return nil, fmt.Errorf("posi is nil")
	}

	if prevEMA.ema > prevEMA.lastPrice {
		s.updatePosition(symbol, "Short", "EMA", key)
		if *posi.Position == "Long" {
			return &Position{
				position: "Short",
			}, nil
		}
	} else if prevEMA.lastPrice > prevEMA.ema {
		s.updatePosition(symbol, "Long", "EMA", key)
		if *posi.Position == "Short" {
			return &Position{
				position: "Long",
			}, nil
		}
	}
	return nil, nil
	// return &Position{
	// 	position: "Short",
	// }, nil
}

// func (s *SignalService) signalCDC(symbol, timeframe string) (*Position, error) {
// 	key := fmt.Sprintf("%s/%s/CDC", symbol, timeframe)

// 	limit := 52
// 	result, err := fetch(symbol, timeframe, limit)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ema12, err := s.calculateEMA(result, 12)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ema26, err := s.calculateEMA(result, 26)
// 	if err != nil {
// 		return nil, err
// 	}

// 	posi, err := s.saveSymbol(symbol, key)
// 	if err != nil {
// 		return nil, err
// 	}

// 	//Short
// 	if ema26.ema > ema12.ema {
// 		s.updatePosition(symbol, "Long", "CDC", key)
// 		if posi.Position != nil && *posi.Position == "Short" {
// 			return &Position{
// 				position: "Short",
// 			}, nil
// 		}
// 	} else if ema12.ema > ema26.ema {
// 		s.updatePosition(symbol, "Short", "CDC", key)
// 		if posi.Position != nil && *posi.Position == "Long" {
// 			return &Position{
// 				position: "Long",
// 			}, nil
// 		}
// 	}

// 	return nil, nil
// }

func (s *SignalService) updatePosition(symbol, position, types, key string) {
	// สร้าง struct Symbol โดยกำหนดค่า symbol, types, และ position
	symbolData := Symbol{
		Symbol:   symbol,
		Types:    &types,
		Position: &position,
	}

	// แปลง struct Symbol เป็น JSON
	result, err := json.Marshal(symbolData)
	if err != nil {
		fmt.Printf("Error marshaling symbol data: %v\n", err)
		return
	}

	// อัปเดตข้อมูลใน LevelDB
	err = s.levelDB.Update(key, string(result))
	if err != nil {
		fmt.Printf("Error updating data: %v\n", err)
	}
}

type Symbol struct {
	Symbol    string  `json:"symbol"`
	Types     *string `json:"types,omitempty"`
	Position  *string `json:"position,omitempty"`
	Ema       *int    `json:"ema,omitempty"`
	Timeframe *string `json:"timeframe,omitempty"`
}

func (s *SignalService) saveSymbol(symbol, key string, ema ...int) (*Symbol, error) {
	var emaValue *int

	if len(ema) > 0 {
		emaValue = &ema[0]
	} else {
		defaultEma := 0
		emaValue = &defaultEma
	}

	result, err := s.levelDB.Read(key)
	if err != nil {
		if err.Error() == "leveldb: not found" {
			var conJson []byte
			if emaValue != nil && *emaValue >= 0 {
				conJson, _ = json.Marshal(Symbol{
					Symbol: symbol,
					Ema:    emaValue,
					// Timeframe: &timeframe,
				})
			} else {
				conJson, _ = json.Marshal(Symbol{
					Symbol: symbol,
					// Timeframe: &timeframe,
				})
			}

			err = s.levelDB.Create(key, string(conJson))
			if err != nil {
				return nil, err
			}

			return &Symbol{
				Symbol: symbol,
				Ema:    emaValue,
				// Timeframe: &timeframe,
			}, nil
		}
		return nil, err
	}

	var jsonData struct {
		Symbol   string `json:"symbol"`
		Types    string `json:"types"`
		Position string `json:"position"`
	}

	// แปลง JSON string ให้เป็น struct
	err = json.Unmarshal([]byte(result), &jsonData)
	if err != nil {
		return nil, err
	}

	return &Symbol{
		Symbol:   jsonData.Symbol,
		Ema:      emaValue,
		Position: &jsonData.Position,
		Types:    &jsonData.Types,
	}, nil
}
