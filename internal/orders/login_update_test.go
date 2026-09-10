package orders

import (
	"context"
	"errors"
	"testing"
)

type verifierStub struct{ err error }

func (v verifierStub) VerifyCode(context.Context, string, string, string) error { return v.err }

func TestVerifyAndReadOrder(t *testing.T) {
	tests := []struct {
		name       string
		verifyErr  error
		order      Order
		wantUpdate string
		wantErr    bool
	}{
		{name: "checkout", order: Order{ID: "ord-41", Phase: Checkout}, wantUpdate: "Checkout confirmed; fulfillment is next."},
		{name: "shipment tracking", order: Order{ID: "ord-42", Phase: Fulfillment, TrackingRef: "PKG-9"}, wantUpdate: "Order shipped; tracking reference: PKG-9"},
		{name: "receipt", order: Order{ID: "ord-43", Phase: Receipt, ReceiptURL: "https://shop.example/receipts/43"}, wantUpdate: "Payment received; receipt: https://shop.example/receipts/43"},
		{name: "verification rejected", verifyErr: errors.New("code rejected"), order: Order{ID: "ord-44", Phase: Checkout}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLoginService(verifierStub{err: tt.verifyErr})
			got, err := service.VerifyAndReadOrder(context.Background(), "+15550100100", "246810", "login-7", tt.order)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Update != tt.wantUpdate {
				t.Fatalf("update = %q, want %q", got.Update, tt.wantUpdate)
			}
		})
	}
}
