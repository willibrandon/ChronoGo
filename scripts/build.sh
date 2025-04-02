#!/usr/bin/env bash
#
# ChronoGo Build Script for Bash
#

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION="0.1.0" # TODO: Use git tags for versioning

# Parse arguments
RELEASE=0
TEST=0
CLEAN=0
LINT=0
ALL=0
OUTPUT="chrono"

# OS detection
case "$(uname -s)" in
  CYGWIN*|MINGW*|MSYS*)
    OUTPUT="chrono.exe"
    ;;
esac

print_usage() {
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  -r, --release    Build in release mode (optimized, stripped)"
  echo "  -t, --test       Run tests after building"
  echo "  -c, --clean      Clean build artifacts"
  echo "  -l, --lint       Run linter"
  echo "  -a, --all        Clean, lint, build, and test"
  echo "  -o, --output     Specify output filename"
  echo "  -h, --help       Show this help message"
  echo ""
  echo "Example: $0 --release --test"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -r|--release)
      RELEASE=1
      shift
      ;;
    -t|--test)
      TEST=1
      shift
      ;;
    -c|--clean)
      CLEAN=1
      shift
      ;;
    -l|--lint)
      LINT=1
      shift
      ;;
    -a|--all)
      ALL=1
      shift
      ;;
    -o|--output)
      OUTPUT="$2"
      shift 2
      ;;
    -h|--help)
      print_usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      print_usage
      exit 1
      ;;
  esac
done

# Common build flags
COMMON_FLAGS="-trimpath"

# Set build flags based on release mode
if [[ $RELEASE -eq 1 ]]; then
  echo -e "\033[32mBuilding in RELEASE mode...\033[0m"
  BUILD_FLAGS="$COMMON_FLAGS -ldflags=\"-s -w -X 'github.com/willibrandon/ChronoGo/pkg/version.Version=$VERSION' -X 'github.com/willibrandon/ChronoGo/pkg/version.BuildTime=$BUILD_TIME'\""
else
  echo -e "\033[33mBuilding in DEBUG mode...\033[0m"
  BUILD_FLAGS="$COMMON_FLAGS -gcflags=\"all=-N -l\" -ldflags=\"-X 'github.com/willibrandon/ChronoGo/pkg/version.Version=dev-$VERSION' -X 'github.com/willibrandon/ChronoGo/pkg/version.BuildTime=$BUILD_TIME'\""
fi

# Function to run the linter
run_lint() {
  echo -e "\033[36mRunning linters...\033[0m"
  if ! command -v golangci-lint &> /dev/null; then
    echo -e "\033[31mgolangci-lint not found. Please install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest\033[0m"
    return 1
  fi
  
  echo -e "\033[33m============================================================\033[0m"
  echo -e "\033[33m                       LINT RESULTS                         \033[0m"
  echo -e "\033[33m============================================================\033[0m"
  
  # Run linter with direct output to preserve coloring
  golangci-lint run --color=always ./...
  local status=$?
  
  echo -e "\033[33m============================================================\033[0m"
  
  if [ $status -ne 0 ]; then
    echo -e "\033[31mLinting failed with errors\033[0m"
    return $status
  fi
  
  echo -e "\033[32mLinting passed\033[0m"
  return 0
}

# Function to build the main executable
build_main() {
  echo -e "\033[36mBuilding ChronoGo main executable...\033[0m"
  
  cd "$PROJECT_ROOT"
  eval go build $BUILD_FLAGS -o "$OUTPUT" ./cmd/chrono
  local status=$?
  if [ $status -ne 0 ]; then
    echo -e "\033[31mBuild failed\033[0m"
    return $status
  fi
  
  echo -e "\033[32mBuild successful: $OUTPUT\033[0m"
  return 0
}

# Function to run tests
run_tests() {
  echo -e "\033[36mRunning tests...\033[0m"
  
  echo -e "\033[33m============================================================\033[0m"
  echo -e "\033[33m                       TEST RESULTS                         \033[0m"
  echo -e "\033[33m============================================================\033[0m"
  
  cd "$PROJECT_ROOT"
  
  # Run tests and capture output
  go test -v ./... | while IFS= read -r line; do
    # Format based on the content
    if [[ $line == "--- PASS"* ]]; then
      echo -e "\033[32m$line\033[0m"
    elif [[ $line == "--- FAIL"* ]]; then
      echo -e "\033[31m$line\033[0m"
    elif [[ $line == "PASS"* ]]; then
      echo -e "\033[32m$line\033[0m"
    elif [[ $line == "FAIL"* ]]; then
      echo -e "\033[31m$line\033[0m"
    elif [[ $line == *"fail"* || $line == *"error"* || $line == *"Error"* ]]; then
      echo -e "\033[31m$line\033[0m"
    else
      echo "$line"
    fi
  done
  
  # Check the exit status of the go test command
  local status=${PIPESTATUS[0]}
  echo -e "\033[33m============================================================\033[0m"
  
  if [ $status -ne 0 ]; then
    echo -e "\033[31mTests failed\033[0m"
    return $status
  fi
  
  echo -e "\033[32mAll tests passed\033[0m"
  return 0
}

# Function to clean build artifacts
clean_build() {
  echo -e "\033[36mCleaning build artifacts...\033[0m"
  
  if [ -f "$PROJECT_ROOT/$OUTPUT" ]; then
    rm -f "$PROJECT_ROOT/$OUTPUT"
    echo -e "\033[90mRemoved $OUTPUT\033[0m"
  fi
  
  # Clean go test cache
  go clean -testcache
  echo -e "\033[90mCleaned Go test cache\033[0m"
  
  echo -e "\033[32mClean complete\033[0m"
  return 0
}

# Main script execution
if [[ $CLEAN -eq 1 || $ALL -eq 1 ]]; then
  clean_build
  if [[ $CLEAN -eq 1 && $ALL -eq 0 ]]; then exit 0; fi
fi

if [[ $LINT -eq 1 || $ALL -eq 1 ]]; then
  run_lint
  lint_result=$?
  if [ $lint_result -ne 0 ]; then exit $lint_result; fi
  if [[ $LINT -eq 1 && $ALL -eq 0 ]]; then exit 0; fi
fi

build_main
build_result=$?
if [ $build_result -ne 0 ]; then exit $build_result; fi

if [[ $TEST -eq 1 || $ALL -eq 1 ]]; then
  run_tests
  test_result=$?
  if [ $test_result -ne 0 ]; then exit $test_result; fi
fi

echo -e "\033[32mAll operations completed successfully\033[0m" 