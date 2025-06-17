package store

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"mock-otp-service/events"
	"sync"
	"time"
)

var (
	ErrNotFound    = errors.New("OTP not found")
	ErrCodeExpired = errors.New("OTP expired")
)

type OTPStore interface {
	Set(user string, code string, ttl time.Duration)
	Get(user string) (string, error)
	Delete(user string)
}

// sttoring 2 major data points: user and code map & channel of scheduled expiry events
type memoryStore struct {
	mu          sync.RWMutex
	data        map[string]string
	expiryTimes map[string]time.Time // NEW: track deadlines
	expiries    chan expiry
	broker      *events.Broker
}

// to store: expire user's data at particular time
type expiry struct {
	user string
	at   time.Time
}

func NewMemoryStore(broker *events.Broker) OTPStore {
	ms := &memoryStore{
		data:        make(map[string]string),
		expiryTimes: make(map[string]time.Time), // NEW
		expiries:    make(chan expiry, 100),
		broker:      broker,
	}
	// using a go routine to start the process in background
	go ms.startExpiryWatcher()
	return ms
}

func (m *memoryStore) Set(user, code string, ttl time.Duration) {
	deadline := time.Now().Add(ttl)

	// acquires a rw lock
	m.mu.Lock()
	m.data[user] = code
	m.expiryTimes[user] = deadline
	m.mu.Unlock()

	// sending an expiry event into channel with a scheduled ttl
	m.expiries <- expiry{user: user, at: deadline}
}

func (m *memoryStore) Get(user string) (string, error) {
	// acquires a read lock
	m.mu.RLock()
	code, exists := m.data[user]
	deadline, hasDeadline := m.expiryTimes[user]
	m.mu.RUnlock()

	// if no code is found for the user, an error is thrown
	if !exists {
		return "", ErrNotFound
	}
	// if its expired
	if hasDeadline && time.Now().After(deadline) {
		// it’s expired → clean up and signal expiry
		m.Delete(user)
		return "", ErrCodeExpired
	}
	return code, nil
}

func (m *memoryStore) Delete(user string) {
	m.mu.Lock()
	delete(m.data, user)
	m.mu.Unlock()
}

// function that keeps removing expired OTPs from the memory after every 500ms
func (m *memoryStore) startExpiryWatcher() {
	// Creates a ticker firing every 500 ms
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// pending list of scheduled expiries
	pending := []expiry{}

	for {
		select {

		// on every new event in the expiries channel, it appends the entry
		case exp := <-m.expiries:
			log.Printf("Scheduled expiry for user=%s at=%s", exp.user, exp.at.Format(time.RFC3339))
			pending = append(pending, exp)

			// on each ticker, it iterates the pending array to delete expired OTPs
		case now := <-ticker.C:
			var next []expiry
			for _, e := range pending {
				// if current time >= expiry time, delete the user's OTP
				if now.After(e.at) {
					m.Delete(e.user)
					m.broker.Publish(events.Event{
						Type: "otp_expired",
						Data: map[string]string{
							"user": e.user,
						},
					})
				} else {
					next = append(next, e)
				}
			}
			pending = next
		}
	}
}

func GenerateCode() string {
	return fmt.Sprintf("%06d", rand.Intn(1_000_000))
}
