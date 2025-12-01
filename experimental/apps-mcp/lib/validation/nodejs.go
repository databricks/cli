package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/log"
)

// ValidationNodeJs implements validation for Node.js-based projects using build, type check, and tests.
type ValidationNodeJs struct{}

// PackageJSON represents package.json structure
type PackageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

// readPackageJSON reads and parses package.json
func readPackageJSON(workDir string) (*PackageJSON, error) {
	pkgPath := filepath.Join(workDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return &pkg, nil
}

type validationStep struct {
	name        string
	command     string
	errorPrefix string
	displayName string
}

func (v *ValidationNodeJs) Validate(ctx context.Context, workDir string) (*ValidateResult, error) {
	log.Info(ctx, "Starting Node.js validation")
	startTime := time.Now()
	var progressLog []string

	progressLog = append(progressLog, "🔄 Starting Node.js validation")

	// Read package.json
	pkg, err := readPackageJSON(workDir)
	if err != nil {
		log.Warnf(ctx, "Could not read package.json: %v. Using defaults.", err)
		progressLog = append(progressLog, "⚠️  Could not read package.json, using defaults")
		pkg = &PackageJSON{Scripts: map[string]string{
			"build": "", "typecheck": "", "test": "",
		}}
	}

	// Build steps based on available scripts
	steps := []validationStep{
		{name: "install", command: "npm install", errorPrefix: "Failed to install dependencies", displayName: "Install"},
	}

	if _, ok := pkg.Scripts["build"]; ok {
		steps = append(steps, validationStep{
			name: "build", command: "npm run build", errorPrefix: "Failed to run build", displayName: "Build",
		})
		progressLog = append(progressLog, "✓ Found 'build' script")
	} else {
		progressLog = append(progressLog, "⚠️  No 'build' script, skipping")
	}

	if _, ok := pkg.Scripts["typecheck"]; ok {
		steps = append(steps, validationStep{
			name: "typecheck", command: "npm run typecheck", errorPrefix: "Failed typecheck", displayName: "Type check",
		})
		progressLog = append(progressLog, "✓ Found 'typecheck' script")
	} else {
		progressLog = append(progressLog, "⚠️  No 'typecheck' script, skipping")
	}

	if _, ok := pkg.Scripts["test"]; ok {
		steps = append(steps, validationStep{
			name: "test", command: "npm run test", errorPrefix: "Failed tests", displayName: "Tests",
		})
		progressLog = append(progressLog, "✓ Found 'test' script")
	} else {
		progressLog = append(progressLog, "⚠️  No 'test' script, skipping")
	}

	// Execute steps
	for i, step := range steps {
		stepNum := fmt.Sprintf("%d/%d", i+1, len(steps))
		log.Infof(ctx, "step %s: running %s...", stepNum, step.name)
		progressLog = append(progressLog, fmt.Sprintf("⏳ Step %s: Running %s...", stepNum, step.displayName))

		stepStart := time.Now()
		err := runCommand(ctx, workDir, step.command)
		if err != nil {
			stepDuration := time.Since(stepStart)
			log.Errorf(ctx, "%s failed (%.1fs)", step.name, stepDuration.Seconds())
			progressLog = append(progressLog, fmt.Sprintf("❌ %s failed (%.1fs)", step.displayName, stepDuration.Seconds()))
			return &ValidateResult{
				Success: false, Message: step.errorPrefix, Details: err, ProgressLog: progressLog,
			}, nil
		}
		stepDuration := time.Since(stepStart)
		log.Infof(ctx, "✓ %s passed: %.1fs", step.name, stepDuration.Seconds())
		progressLog = append(progressLog, fmt.Sprintf("✅ %s passed (%.1fs)", step.displayName, stepDuration.Seconds()))
	}

	totalDuration := time.Since(startTime)
	log.Infof(ctx, "✓ all validation passed: %.1fs", totalDuration.Seconds())
	progressLog = append(progressLog, fmt.Sprintf("✅ All checks passed! Total: %.1fs", totalDuration.Seconds()))

	return &ValidateResult{Success: true, Message: "All validation checks passed", ProgressLog: progressLog}, nil
}
