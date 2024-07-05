package main

import (
	"fmt"
	"log"
	"net"
	"order-server/domain"
	"order-server/gRPC"
	"order-server/services"

	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"
)

func main() {
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

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic(err)
	}

	orderServer := services.NewOrderServer(db, levelDB)

	gRPC.RegisterOrdersServiceServer(s, orderServer)

	go func() {
		fmt.Println("gRPC service listening on port 50051")
		if err := s.Serve(listener); err != nil {
			panic(err)
		}
	}()

	c := cron.New(cron.WithSeconds())
	// c.AddFunc("0 */5 * * * *", func() {
	// 	// c.AddFunc("*/5 * * * * *", func() {
	// 	time.Sleep(5 * time.Second)
	// 	orderServer.(*services.OrderServer).ProcessOrder()
	// })

	// Start the cron scheduler
	c.Start()

	fmt.Println("Cron job scheduler started")

	// Wait indefinitely
	select {}
}
