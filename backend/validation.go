package main

import "fmt"

var artifactStatuses = map[string]bool{"stable": true, "watch": true, "treatment": true}

func ValidateArtifactStatus(status string) error {
	if !artifactStatuses[status] {
		return fmt.Errorf("status must be stable, watch, or treatment")
	}
	return nil
}
