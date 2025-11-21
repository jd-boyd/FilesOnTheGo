package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// TestEvent represents a Go test JSON output event
type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Output  string    `json:"Output"`
	Elapsed *float64  `json:"Elapsed"`
}

// TestSummary contains aggregated test results
type TestSummary struct {
	Packages      map[string]*PackageSummary
	TotalTests    int
	PassedTests   int
	FailedTests   int
	SkippedTests  int
	TotalTime     time.Duration
	FailedPackages []string
}

// PackageSummary contains test results for a single package
type PackageSummary struct {
	Name         string
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Time         time.Duration
	Tests        map[string]*TestResult
	Failed       bool
	Output       []string
}

// TestResult represents a single test's outcome
type TestResult struct {
	Name    string
	Status  string // "pass", "fail", "skip"
	Time    time.Duration
	Output  []string
}

func main() {
	summary := &TestSummary{
		Packages: make(map[string]*PackageSummary),
	}

	decoder := json.NewDecoder(os.Stdin)

	for {
		var event TestEvent
		err := decoder.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			continue
		}

		processEvent(summary, &event)
	}

	printSummary(summary)

	// Exit with non-zero code if any tests failed
	if len(summary.FailedPackages) > 0 {
		os.Exit(1)
	}
}

func processEvent(summary *TestSummary, event *TestEvent) {
	pkgName := event.Package

	// Initialize package summary if not exists
	if _, exists := summary.Packages[pkgName]; !exists {
		summary.Packages[pkgName] = &PackageSummary{
			Name:   pkgName,
			Tests:  make(map[string]*TestResult),
			Output: []string{},
		}
	}

	pkg := summary.Packages[pkgName]

	switch event.Action {
	case "run":
		testName := event.Test
		if testName == "" {
			return // Package-level run event
		}

		pkg.Tests[testName] = &TestResult{
			Name:   testName,
			Status: "running",
		}
		pkg.TotalTests++
		summary.TotalTests++

	case "pass":
		testName := event.Test
		if testName == "" {
			// Package passed
			pkg.Failed = false
			break
		}

		if test, exists := pkg.Tests[testName]; exists {
			test.Status = "pass"
			if event.Elapsed != nil {
				test.Time = time.Duration(*event.Elapsed * float64(time.Second))
				pkg.Time += test.Time
				summary.TotalTime += test.Time
			}
			pkg.PassedTests++
			summary.PassedTests++
		}

	case "fail":
		testName := event.Test
		if testName == "" {
			// Package failed
			pkg.Failed = true
			if !contains(summary.FailedPackages, pkgName) {
				summary.FailedPackages = append(summary.FailedPackages, pkgName)
			}
			break
		}

		if test, exists := pkg.Tests[testName]; exists {
			test.Status = "fail"
			if event.Elapsed != nil {
				test.Time = time.Duration(*event.Elapsed * float64(time.Second))
				pkg.Time += test.Time
				summary.TotalTime += test.Time
			}
			pkg.FailedTests++
			summary.FailedTests++
			pkg.Failed = true
			if !contains(summary.FailedPackages, pkgName) {
				summary.FailedPackages = append(summary.FailedPackages, pkgName)
			}
		}

	case "skip":
		testName := event.Test
		if testName == "" {
			break
		}

		if test, exists := pkg.Tests[testName]; exists {
			test.Status = "skip"
			pkg.SkippedTests++
			summary.SkippedTests++
		}

	case "output":
		if event.Output != "" {
			pkg.Output = append(pkg.Output, event.Output)

			// Also add to specific test if it exists
			if event.Test != "" {
				if test, exists := pkg.Tests[event.Test]; exists {
					test.Output = append(test.Output, event.Output)
				}
			}
		}
	}
}

func printSummary(summary *TestSummary) {
	fmt.Println()
	fmt.Println("🧪 Test Summary")
	fmt.Println(strings.Repeat("=", 60))

	// Overall summary
	fmt.Printf("📊 Overall Results: %d tests, %d passed, %d failed, %d skipped\n",
		summary.TotalTests, summary.PassedTests, summary.FailedTests, summary.SkippedTests)

	if summary.TotalTime > 0 {
		fmt.Printf("⏱️  Total Time: %v\n", summary.TotalTime.Round(time.Millisecond))
	}

	// Status indicator
	if len(summary.FailedPackages) == 0 {
		fmt.Println("✅ All tests passed!")
	} else {
		fmt.Printf("❌ %d package(s) failed\n", len(summary.FailedPackages))
	}

	fmt.Println()

	// Package details
	fmt.Println("📦 Package Details:")
	fmt.Println(strings.Repeat("-", 60))

	for _, pkg := range summary.Packages {
		status := "✅ PASS"
		if pkg.Failed {
			status = "❌ FAIL"
		}

		fmt.Printf("%s %s\n", status, pkg.Name)
		fmt.Printf("   Tests: %d total, %d passed, %d failed, %d skipped\n",
			pkg.TotalTests, pkg.PassedTests, pkg.FailedTests, pkg.SkippedTests)

		if pkg.Time > 0 {
			fmt.Printf("   Time: %v\n", pkg.Time.Round(time.Millisecond))
		}

		// Show failed tests
		var failedTests []string
		for _, test := range pkg.Tests {
			if test.Status == "fail" {
				failedTests = append(failedTests, test.Name)
			}
		}

		if len(failedTests) > 0 {
			fmt.Printf("   Failed tests: %s\n", strings.Join(failedTests, ", "))
		}

		fmt.Println()
	}

	// Show failures in detail if any
	if len(summary.FailedPackages) > 0 {
		fmt.Println("🚨 Failure Details:")
		fmt.Println(strings.Repeat("-", 60))

		for _, pkgName := range summary.FailedPackages {
			pkg := summary.Packages[pkgName]

			fmt.Printf("❌ %s\n", pkg.Name)

			for _, test := range pkg.Tests {
				if test.Status == "fail" {
					fmt.Printf("   🔴 %s\n", test.Name)

					// Show last few lines of output for this test
					if len(test.Output) > 0 {
						fmt.Println("   Output:")
						for _, line := range test.Output {
							if strings.TrimSpace(line) != "" {
								fmt.Printf("     %s", line)
							}
						}
					}
					fmt.Println()
				}
			}
		}
	}

	// Final status line
	fmt.Println(strings.Repeat("=", 60))
	if len(summary.FailedPackages) == 0 {
		fmt.Println("🎉 All tests completed successfully!")
	} else {
		fmt.Printf("💥 Tests completed with %d failures\n", len(summary.FailedPackages))
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}