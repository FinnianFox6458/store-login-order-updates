package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/infrai-examples/store-login-order-updates/internal/infrai"
	"github.com/infrai-examples/store-login-order-updates/internal/orders"
)

type server struct {
	sms   *infrai.SMSOTPClient
	login *orders.LoginService
}

type codeRequest struct {
	Phone     string `json:"phone"`
	RequestID string `json:"request_id"`
}

type verifyRequest struct {
	Phone     string       `json:"phone"`
	Code      string       `json:"code"`
	RequestID string       `json:"request_id"`
	Order     orders.Order `json:"order"`
}

func main() {
	client, err := infrai.NewSMSOTPClient(infrai.DefaultBaseURL, os.Getenv("INFRAI_API_KEY"), nil)
	if err != nil {
		log.Fatal(err)
	}
	s := &server{sms: client, login: orders.NewLoginService(client)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code/request", s.requestCode)
	mux.HandleFunc("POST /login/code/verify", s.verifyCode)
	log.Printf("store login listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *server) requestCode(w http.ResponseWriter, r *http.Request) {
	var input codeRequest
	if err := decode(r, &input); err != nil || input.Phone == "" {
		http.Error(w, "valid phone and request_id required", http.StatusBadRequest)
		return
	}
	if err := s.sms.RequestCode(r.Context(), input.Phone, input.RequestID); err != nil {
		log.Printf("request code: %v", err)
		http.Error(w, "code request failed", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "code_sent"})
}

func (s *server) verifyCode(w http.ResponseWriter, r *http.Request) {
	var input verifyRequest
	if err := decode(r, &input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	result, err := s.login.VerifyAndReadOrder(r.Context(), input.Phone, input.Code, input.RequestID, input.Order)
	if err != nil {
		log.Printf("verify login: %v", err)
		http.Error(w, "verification failed", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
