package main

import (
	"fmt"
	"order-server/domain"
	"order-server/services"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	db, err := services.ConnectDB()
	if err != nil {
		panic(err)
	}

	// สร้าง instance ของ LevelDBService
	levelDB, err := services.NewLevelDBService("./db")
	if err != nil {
		panic(err)
	}
	defer levelDB.Close()

	// Migrate the schema
	db.AutoMigrate(&domain.Orders{})

	orderServer := services.NewOrderServer(db, levelDB)

	// orderServer.ProcessOrder("5m")
	c := cron.New(cron.WithSeconds())
	c.AddFunc("0 */5 * * * *", func() {
		now := time.Now()
		fmt.Println("Current time:", now.Format("2006-01-02 15:04:05"))
		time.Sleep(5 * time.Second)
		orderServer.ProcessOrder("5m")
	})

	c.AddFunc("0 */4 * * *", func() {
		time.Sleep(5 * time.Second)
		orderServer.ProcessOrder("4h")
	})

	c.AddFunc("0 7 * * *", func() {
		time.Sleep(5 * time.Second)
		orderServer.ProcessOrder("1d")
	})

	// Start the cron scheduler
	c.Start()

	fmt.Println("Cron job scheduler started")

	// Wait indefinitely
	select {}
}
