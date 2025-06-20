package handlers

import (
	"encoding/json"
	"mock-otp-service/events"
	"mock-otp-service/store"
	"net/http"
	"time"
)

type Handler struct {
	store  store.OTPStore
	broker *events.Broker
	ttl    time.Duration
}

// Constructor
func New(store store.OTPStore, broker *events.Broker, ttl time.Duration) *Handler {
	return &Handler{store: store, broker: broker, ttl: ttl}
}

func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User string `json:"user"`
	}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil || req.User == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	code := store.GenerateCode()
	h.store.Set(req.User, code, h.ttl)

	h.broker.Publish(events.Event{
		Type: "otp_requested",
		Data: map[string]string{
			"user": req.User,
			"code": code,
		},
	})

	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.Encode(map[string]string{
		"message": "OTP generated successfully",
		"code":    code,
	})
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User string `json:"user"`
		Code string `json:"code"`
	}

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	stored, err := h.store.Get(req.User)

	switch err {
	case store.ErrCodeExpired:
		http.Error(w, "OTP Expired", http.StatusGone)
		return

	case store.ErrNotFound:
		http.Error(w, "No OTP available for user", http.StatusNotFound)
		return

	case nil:
		// continue below
	default:
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if req.Code != stored {
		http.Error(w, "Invalid OTP", http.StatusUnauthorized)
		return
	}

	// Delete immediately upon successful use
	h.store.Delete(req.User)
	h.broker.Publish(events.Event{
		Type: "otp_verified",
		Data: map[string]string{
			"user": req.User,
		},
	})
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.Encode(map[string]string{"Message": "OTP Verified Successfully!"})
}
