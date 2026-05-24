// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"os/signal"
)

func defaultRunContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), shutdownSignals()...)
}
