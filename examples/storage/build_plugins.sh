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

# A Go plugin is only loadable by a host built the same way, and the race
# detector changes that. Pass RACE=1 to build a plugin that `go test -race`
# can open; without it the race-enabled tests cannot load this plugin at all.
RACE_FLAG=""
if [ "${RACE:-0}" = "1" ]; then
    RACE_FLAG="-race"
    echo " (race detector enabled)"
fi

echo " Building memory storage plugin..."
cd providers
GOWORK=off go build $RACE_FLAG -buildmode=plugin -o ../plugins/memory.so memory.go
echo " memory.so built successfully"

echo ""
echo " All plugins built successfully!"
echo " Plugin files:"
ls -la ../plugins/

echo ""
echo " Plugin verification:"
file ../plugins/*.so