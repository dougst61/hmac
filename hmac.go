// Copyright (c) 2026 Doug Stewart

// Feature Token Tool is an interactive command-line application for creating
// and decoding HMAC-signed feature licensing tokens.
//
// The tool provides two modes of operation:
//
//  1. Create mode: Prompts the operator to select which features to enable,
//     enter a user ID, and set an expiry date. Outputs a signed base64 token.
//  2. Decode mode: Prompts for a user ID and a token string. Verifies the
//     token signature, then displays the enabled features, expiry date,
//     and whether the token is currently active or expired.
//
// The application uses the featuretoken package for all cryptographic operations.
// Feature definitions and the HMAC secret key are configured in this file and
// passed to the package at startup.
//
// Usage:
//
//	./featuretoken
//
// The version number is injected at build time via ldflags. See build.sh for
// cross-compilation instructions.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"featuretoken/featuretoken"
)

// timeFormat is the Go reference time layout used for parsing and displaying
// dates in "YYYY-MM-DD HH:MM" format throughout the application.
const timeFormat = "2006-01-02 15:04"

// Version is the application version string, set at build time via:
//
//	go build -ldflags "-X main.Version=1.0.0.2602010000"
//
// If not set (e.g., during development), it defaults to "dev".
var Version = "dev"

// secretKey is the HMAC-SHA256 signing key used by the featuretoken package.
// In production, this should be loaded from secure storage (environment
// variable, secrets manager, encrypted config file, etc.).
var secretKey = []byte("mysecretkey123")

// featureNames maps bit positions (0-31) to human-readable feature names.
// Modify this map to match the features of your application. Not all 32
// positions need to be populated; only mapped positions will appear when
// listing available features or resolving enabled features from a token.
//
// The key is the bit position (0 = least significant bit) and the value
// is the display name shown to the operator.
var featureNames = map[int]string{
	0:  "Feature1",
	1:  "Feature2",
	2:  "Feature3",
	3:  "Feature4",
	4:  "Feature5",
	5:  "Feature6",
	6:  "Feature7",
	7:  "Feature8",
	8:  "Feature9",
	9:  "Feature10",
	10: "Feature11",
	11: "Feature12",
	12: "Feature13",
	13: "Feature14",
	14: "Feature15",
	15: "Feature16",
	16: "Feature17",
	17: "Feature18",
	18: "Feature19",
	19: "Feature20",
	20: "Feature21",
	21: "Feature22",
	22: "Feature23",
	23: "Feature24",
	24: "Feature25",
	25: "Feature26",
	26: "Feature27",
	27: "Feature28",
	28: "Feature29",
	29: "Feature30",
	30: "Feature31",
	31: "Feature32",
}

// tm is the global TokenManager instance, initialized in main() using the
// secret key and feature names defined above.
var tm *featuretoken.TokenManager

// main is the application entry point. It initializes the TokenManager with
// the configured secret key and feature names, then presents an interactive
// menu for the operator to create or decode a token.
//
// Exit codes:
//   - 0: Normal exit.
//   - 1: Failed to initialize the TokenManager (invalid configuration).
func main() {
	// Initialize the TokenManager with our secret key and feature map.
	// This validates the configuration and returns an error if the key
	// is empty or any feature position is out of range.
	var err error
	tm, err = featuretoken.New(secretKey, featureNames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize: %s\n", err)
		os.Exit(1)
	}

	// Create a buffered reader for interactive stdin input.
	reader := bufio.NewReader(os.Stdin)

	// Display the version banner and main menu.
	fmt.Printf("=== Feature Token Tool v%s ===\n", Version)
	fmt.Println("1) Create a token")
	fmt.Println("2) Decode a token")
	fmt.Print("Select option: ")

	// Read the operator's menu selection.
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	// Route to the appropriate flow based on the selection.
	switch choice {
	case "1":
		createFlow(reader)
	case "2":
		decodeFlow(reader)
	default:
		fmt.Println("Invalid option.")
	}
}

// parseExpiry parses a date or date-time string into a time.Time value.
//
// Accepted formats:
//   - "YYYY-MM-DD HH:MM" — Parsed as-is in UTC.
//   - "YYYY-MM-DD"        — Defaults to 23:59 UTC on the given date.
//
// Parameters:
//   - input: The date or date-time string entered by the operator.
//
// Returns:
//   - time.Time: The parsed expiry time in UTC.
//   - error:     Non-nil if the input does not match either accepted format.
//
// Example:
//
//	expiry, err := parseExpiry("2026-12-31")
//	// expiry = 2026-12-31 23:59:00 UTC
//
//	expiry, err := parseExpiry("2026-12-31 14:30")
//	// expiry = 2026-12-31 14:30:00 UTC
func parseExpiry(input string) (time.Time, error) {
	// First, try the full date-time format "YYYY-MM-DD HH:MM".
	expiry, err := time.Parse(timeFormat, input)
	if err == nil {
		return expiry.UTC(), nil
	}

	// Fall back to date-only format "YYYY-MM-DD", defaulting to 23:59 UTC.
	expiry, err = time.Parse("2006-01-02", input)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %s", input)
	}
	return time.Date(expiry.Year(), expiry.Month(), expiry.Day(), 23, 59, 0, 0, time.UTC), nil
}

