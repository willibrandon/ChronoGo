package instrumentation

import (
	"os"
	"testing"
)

func TestShouldInstrument(t *testing.T) {
	// Save current options and restore at end of test
	originalOptions := CurrentOptions
	defer func() {
		CurrentOptions = originalOptions
	}()

	tests := []struct {
		name             string
		options          InstrumentationOptions
		packagePath      string
		shouldInstrument bool
	}{
		{
			name: "all packages enabled",
			options: InstrumentationOptions{
				Enabled:          true,
				IncludePackages:  []string{},
				ExcludePackages:  []string{},
				InstrumentStdlib: false,
			},
			packagePath:      "github.com/willibrandon/ChronoGo/pkg/test",
			shouldInstrument: true,
		},
		{
			name: "disabled instrumentation",
			options: InstrumentationOptions{
				Enabled:          false,
				IncludePackages:  []string{},
				ExcludePackages:  []string{},
				InstrumentStdlib: false,
			},
			packagePath:      "github.com/willibrandon/ChronoGo/pkg/test",
			shouldInstrument: false,
		},
		{
			name: "specific package included",
			options: InstrumentationOptions{
				Enabled:          true,
				IncludePackages:  []string{"github.com/willibrandon/ChronoGo/pkg/test"},
				ExcludePackages:  []string{},
				InstrumentStdlib: false,
			},
			packagePath:      "github.com/willibrandon/ChronoGo/pkg/test",
			shouldInstrument: true,
		},
		{
			name: "specific package excluded",
			options: InstrumentationOptions{
				Enabled:          true,
				IncludePackages:  []string{},
				ExcludePackages:  []string{"github.com/willibrandon/ChronoGo/pkg/test"},
				InstrumentStdlib: false,
			},
			packagePath:      "github.com/willibrandon/ChronoGo/pkg/test",
			shouldInstrument: false,
		},
		{
			name: "wildcard include",
			options: InstrumentationOptions{
				Enabled:          true,
				IncludePackages:  []string{"github.com/willibrandon/ChronoGo/pkg/..."},
				ExcludePackages:  []string{},
				InstrumentStdlib: false,
			},
			packagePath:      "github.com/willibrandon/ChronoGo/pkg/test",
			shouldInstrument: true,
		},
		{
			name: "wildcard exclude",
			options: InstrumentationOptions{
				Enabled:          true,
				IncludePackages:  []string{},
				ExcludePackages:  []string{"github.com/willibrandon/ChronoGo/pkg/..."},
				InstrumentStdlib: false,
			},
			packagePath:      "github.com/willibrandon/ChronoGo/pkg/test",
			shouldInstrument: false,
		},
		{
			name: "stdlib not instrumented by default",
			options: InstrumentationOptions{
				Enabled:          true,
				IncludePackages:  []string{},
				ExcludePackages:  []string{},
				InstrumentStdlib: false,
			},
			packagePath:      "fmt",
			shouldInstrument: false,
		},
		{
			name: "stdlib instrumented when enabled",
			options: InstrumentationOptions{
				Enabled:          true,
				IncludePackages:  []string{},
				ExcludePackages:  []string{},
				InstrumentStdlib: true,
			},
			packagePath:      "fmt",
			shouldInstrument: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set options for this test
			CurrentOptions = tt.options

			// Check if should instrument
			result := ShouldInstrument(tt.packagePath)

			// Verify result
			if result != tt.shouldInstrument {
				t.Errorf("ShouldInstrument(%q) = %v, want %v",
					tt.packagePath, result, tt.shouldInstrument)
			}
		})
	}
}

func TestLoadOptionsFromEnvironment(t *testing.T) {
	// Save original environment values
	origEnabled := os.Getenv("CHRONOGO_ENABLED")
	origInstrument := os.Getenv("CHRONOGO_INSTRUMENT")
	origExclude := os.Getenv("CHRONOGO_EXCLUDE")
	origStdlib := os.Getenv("CHRONOGO_INSTRUMENT_STDLIB")

	// Restore environment values at end of test
	defer func() {
		os.Setenv("CHRONOGO_ENABLED", origEnabled)
		os.Setenv("CHRONOGO_INSTRUMENT", origInstrument)
		os.Setenv("CHRONOGO_EXCLUDE", origExclude)
		os.Setenv("CHRONOGO_INSTRUMENT_STDLIB", origStdlib)
	}()

	// Test with explicit values
	os.Setenv("CHRONOGO_ENABLED", "1")
	os.Setenv("CHRONOGO_INSTRUMENT", "pkg1,pkg2,pkg3")
	os.Setenv("CHRONOGO_EXCLUDE", "test,benchmark")
	os.Setenv("CHRONOGO_INSTRUMENT_STDLIB", "true")

	options := loadOptionsFromEnvironment()

	// Verify options
	if !options.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if len(options.IncludePackages) != 3 {
		t.Errorf("Expected 3 include packages, got %d", len(options.IncludePackages))
	} else {
		expectedIncludes := []string{"pkg1", "pkg2", "pkg3"}
		for i, pkg := range expectedIncludes {
			if options.IncludePackages[i] != pkg {
				t.Errorf("Expected include package %d to be %q, got %q",
					i, pkg, options.IncludePackages[i])
			}
		}
	}

	if len(options.ExcludePackages) != 2 {
		t.Errorf("Expected 2 exclude packages, got %d", len(options.ExcludePackages))
	} else {
		expectedExcludes := []string{"test", "benchmark"}
		for i, pkg := range expectedExcludes {
			if options.ExcludePackages[i] != pkg {
				t.Errorf("Expected exclude package %d to be %q, got %q",
					i, pkg, options.ExcludePackages[i])
			}
		}
	}

	if !options.InstrumentStdlib {
		t.Error("Expected InstrumentStdlib to be true")
	}

	// Test with disabled values
	os.Setenv("CHRONOGO_ENABLED", "0")
	os.Setenv("CHRONOGO_INSTRUMENT", "")
	os.Setenv("CHRONOGO_EXCLUDE", "")
	os.Setenv("CHRONOGO_INSTRUMENT_STDLIB", "false")

	options = loadOptionsFromEnvironment()

	// Verify options
	if options.Enabled {
		t.Error("Expected Enabled to be false")
	}

	if options.InstrumentStdlib {
		t.Error("Expected InstrumentStdlib to be false")
	}
}
