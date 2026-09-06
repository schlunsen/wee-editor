//go:build !race
// +build !race

package cmd

// raceDetectorEnabled is false when NOT compiled with -race flag
const raceDetectorEnabled = false
