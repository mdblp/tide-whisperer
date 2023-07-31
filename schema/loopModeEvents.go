package schema

import (
	"time"
)

type (
	LoopModeEvent struct {
		TimeRange    `bson:"inline"`
		DeliveryType string
	}
)

func NewLoopModeEvent(start time.Time, end *time.Time, deliveryType string) LoopModeEvent {
	return LoopModeEvent{
		TimeRange: TimeRange{
			Start: start,
			End:   end,
		},
		DeliveryType: deliveryType,
	}
}

func filterLoopModes(loopModes []LoopModeEvent) []LoopModeEvent {
	// The terminal sends not updated loopModes without duration
	// We may have loopMode objects without duration but only one
	// i.e. the last one meaning that the terminal is currently in loopMode ON
	var filteredLoopModes []LoopModeEvent
	lastIndex := len(loopModes) - 1
	for i, loopMode := range loopModes {
		if loopMode.End == nil && i != lastIndex {
			continue
		}
		filteredLoopModes = append(filteredLoopModes, loopMode)
	}
	return filteredLoopModes
}

func FillLoopModeEvents(loopModes []LoopModeEvent) []LoopModeEvent {
	if len(loopModes) == 0 {
		return []LoopModeEvent{}
	}
	var filledLoopModes []LoopModeEvent
	filteredLoopModes := filterLoopModes(loopModes)
	lastIndex := len(filteredLoopModes) - 1
	currentLoopMode := filteredLoopModes[0]

	for i := range filteredLoopModes {
		if i != lastIndex {
			next := filteredLoopModes[i+1]
			unionRanges := currentLoopMode.TimeRange.Union(next.TimeRange, true)
			lastUnionIndex := len(unionRanges) - 1
			for j, tRange := range unionRanges {
				deliveryType := "automated"
				if tRange.Fill {
					deliveryType = "scheduled"
				}
				loopModeEvent := NewLoopModeEvent(tRange.Start, tRange.End, deliveryType)
				if j == lastUnionIndex {
					currentLoopMode = loopModeEvent
				} else {
					filledLoopModes = append(filledLoopModes, loopModeEvent)
				}
			}
		} else {
			filledLoopModes = append(filledLoopModes, currentLoopMode)
		}
	}
	return filledLoopModes
}

func GetLoopModeEventsBetween(start time.Time, end *time.Time, loopModes []LoopModeEvent) []LoopModeEvent {
	var matchedLoopModes []LoopModeEvent
	myLoopMode := NewLoopModeEvent(start, end, "")
	for _, loopMode := range loopModes {
		if loopMode.TimeRange.OverLaps(myLoopMode.TimeRange) {
			matchingLoopMode := LoopModeEvent{
				TimeRange{
					Start: loopMode.Start,
					End:   loopMode.End,
				},
				loopMode.DeliveryType,
			}
			if matchingLoopMode.Start.Before(myLoopMode.Start) {
				matchingLoopMode.Start = myLoopMode.Start
			}
			if TimeAfterOrEqual(matchingLoopMode.End, myLoopMode.End) {
				matchingLoopMode.End = myLoopMode.End
			}
			matchedLoopModes = append(matchedLoopModes, matchingLoopMode)
		}
	}
	if len(matchedLoopModes) == 0 {
		myLoopMode.DeliveryType = "scheduled"
		return []LoopModeEvent{myLoopMode}
	}
	return matchedLoopModes
}
