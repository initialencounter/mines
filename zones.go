package main

import (
	"math/rand/v2"
)

const (
	doubleScoreZoneSize = 17
	noFlagZoneSize      = 11
)

func generateZones(width, height int) []Zone {
	zones := make([]Zone, 0)

	// Always generate double-score zone (fixed 17x17)
	if z := generateOneZone(width, height, doubleScoreZoneSize, doubleScoreZoneSize, ZoneDoubleScore, zones); z != nil {
		zones = append(zones, *z)
	}

	// Always generate no-flag zone (fixed 11x11)
	if z := generateOneZone(width, height, noFlagZoneSize, noFlagZoneSize, ZoneNoFlag, zones); z != nil {
		zones = append(zones, *z)
	}

	return zones
}

func generateOneZone(width, height, zoneW, zoneH int, zoneType ZoneType, existing []Zone) *Zone {
	if zoneW > width {
		zoneW = width
	}
	if zoneH > height {
		zoneH = height
	}

	// Try up to 20 times to find non-overlapping position
	for attempt := 0; attempt < 20; attempt++ {
		startRow := rand.IntN(height - zoneH + 1)
		startCol := rand.IntN(width - zoneW + 1)
		endRow := startRow + zoneH - 1
		endCol := startCol + zoneW - 1

		candidate := Zone{
			StartRow: startRow,
			StartCol: startCol,
			EndRow:   endRow,
			EndCol:   endCol,
			Type:     zoneType,
		}

		if !zoneOverlaps(candidate, existing) {
			return &candidate
		}
	}
	return nil
}

func zoneOverlaps(z Zone, existing []Zone) bool {
	for _, e := range existing {
		if !(z.EndRow < e.StartRow || z.StartRow > e.EndRow ||
			z.EndCol < e.StartCol || z.StartCol > e.EndCol) {
			return true
		}
	}
	return false
}
