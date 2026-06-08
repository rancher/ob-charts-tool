#!/bin/bash
# Check that helmtools/ maintains proper package boundaries
# helmtools should not import CLI frameworks or internal/cmd packages

set -e

echo "Checking helmtools package boundaries..."

FORBIDDEN_PATTERNS=(
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "github.com/urfave/cli"
    "github.com/rancher/ob-charts-tool/internal"
    "github.com/rancher/ob-charts-tool/cmd"
)

VIOLATIONS=0

for pattern in "${FORBIDDEN_PATTERNS[@]}"; do
    if grep -r "$pattern" helmtools/ --include="*.go" > /dev/null 2>&1; then
        echo "❌ VIOLATION: helmtools/ imports forbidden package: $pattern"
        grep -rn "$pattern" helmtools/ --include="*.go"
        VIOLATIONS=$((VIOLATIONS + 1))
    fi
done

if [ $VIOLATIONS -eq 0 ]; then
    echo "✅ All checks passed - helmtools/ maintains proper boundaries"
    exit 0
else
    echo ""
    echo "❌ Found $VIOLATIONS violation(s)"
    echo "helmtools/ must not import CLI frameworks or internal/cmd packages"
    exit 1
fi
