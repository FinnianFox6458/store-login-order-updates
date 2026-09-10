package orders

import (
	"context"
	"errors"
	"fmt"
)

type CodeVerifier interface {
	VerifyCode(context.Context, string, string, string) error
}

type Phase string

const (
	Checkout    Phase = "checkout"
	Fulfillment Phase = "fulfillment"
	Receipt     Phase = "receipt"
)

type Order struct {
	ID          string `json:"id"`
	Phase       Phase  `json:"phase"`
	ReceiptURL  string `json:"receipt_url,omitempty"`
	TrackingRef string `json:"tracking_ref,omitempty"`
}

type LoginResult struct {
	Verified bool   `json:"verified"`
	OrderID  string `json:"order_id"`
	Update   string `json:"update"`
}

type LoginService struct {
	verifier CodeVerifier
}

func NewLoginService(verifier CodeVerifier) *LoginService {
	return &LoginService{verifier: verifier}
}

func (s *LoginService) VerifyAndReadOrder(ctx context.Context, phone, code, requestID string, order Order) (LoginResult, error) {
	if phone == "" || code == "" || order.ID == "" {
		return LoginResult{}, errors.New("phone, code, and order.id are required")
	}
	if err := s.verifier.VerifyCode(ctx, phone, code, requestID); err != nil {
		return LoginResult{}, fmt.Errorf("verify login code: %w", err)
	}
	update, err := customerUpdate(order)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Verified: true, OrderID: order.ID, Update: update}, nil
}

func customerUpdate(order Order) (string, error) {
	switch order.Phase {
	case Checkout:
		return "Checkout confirmed; fulfillment is next.", nil
	case Fulfillment:
		if order.TrackingRef == "" {
			return "Order is being prepared for shipment.", nil
		}
		return "Order shipped; tracking reference: " + order.TrackingRef, nil
	case Receipt:
		if order.ReceiptURL == "" {
			return "Payment received; receipt is being prepared.", nil
		}
		return "Payment received; receipt: " + order.ReceiptURL, nil
	default:
		return "", fmt.Errorf("unsupported order phase %q", order.Phase)
	}
}
