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
func BuildAll() {
	b := build.Build{}
	mg.Deps(b.Linux, b.Windows, b.Darwin, b.DarwinARM64, b.LinuxARM64)
}

// Default configures the default target.
var Default = BuildAll
