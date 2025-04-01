package instrumentation

import (
	"os"
	"path/filepath"
	"strings"
)

// InstrumentationOptions stores configuration for selective instrumentation
type InstrumentationOptions struct {
	// Enabled indicates whether instrumentation is enabled
	Enabled bool

	// IncludePackages is a list of package paths to instrument
	// Empty means all packages are instrumented
	IncludePackages []string

	// ExcludePackages is a list of package paths to exclude from instrumentation
	// This takes precedence over IncludePackages
	ExcludePackages []string

	// InstrumentStdlib indicates whether to instrument standard library code
	InstrumentStdlib bool
}

// DefaultInstrumentationOptions returns the default instrumentation options
func DefaultInstrumentationOptions() InstrumentationOptions {
	return InstrumentationOptions{
		Enabled:          true,
		IncludePackages:  []string{}, // Empty means all packages
		ExcludePackages:  []string{}, // Don't exclude any packages by default
		InstrumentStdlib: false,      // Don't instrument stdlib by default
	}
}

// Global instrumentation options
var (
	CurrentOptions = loadOptionsFromEnvironment()
)

// loadOptionsFromEnvironment loads instrumentation options from environment variables
func loadOptionsFromEnvironment() InstrumentationOptions {
	options := DefaultInstrumentationOptions()

	// CHRONOGO_ENABLED controls whether instrumentation is enabled
	if enabled := os.Getenv("CHRONOGO_ENABLED"); enabled != "" {
		options.Enabled = enabled == "1" || enabled == "true" || enabled == "yes"
	}

	// CHRONOGO_INSTRUMENT controls which packages to instrument
	if instruments := os.Getenv("CHRONOGO_INSTRUMENT"); instruments != "" {
		options.IncludePackages = strings.Split(instruments, ",")
		for i, pkg := range options.IncludePackages {
			options.IncludePackages[i] = strings.TrimSpace(pkg)
		}
	}

	// CHRONOGO_EXCLUDE controls which packages to exclude
	if excludes := os.Getenv("CHRONOGO_EXCLUDE"); excludes != "" {
		options.ExcludePackages = strings.Split(excludes, ",")
		for i, pkg := range options.ExcludePackages {
			options.ExcludePackages[i] = strings.TrimSpace(pkg)
		}
	}

	// CHRONOGO_INSTRUMENT_STDLIB controls whether to instrument standard library
	if instrumentStdlib := os.Getenv("CHRONOGO_INSTRUMENT_STDLIB"); instrumentStdlib != "" {
		options.InstrumentStdlib = instrumentStdlib == "1" || instrumentStdlib == "true" || instrumentStdlib == "yes"
	}

	return options
}

// ShouldInstrument checks if a package should be instrumented
func ShouldInstrument(packagePath string) bool {
	if !CurrentOptions.Enabled {
		return false
	}

	// Check if package is part of the standard library
	isStdlib := !strings.Contains(packagePath, ".")
	if isStdlib && !CurrentOptions.InstrumentStdlib {
		return false
	}

	// Check if package is explicitly excluded
	for _, exclude := range CurrentOptions.ExcludePackages {
		if matchesPackagePath(packagePath, exclude) {
			return false
		}
	}

	// If no includes specified, instrument everything except exclusions
	if len(CurrentOptions.IncludePackages) == 0 {
		return true
	}

	// Check if package is explicitly included
	for _, include := range CurrentOptions.IncludePackages {
		if matchesPackagePath(packagePath, include) {
			return true
		}
	}

	return false
}

// matchesPackagePath checks if a package matches a pattern
func matchesPackagePath(packagePath, pattern string) bool {
	// Handle wildcard patterns
	if strings.HasSuffix(pattern, "...") {
		prefix := strings.TrimSuffix(pattern, "...")
		return strings.HasPrefix(packagePath, prefix)
	}

	// Direct match
	matched, _ := filepath.Match(pattern, packagePath)
	return matched
}

// SetInstrumentationOptions sets the current instrumentation options
func SetInstrumentationOptions(options InstrumentationOptions) {
	CurrentOptions = options
}
