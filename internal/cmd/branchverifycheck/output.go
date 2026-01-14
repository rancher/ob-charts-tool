package branchverifycheck

import (
	"encoding/json"
	"fmt"

	log "github.com/rancher/ob-charts-tool/internal/logging"
)

// ProgressPrinter handles progress output during verification.
// When jsonOutput is true, progress messages are suppressed.
type ProgressPrinter struct {
	jsonOutput bool
}

// NewProgressPrinter creates a new progress printer.
func NewProgressPrinter(jsonOutput bool) *ProgressPrinter {
	return &ProgressPrinter{jsonOutput: jsonOutput}
}

// Print prints a progress message without newline (only in human mode).
func (p *ProgressPrinter) Print(msg string) {
	if !p.jsonOutput {
		fmt.Print(msg)
	}
}

// Println prints a progress message with newline (only in human mode).
func (p *ProgressPrinter) Println(msg string) {
	if !p.jsonOutput {
		fmt.Println(msg)
	}
}

// Printf prints a formatted progress message without newline (only in human mode).
func (p *ProgressPrinter) Printf(format string, args ...interface{}) {
	if !p.jsonOutput {
		fmt.Printf(format, args...)
	}
}

// OutputJSON prints the verification result as JSON.
func OutputJSON(result *VerificationResult) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Log.Errorf("Failed to marshal JSON: %v", err)
		return
	}
	fmt.Println(string(data))
}

// OutputHuman prints the verification result in human-readable format.
func OutputHuman(result *VerificationResult, branchName string) {
	fmt.Printf("\n=== Branch Verification Results for '%s' ===\n\n", branchName)

	for _, check := range result.Checks {
		status := "PASS"
		if !check.Passed {
			if check.Critical {
				status = "FAIL"
			} else {
				status = "WARN"
			}
		}

		fmt.Printf("[%s] %s\n", status, check.Name)
		fmt.Printf("  %s\n\n", check.Message)
	}

	// Summary
	passed := 0
	failed := 0
	warnings := 0
	for _, check := range result.Checks {
		if check.Passed {
			passed++
		} else if check.Critical {
			failed++
		} else {
			warnings++
		}
	}

	fmt.Printf("Summary: %d passed, %d failed, %d warnings\n", passed, failed, warnings)

	if failed > 0 {
		fmt.Println("\nVerification FAILED - Critical issues found")
	} else if warnings > 0 {
		fmt.Println("\nVerification PASSED with warnings")
	} else {
		fmt.Println("\nVerification PASSED - All checks successful")
	}
}
