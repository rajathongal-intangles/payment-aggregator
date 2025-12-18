package grpc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/rajathongal-intangles/payment-aggregator/go-service/pb"
)

// PaymentServer implements pb.PaymentServiceServer
type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer // Required embed

	// In-memory storage (later: fed by Kafka)
	mu       sync.RWMutex
	payments map[string]*pb.Payment
}

// NewPaymentServer creates a new server instance
func NewPaymentServer() *PaymentServer {
	return &PaymentServer{
		payments: make(map[string]*pb.Payment),
	}
}

// AddPayment adds a payment to storage (called by Kafka consumer later)
func (s *PaymentServer) AddPayment(p *pb.Payment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments[p.Id] = p
	log.Printf("[STORE] Added payment: %s", p.Id)
}

// ============================================
// gRPC Method Implementations
// ============================================

// GetPayment returns a single payment by ID
func (s *PaymentServer) GetPayment(
	ctx context.Context,
	req *pb.GetPaymentRequest,
) (*pb.Payment, error) {
	log.Printf("[RPC] GetPayment called: %s", req.PaymentId)

	// Validate request
	if req.PaymentId == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_id is required")
	}

	// Look up payment
	s.mu.RLock()
	payment, exists := s.payments[req.PaymentId]
	s.mu.RUnlock()

	if !exists {
		return nil, status.Errorf(codes.NotFound, "payment not found: %s", req.PaymentId)
	}

	return payment, nil
}

// ListPayments returns filtered list of payments
func (s *PaymentServer) ListPayments(
	ctx context.Context,
	req *pb.ListPaymentsRequest,
) (*pb.PaymentList, error) {
	log.Printf("[RPC] ListPayments called: provider=%v, status=%v, limit=%d",
		req.Provider, req.Status, req.Limit)

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Set default limit
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// Filter and collect payments
	var result []*pb.Payment
	for _, p := range s.payments {
		// Apply filters
		if req.Provider != pb.Provider_PROVIDER_UNKNOWN && p.Provider != req.Provider {
			continue
		}
		if req.Status != pb.PaymentStatus_STATUS_UNKNOWN && p.Status != req.Status {
			continue
		}

		result = append(result, p)

		if len(result) >= limit {
			break
		}
	}

	return &pb.PaymentList{
		Payments:   result,
		TotalCount: int32(len(result)),
		NextCursor: "", // Simplified: no pagination yet
	}, nil
}

// StreamPayments sends real-time payment events to client
func (s *PaymentServer) StreamPayments(
	req *pb.ListPaymentsRequest,
	stream pb.PaymentService_StreamPaymentsServer,
) error {
	log.Printf("[RPC] StreamPayments started: provider=%v", req.Provider)

	// For now, send existing payments as events
	// Later: this will be connected to Kafka consumer
	s.mu.RLock()
	payments := make([]*pb.Payment, 0, len(s.payments))
	for _, p := range s.payments {
		payments = append(payments, p)
	}
	s.mu.RUnlock()

	for _, p := range payments {
		// Apply filter
		if req.Provider != pb.Provider_PROVIDER_UNKNOWN && p.Provider != req.Provider {
			continue
		}

		event := &pb.PaymentEvent{
			Payment:   p,
			EventType: "existing",
		}

		if err := stream.Send(event); err != nil {
			return status.Errorf(codes.Internal, "failed to send: %v", err)
		}

		// Small delay to simulate real-time
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[RPC] StreamPayments completed")
	return nil
}

// ============================================
// Helper: Seed with test data
// ============================================

func (s *PaymentServer) SeedTestData() {
	testPayments := []*pb.Payment{
		{
			Id:            "pay_001",
			Provider:      pb.Provider_PROVIDER_STRIPE,
			Amount:        99.99,
			Currency:      "USD",
			Status:        pb.PaymentStatus_STATUS_COMPLETED,
			CustomerEmail: "alice@example.com",
			CreatedAt:     time.Now().Unix(),
			ProcessedAt:   time.Now().Unix(),
			Metadata:      map[string]string{"order_id": "ORD-001"},
		},
		{
			Id:            "pay_002",
			Provider:      pb.Provider_PROVIDER_RAZORPAY,
			Amount:        1500.00,
			Currency:      "INR",
			Status:        pb.PaymentStatus_STATUS_PENDING,
			CustomerEmail: "bob@example.com",
			CreatedAt:     time.Now().Unix(),
			Metadata:      map[string]string{"order_id": "ORD-002"},
		},
		{
			Id:            "pay_003",
			Provider:      pb.Provider_PROVIDER_PAYPAL,
			Amount:        250.00,
			Currency:      "EUR",
			Status:        pb.PaymentStatus_STATUS_COMPLETED,
			CustomerEmail: "carol@example.com",
			CreatedAt:     time.Now().Unix(),
			ProcessedAt:   time.Now().Unix(),
			Metadata:      map[string]string{"order_id": "ORD-003"},
		},
	}

	for _, p := range testPayments {
		s.AddPayment(p)
	}

	fmt.Printf("✅ Seeded %d test payments\n", len(testPayments))
}