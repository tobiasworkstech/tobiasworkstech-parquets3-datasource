//go:build mage
// +build mage

package main

import (
	"github.com/grafana/grafana-plugin-sdk-go/build"
	"github.com/magefile/mage/mg"
)

// BuildAll builds production executables for every platform we ship except
// 32-bit linux/arm. The 32-bit target is dropped because Apache Thrift v0.23.0
// (pulled in transitively by apache/arrow-go and required to clear
// CVE-2026-41602) uses `math.MaxUint32` as an untyped int constant in
// framed_transport.go, which overflows on platforms where `int` is 32 bits.
// 32-bit ARM is rarely used for Grafana servers in practice (RPi 4+ runs
// arm64), so dropping the artifact is preferable to staying on the vulnerable
// thrift version.
//
// We do not use `// mage:import` for the SDK build package because mage
// disallows duplicate target names across imports and local definitions —
// importing the SDK as a namespace would shadow this override of BuildAll.
// The Coverage wrapper below delegates to the SDK's implementation so the
// `grafana/plugin-actions/package-plugin` step that runs `mage coverage`
// keeps working.
func BuildAll() {
	b := build.Build{}
	mg.Deps(b.Linux, b.Windows, b.Darwin, b.DarwinARM64, b.LinuxARM64)
}

// Coverage runs backend tests and writes coverage/backend.out, matching the
// target that `grafana/plugin-actions/package-plugin` invokes before the
// build step.
func Coverage() error {
	return build.Coverage()
}

// Default configures the default target.
var Default = BuildAll
