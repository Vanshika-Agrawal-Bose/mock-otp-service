package main

import (
	"fmt"
	"log"
	"mock-otp-service/events"
	"mock-otp-service/handlers"
	"mock-otp-service/store"
	"net/http"
	"os"
	"strconv"
	"time"
)

func getTTL() time.Duration {
	ttlStr := os.Getenv("OTP_TTL")
	if ttlStr == "" {
		log.Println("OTP_TTL not set, defaulting to 2 minutes")
		return 2 * time.Minute
	}

	ttlSeconds, err := strconv.Atoi(ttlStr)
	if err != nil {
		log.Printf("Invalid OTP_TTL value '%s', defaulting to 2 minutes\n", ttlStr)
		return 2 * time.Minute
	}

	return time.Duration(ttlSeconds) * time.Second
}

func main() {
	broker := events.NewBroker()

	// Subscribe
	broker.Subscribe("otp_requested", func(e events.Event) {
		log.Printf("[EVENT] OTP requested for %s", e.Data["user"])
	})
	broker.Subscribe("otp_verified", func(e events.Event) {
		log.Printf("[EVENT] OTP verified for %s", e.Data["user"])
	})
	broker.Subscribe("otp_expired", func(e events.Event) {
		log.Printf("[EVENT] OTP expired for %s", e.Data["user"])
	})

	otpStore := store.NewMemoryStore(broker)
	h := handlers.New(otpStore, broker, getTTL())

	http.HandleFunc("/otp/request", h.RequestOTP)
	http.HandleFunc("/otp/verify", h.VerifyOTP)
	addr := ":8080"
	fmt.Printf("OTP service listening on http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