// createFlow guides the operator through the interactive token creation process.
//
// The flow proceeds as follows:
//  1. Prompt for a user ID (required; the token will be bound to this ID).
//  2. Display the list of available features with their numbered positions.
//  3. Prompt for a comma-separated list of feature numbers to enable.
//  4. Prompt for an expiry date (or date-time).
//  5. Generate the signed token via the featuretoken package.
//  6. Display the token, enabled features, and expiry date.
//
// Parameters:
//   - reader: A buffered reader connected to stdin for interactive input.
//
// Returns:
//   - This function prints results to stdout and returns nothing. Errors
//     are printed to stdout and cause the flow to exit early.
//
// Example session:
//
//	Enter user ID: jdoe
//	Available features:
//	   1) Feature1
//	   2) Feature2
//	  ...
//	Enter feature numbers to enable (comma-separated, e.g. 1,3,5):
//	> 1,3
//	Enter expiry date (YYYY-MM-DD) or date and time (YYYY-MM-DD HH:MM):
//	> 2026-12-31
//	--- Generated Token ---
//	AAAABQAAAABrNuxE... (base64 string)
//	Enabled features:
//	  - Feature1
//	  - Feature3
//	Expires: 2026-12-31 23:59 UTC
func createFlow(reader *bufio.Reader) {
	// Step 1: Prompt for the user ID. This value is mixed into the HMAC
	// signature, binding the token to this specific user.
	fmt.Print("\nEnter user ID: ")
	userID, _ := reader.ReadString('\n')
	userID = strings.TrimSpace(userID)
	if userID == "" {
		fmt.Println("User ID is required. Aborting.")
		return
	}

	// Step 2: Display the numbered list of available features.
	// Features are shown in bit-position order (0-31), numbered 1-32
	// for operator convenience (the display number = bit position + 1).
	fmt.Println("\nAvailable features:")
	for i := 0; i < 32; i++ {
		if name, ok := featureNames[i]; ok {
			fmt.Printf("  %2d) %s\n", i+1, name)
		}
	}

	// Step 3: Read the operator's feature selection as a comma-separated
	// list of numbers (e.g., "1,3,5"). Each number sets the corresponding
	// bit in the features bitfield (number 1 = bit 0, number 2 = bit 1, etc.).
	fmt.Println("\nEnter feature numbers to enable (comma-separated, e.g. 1,3,5):")
	fmt.Print("> ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var features uint32
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		num, err := strconv.Atoi(part)
		if err != nil || num < 1 || num > 32 {
			fmt.Printf("Skipping invalid feature number: %s\n", part)
			continue
		}
		// Convert from 1-based display number to 0-based bit position.
		features |= 1 << uint(num-1)
	}

	if features == 0 {
		fmt.Println("No valid features selected. Aborting.")
		return
	}

	// Step 4: Read and parse the expiry date or date-time.
	fmt.Println("\nEnter expiry date (YYYY-MM-DD) or date and time (YYYY-MM-DD HH:MM):")
	fmt.Println("If no time is provided, defaults to 23:59 UTC.")
	fmt.Print("> ")
	dateInput, _ := reader.ReadString('\n')
	dateInput = strings.TrimSpace(dateInput)

	expiry, err := parseExpiry(dateInput)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Step 5: Generate the signed token via the featuretoken package.
	token, err := tm.CreateToken(userID, features, expiry)
	if err != nil {
		fmt.Printf("Error creating token: %s\n", err)
		return
	}

	// Step 6: Display the generated token and a summary of what was encoded.
	fmt.Println("\n--- Generated Token ---")
	fmt.Println(token)
	fmt.Println("\nEnabled features:")
	for _, name := range tm.EnabledFeatures(features) {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Printf("Expires: %s UTC\n", expiry.Format(timeFormat))
}

// decodeFlow guides the operator through the interactive token decoding process.
//
// The flow proceeds as follows:
//  1. Prompt for the user ID (must match the ID used during token creation).
//  2. Prompt for the base64 token string.
//  3. Verify the token via the featuretoken package.
//  4. Display the enabled features, expiry date, and active/expired status.
//
// Parameters:
//   - reader: A buffered reader connected to stdin for interactive input.
//
// Returns:
//   - This function prints results to stdout and returns nothing. Errors
//     are printed to stdout and cause the flow to exit early.
//
// Example session:
//
//	Enter user ID: jdoe
//	Enter token: AAAABQAAAABrNuxE... (paste token here)
//	--- Decoded Token ---
//	Enabled features:
//	  - Feature1
//	  - Feature3
//	Expires: 2026-12-31 23:59 UTC
//	STATUS: ACTIVE
func decodeFlow(reader *bufio.Reader) {
	// Step 1: Prompt for the user ID. The token will only verify if this
	// matches the user ID that was used during creation.
	fmt.Print("\nEnter user ID: ")
	userID, _ := reader.ReadString('\n')
	userID = strings.TrimSpace(userID)
	if userID == "" {
		fmt.Println("User ID is required. Aborting.")
		return
	}

	// Step 2: Prompt for the base64-encoded token string.
	fmt.Print("Enter token: ")
	tokenStr, _ := reader.ReadString('\n')
	tokenStr = strings.TrimSpace(tokenStr)

	// Step 3: Verify and decode the token. This recomputes the HMAC using
	// the provided user ID and compares it against the stored signature.
	// If the user ID is wrong or the token was tampered with, an error
	// is returned.
	features, expiry, err := tm.DecodeToken(userID, tokenStr)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	// Step 4: Display the decoded token contents and active/expired status.
	fmt.Println("\n--- Decoded Token ---")
	fmt.Println("Enabled features:")
	for _, name := range tm.EnabledFeatures(features) {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Printf("Expires: %s UTC\n", expiry.Format(timeFormat))

	// Compare the current UTC time against the token's expiry to determine
	// whether the licensed features are still active.
	if time.Now().UTC().After(expiry) {
		fmt.Println("STATUS: EXPIRED")
	} else {
		fmt.Println("STATUS: ACTIVE")
	}
}
