package basal

import (
	"fmt"
	"sort"
	"time"

	"github.com/mdblp/tide-whisperer-v2/v2/schema"
	schemaV1 "github.com/tidepool-org/tide-whisperer/schema"
)

func fillMissingBasal(currentBasal schema.BasalSample, previousEndBasalTime time.Time, countBasal int, loopModes []schemaV1.LoopModeEvent) (time.Time, int, []schema.BasalSample) {
	sampleTime := previousEndBasalTime
	previousEndBasalTime = currentBasal.Timestamp
	fillSamples := []schema.BasalSample{}
	for _, loopMode := range schemaV1.GetLoopModeEventsBetween(sampleTime, &previousEndBasalTime, loopModes) {
		fmt.Printf("%v %v %v\n", sampleTime, previousEndBasalTime, loopMode)
		currentSample := getSample(currentBasal.Guid, countBasal, 0, currentBasal.Timezone, previousEndBasalTime, loopMode)
		fillSamples = append(fillSamples, currentSample)
		countBasal++
	}
	return previousEndBasalTime, countBasal, fillSamples
}

func fixOverlappingBasal(previousSamples []schema.BasalSample, currentBasal schema.BasalSample) []schema.BasalSample {
	previousBasalSample := previousSamples[len(previousSamples)-1]
	previousSamples = previousSamples[:len(previousSamples)-1]
	previousDuration := int(currentBasal.Timestamp.UnixMilli() - previousBasalSample.Timestamp.UnixMilli())
	for previousDuration < 0 && len(previousSamples) > 0 {
		previousBasalSample = previousSamples[len(previousSamples)-1]
		previousSamples = previousSamples[:len(previousSamples)-1]
		previousDuration = int(currentBasal.Timestamp.UnixMilli() - previousBasalSample.Timestamp.UnixMilli())
	}
	if previousDuration > 0 {
		previousBasalSample.Duration = previousDuration
		previousSamples = append(previousSamples, previousBasalSample)
	}
	return previousSamples
}


func cutBasalWithLoopModes(currentBasal schema.BasalSample, countBasal int, loopModes []schemaV1.LoopModeEvent) (time.Time, int, []schema.BasalSample) {
	duration := time.Duration(currentBasal.Duration) * time.Millisecond
	endBasalTime := currentBasal.Timestamp.Add(duration)
	cutSamples := []schema.BasalSample{}
	for _, loopMode := range schemaV1.GetLoopModeEventsBetween(currentBasal.Timestamp, &endBasalTime, loopModes) {
		currentSample := getSample(currentBasal.Guid, countBasal, currentBasal.Rate, currentBasal.Timezone, endBasalTime, loopMode)
		cutSamples = append(cutSamples, currentSample)
		countBasal++
	}
	return endBasalTime, countBasal, cutSamples
}


func CleanUpBasals(basals []schema.BasalBucket, loopModes []schemaV1.LoopModeEvent) []schema.BasalBucket {
	var endBasalTime time.Time
	sort.SliceStable(basals, func(i, j int) bool {
		return basals[i].Day.Before(basals[j].Day)
	})
	cleanBuckets := []schema.BasalBucket{}
	countBasal := 0
	for _, basalDay := range basals {
		sort.SliceStable(basalDay.Samples, func(i, j int) bool {
			return basalDay.Samples[i].Timestamp.Before(basalDay.Samples[j].Timestamp)
		})
		cleanSamples := []schema.BasalSample{}
		for _, basal := range basalDay.Samples {
			// Filling time holes with 0 rate basals...
			if !endBasalTime.IsZero() && basal.Timestamp.Sub(endBasalTime) >= time.Second {
				var fillSamples []schema.BasalSample
				endBasalTime, countBasal, fillSamples = fillMissingBasal(basal, endBasalTime, countBasal, loopModes)
				cleanSamples = append(cleanSamples, fillSamples...)
			}
			// Checking basal overlap
			// Reprocessing previous basal duration
			if endBasalTime.After(basal.Timestamp) {
				cleanSamples = fixOverlappingBasal(cleanSamples, basal)
			}
			var cutSamples []schema.BasalSample
			endBasalTime, countBasal, cutSamples = cutBasalWithLoopModes(basal, countBasal, loopModes)
			cleanSamples = append(cleanSamples, cutSamples...)
		}
		basalDay.Samples = cleanSamples
		cleanBuckets = append(cleanBuckets, basalDay)
	}
	return cleanBuckets
}
