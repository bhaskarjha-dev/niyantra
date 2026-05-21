package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bhaskarjha-com/niyantra/internal/cursor"
	"github.com/bhaskarjha-com/niyantra/internal/store"
	_ "modernc.org/sqlite"
)

func cursorDataDir() string {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			home, _ := os.UserHomeDir()
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appdata, "Cursor")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "Cursor")
	default:
		home, _ := os.UserHomeDir()
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "Cursor")
	}
}

func main() {
	fmt.Println("================================================================================")
	fmt.Println("Niyantra Cursor Diagnostics and Test Utility")
	fmt.Println("================================================================================")

	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		userProfile = os.Getenv("HOME")
	}
	dbPath := filepath.Join(userProfile, ".niyantra", "niyantra.db")

	var db *store.Store
	var err error
	if _, errStat := os.Stat(dbPath); errStat == nil {
		db, err = store.Open(dbPath)
		if err != nil {
			fmt.Printf("Warning: Error opening Niyantra db: %v\n", err)
		} else {
			defer db.Close()
			fmt.Printf("Opened Niyantra DB at: %s\n", dbPath)
		}
	} else {
		fmt.Printf("Niyantra DB not found at: %s (will skip DB snapshot querying)\n", dbPath)
	}

	// 1. Diagnose local state.vscdb
	fmt.Println("\n--- 1. Local state.vscdb Diagnostics ---")
	dataDir := cursorDataDir()
	vscdbPath := filepath.Join(dataDir, "User", "globalStorage", "state.vscdb")
	if _, errStat := os.Stat(vscdbPath); errStat == nil {
		fmt.Printf("Reading state.vscdb at %s...\n", vscdbPath)
		vscdb, errVsc := sql.Open("sqlite", vscdbPath+"?mode=ro&immutable=1")
		if errVsc != nil {
			fmt.Printf("Error opening state.vscdb: %v\n", errVsc)
		} else {
			defer vscdb.Close()
			rows, errQ := vscdb.Query("SELECT key, value FROM ItemTable WHERE key LIKE 'cursorAuth/%'")
			if errQ != nil {
				fmt.Printf("Query error on state.vscdb: %v\n", errQ)
			} else {
				defer rows.Close()
				for rows.Next() {
					var key, val string
					if errScan := rows.Scan(&key, &val); errScan == nil {
						if len(val) > 40 {
							fmt.Printf("  Key: %s -> Length: %d (Value starts with: %s...)\n", key, len(val), val[:20])
						} else {
							fmt.Printf("  Key: %s -> %s\n", key, val)
						}
					}
				}
			}
		}
	} else {
		fmt.Printf("state.vscdb not found at: %s\n", vscdbPath)
	}

	// 2. Detect credentials
	fmt.Println("\n--- 2. Credentials Detection ---")
	var manualToken string
	if db != nil {
		manualToken = db.GetConfig("cursor_session_token")
		fmt.Printf("Cursor Session Token override configured: %v\n", manualToken != "")
	}
	creds, err := cursor.DetectCredentials(nil, manualToken)
	if err != nil {
		fmt.Printf("Error detecting credentials: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Detected Credentials:\n")
	fmt.Printf("  UserID:               %s\n", creds.UserID)
	fmt.Printf("  Email:                %s\n", creds.Email)
	fmt.Printf("  Source:               %s\n", creds.Source)
	fmt.Printf("  StripeMembershipType: %s\n", creds.StripeMembershipType)

	// 3. Call remote APIs
	fmt.Println("\n--- 3. Live Remote API Calls ---")
	if creds.UserID != "" {
		u1 := "https://www.cursor.com/api/usage?user=" + url.QueryEscape(creds.UserID)
		req1, _ := http.NewRequest(http.MethodGet, u1, nil)
		req1.Header.Set("Cookie", creds.SessionCookie())
		req1.Header.Set("User-Agent", "Mozilla/5.0 (Niyantra)")
		resp1, err1 := http.DefaultClient.Do(req1)
		if err1 != nil {
			fmt.Printf("Legacy API error: %v\n", err1)
		} else {
			defer resp1.Body.Close()
			b, _ := io.ReadAll(resp1.Body)
			fmt.Printf("Legacy /api/usage Status=%d\nResponse: %s\n\n", resp1.StatusCode, string(b))
		}
	} else {
		fmt.Println("Skipping Legacy API call (UserID not found)")
	}

	if creds.AccessToken != "" {
		req2, _ := http.NewRequest(http.MethodPost, "https://api2.cursor.sh/aiserver.v1.DashboardService/GetCurrentPeriodUsage", strings.NewReader("{}"))
		req2.Header.Set("Authorization", "Bearer "+creds.AccessToken)
		req2.Header.Set("Connect-Protocol-Version", "1")
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("User-Agent", "Mozilla/5.0 (Niyantra)")
		resp2, err2 := http.DefaultClient.Do(req2)
		if err2 != nil {
			fmt.Printf("DashboardService API error: %v\n", err2)
		} else {
			defer resp2.Body.Close()
			b, _ := io.ReadAll(resp2.Body)
			fmt.Printf("Connect/DashboardService Status=%d\nResponse: %s\n\n", resp2.StatusCode, string(b))
		}
	} else {
		fmt.Println("Skipping DashboardService API call (AccessToken not found)")
	}

	if creds.AccessToken != "" {
		req3, _ := http.NewRequest(http.MethodGet, "https://www.cursor.com/api/auth/stripe", nil)
		req3.Header.Set("Cookie", creds.SessionCookie())
		req3.Header.Set("User-Agent", "Mozilla/5.0 (Niyantra)")
		resp3, err3 := http.DefaultClient.Do(req3)
		if err3 != nil {
			fmt.Printf("Stripe API error: %v\n", err3)
		} else {
			defer resp3.Body.Close()
			b, _ := io.ReadAll(resp3.Body)
			fmt.Printf("Stripe /api/auth/stripe Status=%d\nResponse: %s\n\n", resp3.StatusCode, string(b))
		}
	} else {
		fmt.Println("Skipping Stripe API call (SessionCookie not constructed)")
	}

	// 4. Fetch unified snapshot using client logic
	fmt.Println("\n--- 4. Client Snapshot Fetching Result ---")
	client := cursor.NewClient(creds, nil)
	snap, err := client.FetchSnapshot(context.Background())
	if err != nil {
		fmt.Printf("FetchSnapshot error: %v\n", err)
	} else {
		snapJSON, _ := json.MarshalIndent(snap, "", "  ")
		fmt.Printf("FetchSnapshot returned Snapshot:\n%s\n", string(snapJSON))
	}

	// 5. Query stored snapshots in Niyantra DB
	if db != nil {
		fmt.Println("\n--- 5. Stored Cursor Snapshots in Niyantra DB ---")
		snaps, errSnaps := db.LatestCursorSnapshots()
		if errSnaps != nil {
			fmt.Printf("Error fetching latest cursor snaps: %v\n", errSnaps)
		} else if len(snaps) == 0 {
			fmt.Println("No Cursor snapshots found in Niyantra database.")
		} else {
			for i, s := range snaps {
				b, _ := json.MarshalIndent(s, "", "  ")
				fmt.Printf("Snapshot %d:\n%s\n", i+1, string(b))
			}
		}
	}
	fmt.Println("================================================================================")
}
