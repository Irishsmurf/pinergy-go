// Command monitor polls the Pinergy balance every 5 minutes and prints an
// alert if the credit is low or the meter is on emergency credit.
//
// Usage:
//
//	PINERGY_EMAIL=user@example.com PINERGY_PASSWORD=secret go run ./examples/monitor/
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	pinergy "github.com/Irishsmurf/pinergy-go"
)

const pollInterval = 5 * time.Minute

func main() {
	email := os.Getenv("PINERGY_EMAIL")
	password := os.Getenv("PINERGY_PASSWORD")
	if email == "" || password == "" {
		log.Fatal("set PINERGY_EMAIL and PINERGY_PASSWORD environment variables")
	}

	client := pinergy.NewClient(
		// Refresh balance every 30 seconds so polling at 5-minute intervals
		// always gets fresh data (cache TTL < poll interval).
		pinergy.WithCacheTTL("/api/balance/", 30*time.Second),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("Logging in as %s...", email)
	if err := client.Login(ctx, email, password); err != nil {
		log.Fatalf("Login: %v", err)
	}
	log.Println("Logged in. Polling every", pollInterval)

	// Poll immediately, then on each tick.
	tick := time.NewTicker(pollInterval)
	defer tick.Stop()

	for {
		poll(ctx, client, email, password)

		select {
		case <-ctx.Done():
			log.Println("Shutting down.")
			return
		case <-tick.C:
		}
	}
}

func poll(ctx context.Context, client *pinergy.Client, email, password string) {
	bal, err := client.GetBalance(ctx)
	if err != nil {
		// Re-authenticate on token expiry.
		if errors.Is(err, pinergy.ErrUnauthorized) {
			log.Println("Token expired, re-authenticating...")
			if loginErr := client.Login(ctx, email, password); loginErr != nil {
				log.Printf("Re-login failed: %v", loginErr)
				return
			}
			bal, err = client.GetBalance(ctx)
		}
		if err != nil {
			log.Printf("GetBalance error: %v", err)
			return
		}
	}

	status := "OK"
	switch {
	case bal.PowerOff:
		status = "⚠️  POWER OFF"
	case bal.EmergencyCredit:
		status = "🔴 EMERGENCY CREDIT"
	case bal.CreditLow:
		status = "🟡 LOW CREDIT"
	}

	fmt.Printf("[%s] Balance: €%.2f | Days: %d | Status: %s\n",
		time.Now().Format("15:04:05"),
		bal.Balance,
		bal.TopUpInDays,
		status,
	)
}
