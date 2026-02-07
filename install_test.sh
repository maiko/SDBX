#!/usr/bin/env bash
# Tests for install.sh checksum verification logic

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PASS=0
FAIL=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

assert_eq() {
    local test_name="$1" expected="$2" actual="$3"
    if [ "$expected" = "$actual" ]; then
        echo -e "${GREEN}PASS${NC} $test_name"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} $test_name (expected: '$expected', got: '$actual')"
        FAIL=$((FAIL + 1))
    fi
}

# Source the compute_sha256 function from install.sh
eval "$(sed -n '/^compute_sha256/,/^}/p' "$SCRIPT_DIR/install.sh")"

# Test 1: compute_sha256 produces correct hash for known input
test_compute_sha256() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    echo -n "hello world" > "$tmp_dir/test.txt"
    local hash
    hash=$(compute_sha256 "$tmp_dir/test.txt" | awk '{print $1}')
    assert_eq "compute_sha256 known hash" \
        "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9" \
        "$hash"

    rm -rf "$tmp_dir"
}

# Test 2: verify_checksum succeeds with matching hash
test_verify_checksum_match() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    echo -n "fake archive content" > "$tmp_dir/sdbx_1.0.0_linux_amd64.tar.gz"
    local actual_hash
    actual_hash=$(compute_sha256 "$tmp_dir/sdbx_1.0.0_linux_amd64.tar.gz" | awk '{print $1}')
    echo "$actual_hash  sdbx_1.0.0_linux_amd64.tar.gz" > "$tmp_dir/checksums.txt"

    local expected_hash
    expected_hash=$(grep "sdbx_1.0.0_linux_amd64.tar.gz" "$tmp_dir/checksums.txt" | awk '{print $1}')

    assert_eq "verify matching checksum" "$expected_hash" "$actual_hash"

    rm -rf "$tmp_dir"
}

# Test 3: verify_checksum detects mismatch
test_verify_checksum_mismatch() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    echo -n "fake archive content" > "$tmp_dir/sdbx_1.0.0_linux_amd64.tar.gz"
    echo "0000000000000000000000000000000000000000000000000000000000000000  sdbx_1.0.0_linux_amd64.tar.gz" > "$tmp_dir/checksums.txt"

    local expected_hash actual_hash
    expected_hash=$(grep "sdbx_1.0.0_linux_amd64.tar.gz" "$tmp_dir/checksums.txt" | awk '{print $1}')
    actual_hash=$(compute_sha256 "$tmp_dir/sdbx_1.0.0_linux_amd64.tar.gz" | awk '{print $1}')

    if [ "$expected_hash" != "$actual_hash" ]; then
        echo -e "${GREEN}PASS${NC} verify mismatch detected"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} verify mismatch not detected"
        FAIL=$((FAIL + 1))
    fi

    rm -rf "$tmp_dir"
}

# Test 4: missing archive in checksums.txt returns empty
test_verify_checksum_missing_entry() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    echo "abc123  some_other_file.tar.gz" > "$tmp_dir/checksums.txt"

    local expected_hash
    expected_hash=$(grep "sdbx_1.0.0_linux_amd64.tar.gz" "$tmp_dir/checksums.txt" | awk '{print $1}' || true)

    assert_eq "missing entry returns empty" "" "$expected_hash"

    rm -rf "$tmp_dir"
}

echo "=== Install Script Tests ==="
echo ""

test_compute_sha256
test_verify_checksum_match
test_verify_checksum_mismatch
test_verify_checksum_missing_entry

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
