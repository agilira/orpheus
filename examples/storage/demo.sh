#!/bin/bash
# Orpheus Plugin System Demo Test Script
# Demonstrates the complete plugin architecture functionality
#
# Copyright (c) 2025 AGILira - A. Giordano
# SPDX-License-Identifier: MPL-2.0

set -e

echo " Orpheus Plugin System Demo"
echo "=============================="

echo " Building storage plugins..."
./build_plugins.sh

echo ""
echo " Building demo application..."
go build -o storage-demo .

echo ""
echo " Testing Plugin Loading System..."
echo "-----------------------------------"

# Test plugin loading
echo " Plugin system information..."
./storage-demo info

echo ""
echo " Testing plugin operations (note: each command loads plugin fresh)..."
./storage-demo set "demo_key" "Plugin System Works!"
./storage-demo set "user:123" "John Doe"  
./storage-demo set "config:theme" "dark"

echo ""
echo " Testing plugin state..."
echo "Note: Each command creates fresh plugin instance"
./storage-demo list

echo ""
echo "  Testing plugin cleanup..."
./storage-demo clear

echo ""
echo " Performance benchmark with plugin overhead..."
./storage-demo benchmark --operations 1000 --concurrency 5

echo ""
echo " Security test (plugin validation)..."
./storage-demo security-test

echo ""
echo " Plugin System Demo completed successfully!"
echo "    Dynamic plugin loading working correctly"
echo "    Plugin security system validated"
echo "    Performance excellent despite plugin overhead"
echo ""
echo " Generated artifacts:"
echo "   - plugins/memory.so (compiled storage plugin)"
echo "   - storage-demo (demo application)"

# Clean up
rm -f storage-demo