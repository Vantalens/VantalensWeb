//go:build !desktop && !legacy

package main

import "fmt"

func main() {
	fmt.Println("TalentWriter legacy root entry is disabled. Run web.exe or build ./cmd/server instead.")
}

// launchDesktopShell is a no-op in non-desktop builds.
func launchDesktopShell(url, title string, width, height int) (bool, error) {
	return false, fmt.Errorf("desktop shell is only available with -tags desktop")
}
