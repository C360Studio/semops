package mavlink

import (
	"fmt"
	"strings"
)

func MAVResultString(result uint8) string {
	switch result {
	case MAVResultAccepted:
		return "accepted"
	case MAVResultTemporarilyRejected:
		return "temporarily_rejected"
	case MAVResultDenied:
		return "denied"
	case MAVResultUnsupported:
		return "unsupported"
	case MAVResultFailed:
		return "failed"
	case MAVResultInProgress:
		return "in_progress"
	case MAVResultCancelled:
		return "cancelled"
	default:
		return fmt.Sprintf("unknown(%d)", result)
	}
}

func ArduCopterCustomMode(mode string) (uint32, error) {
	normalized := strings.ToUpper(strings.TrimSpace(mode))
	if customMode, ok := arducopterModeByName[normalized]; ok {
		return customMode, nil
	}
	return 0, fmt.Errorf("unknown ArduCopter mode %q", mode)
}

func ArduCopterModeName(customMode uint32) string {
	if mode, ok := arducopterModeByValue[customMode]; ok {
		return mode
	}
	return fmt.Sprintf("UNKNOWN(%d)", customMode)
}

var arducopterModeByName = map[string]uint32{
	"STABILIZE":    0,
	"ACRO":         1,
	"ALT_HOLD":     2,
	"AUTO":         3,
	"GUIDED":       4,
	"LOITER":       5,
	"RTL":          6,
	"CIRCLE":       7,
	"LAND":         9,
	"DRIFT":        11,
	"SPORT":        13,
	"FLIP":         14,
	"AUTOTUNE":     15,
	"POSHOLD":      16,
	"BRAKE":        17,
	"THROW":        18,
	"AVOID_ADSB":   19,
	"GUIDED_NOGPS": 20,
	"SMART_RTL":    21,
	"FLOWHOLD":     22,
	"FOLLOW":       23,
	"ZIGZAG":       24,
	"SYSTEMID":     25,
	"AUTOROTATE":   26,
}

var arducopterModeByValue = func() map[uint32]string {
	out := make(map[uint32]string, len(arducopterModeByName))
	for mode, customMode := range arducopterModeByName {
		out[customMode] = mode
	}
	return out
}()
