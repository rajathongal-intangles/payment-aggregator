package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcserver "github.com/rajathongal-intangles/payment-aggregator/go-service/internal/grpc"
	pb "github.com/rajathongal-intangles/payment-aggregator/go-service/pb"
)

const (
	defaultPort = "50051"
)

func main() {
	// Get port from env or use default
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = defaultPort
	}

	// Create TCP listener
	addr := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create and register our payment server
	paymentServer := grpcserver.NewPaymentServer()
	pb.RegisterPaymentServiceServer(grpcServer, paymentServer)

	// Enable reflection (for grpcurl and debugging)
	reflection.Register(grpcServer)

	// Seed test data (remove in production)
	paymentServer.SeedTestData()

	// Graceful shutdown handling
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("\n⏳ Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	// Start server
	log.Printf("🚀 gRPC server listening on %s", addr)
	log.Println("   Services: PaymentService")
	log.Println("   Methods:  GetPayment, ListPayments, StreamPayments")
	log.Println("\nPress Ctrl+C to stop")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	log.Println("👋 Server stopped")
}