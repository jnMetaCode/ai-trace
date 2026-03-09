package main

import (
	"fmt"
	"os"

	"github.com/ai-trace/verifier/pkg/verify"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	// Flags
	proofFile   string
	certFile    string
	rootHash    string
	verbose     bool
	jsonOutput  bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ai-trace-verify",
		Short: "AI-Trace Certificate Verifier",
		Long: `AI-Trace Verifier is an open-source tool to verify AI audit certificates.

It can verify:
- Merkle tree proofs
- Certificate integrity
- Event hash chains

Examples:
  ai-trace-verify --proof proof.json
  ai-trace-verify --cert certificate.json
  ai-trace-verify --root-hash sha256:abc123...`,
		Run: runVerify,
	}

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ai-trace-verify version %s\n", version)
		},
	}

	// Flags
	rootCmd.Flags().StringVarP(&proofFile, "proof", "p", "", "Path to proof JSON file")
	rootCmd.Flags().StringVarP(&certFile, "cert", "c", "", "Path to certificate JSON file")
	rootCmd.Flags().StringVarP(&rootHash, "root-hash", "r", "", "Root hash to verify")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runVerify(cmd *cobra.Command, args []string) {
	if proofFile == "" && certFile == "" && rootHash == "" {
		cmd.Help()
		return
	}

	verifier := verify.NewVerifier(verbose)
	var result *verify.VerifyResult

	if proofFile != "" {
		fmt.Printf("Verifying proof file: %s\n\n", proofFile)
		var err error
		result, err = verifier.VerifyProofFile(proofFile)
		if err != nil {
			printError("Failed to verify proof: %v", err)
			os.Exit(1)
		}
	} else if certFile != "" {
		fmt.Printf("Verifying certificate file: %s\n\n", certFile)
		var err error
		result, err = verifier.VerifyCertFile(certFile)
		if err != nil {
			printError("Failed to verify certificate: %v", err)
			os.Exit(1)
		}
	}

	if result == nil {
		printError("No verification performed")
		os.Exit(1)
	}

	if jsonOutput {
		printJSONResult(result)
	} else {
		printResult(result)
	}

	if !result.Valid {
		os.Exit(1)
	}
}

func printResult(result *verify.VerifyResult) {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("                    VERIFICATION RESULT                    ")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Overall result
	if result.Valid {
		green.Println("  ✓ VERIFICATION PASSED")
	} else {
		red.Println("  ✗ VERIFICATION FAILED")
	}
	fmt.Println()

	// Certificate info
	if result.CertID != "" {
		cyan.Println("Certificate Information:")
		fmt.Printf("  Cert ID:    %s\n", result.CertID)
		fmt.Printf("  Root Hash:  %s\n", result.RootHash)
		fmt.Printf("  Events:     %d\n", result.EventCount)
		fmt.Println()
	}

	// Checks
	cyan.Println("Verification Checks:")
	for _, check := range result.Checks {
		if check.Passed {
			green.Printf("  ✓ ")
		} else {
			red.Printf("  ✗ ")
		}
		fmt.Printf("%s", check.Name)
		if check.Message != "" {
			yellow.Printf(" - %s", check.Message)
		}
		fmt.Println()
	}
	fmt.Println()

	fmt.Println("═══════════════════════════════════════════════════════════")
}

func printJSONResult(result *verify.VerifyResult) {
	fmt.Printf(`{
  "valid": %v,
  "cert_id": "%s",
  "root_hash": "%s",
  "event_count": %d,
  "checks": [`,
		result.Valid, result.CertID, result.RootHash, result.EventCount)

	for i, check := range result.Checks {
		fmt.Printf(`
    {"name": "%s", "passed": %v, "message": "%s"}`,
			check.Name, check.Passed, check.Message)
		if i < len(result.Checks)-1 {
			fmt.Print(",")
		}
	}
	fmt.Println(`
  ]
}`)
}

func printError(format string, args ...interface{}) {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
