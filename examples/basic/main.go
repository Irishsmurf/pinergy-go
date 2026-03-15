// Command basic demonstrates how to log in and retrieve the current balance.
//
// Usage:
//
//	PINERGY_EMAIL=user@example.com PINERGY_PASSWORD=secret go run ./examples/basic/
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	pinergy "github.com/Irishsmurf/pinergy-go"
)

func main() {
	email := os.Getenv("PINERGY_EMAIL")
	password := os.Getenv("PINERGY_PASSWORD")
	if email == "" || password == "" {
		log.Fatal("set PINERGY_EMAIL and PINERGY_PASSWORD environment variables")
	}

	client := pinergy.NewClient()
	ctx := context.Background()

	fmt.Printf("Checking email %s...\n", email)
	if err := client.CheckEmail(ctx, email); err != nil {
		log.Fatalf("CheckEmail: %v", err)
	}
	fmt.Println("Email registered.")

	fmt.Println("Logging in...")
	if err := client.Login(ctx, email, password); err != nil {
		log.Fatalf("Login: %v", err)
	}
	fmt.Println("Logged in.")

	bal, err := client.GetBalance(ctx)
	if err != nil {
		log.Fatalf("GetBalance: %v", err)
	}

	fmt.Printf("\nCurrent balance:   €%.2f\n", bal.Balance)
	fmt.Printf("Days remaining:    %d\n", bal.TopUpInDays)
	fmt.Printf("Last top-up:       €%.2f on %s\n",
		bal.LastTopUpAmount,
		bal.LastTopUpTime.Format("2 Jan 2006"))
	fmt.Printf("Credit low:        %v\n", bal.CreditLow)
	fmt.Printf("Emergency credit:  %v\n", bal.EmergencyCredit)
	fmt.Printf("Power off:         %v\n", bal.PowerOff)
}
