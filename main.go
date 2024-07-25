package main

import (
	"fmt"
	"log"
	"net"
	"order-server/domain"
	"order-server/services"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	port := os.Getenv("PORT")

	db, err := services.ConnectDB()
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// สร้าง instance ของ LevelDBService
	levelDB, err := services.NewLevelDBService("./db")
	if err != nil {
		panic(err)
	}
	defer levelDB.Close()

	// Migrate the schema
	db.AutoMigrate(&domain.Orders{})

	s := grpc.NewServer()

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		panic(err)
	}

	orderServer := services.NewOrderServer(db, levelDB)

	services.RegisterOrdersServiceServer(s, orderServer)

	go func() {
		fmt.Println("gRPC service listening on port", port)
		if err := s.Serve(listener); err != nil {
			panic(err)
		}
	}()

	c := cron.New(cron.WithSeconds())
	c.AddFunc("0 */5 * * * *", func() {
		now := time.Now()
		fmt.Println("Current time:", now.Format("2006-01-02 15:04:05"))
		time.Sleep(5 * time.Second)
		orderServer.(*services.OrderServer).ProcessOrder("5m")
	})

	c.AddFunc("0 */4 * * *", func() {
		time.Sleep(5 * time.Second)
		orderServer.(*services.OrderServer).ProcessOrder("4h")
	})

	c.AddFunc("0 7 * * *", func() {
		time.Sleep(5 * time.Second)
		orderServer.(*services.OrderServer).ProcessOrder("1d")
	})

	// Start the cron scheduler
	c.Start()

	fmt.Println("Cron job scheduler started")

	// Wait indefinitely
	select {}
}
