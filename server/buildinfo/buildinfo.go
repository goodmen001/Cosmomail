// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 NovaMail

package buildinfo

// Values are replaced by release/build scripts through -ldflags -X.
var (
	Version   = "2.0.8"
	BuildTime = "unknown"
	GitCommit = "dev"
)
