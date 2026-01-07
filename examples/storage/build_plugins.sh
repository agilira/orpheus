#!/bin/bash
# Build Storage Plugins Script
# Compiles storage providers as shared library plugins
#
# Copyright (c) 2025 AGILira - A. Giordano
# SPDX-License-Identifier: MPL-2.0

set -e

echo " Building Orpheus Storage Plugins"
echo "==================================="

# Create plugins directory
mkdir -p plugins

echo " Building memory storage plugin..."
cd providers
GOWORK=off go build -buildmode=plugin -o ../plugins/memory.so memory.go
echo " memory.so built successfully"

echo ""
echo " All plugins built successfully!"
echo " Plugin files:"
ls -la ../plugins/

echo ""
echo " Plugin verification:"
file ../plugins/*.so