package common

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TimeItKey int

type TimeItType struct {
	timers  map[string]time.Time
	results string
}

// IsValidUUID check if the uuid is valid
func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// contains search an element in an array
//
// go seems to not have this helper in the base API
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
func ContainsInt(a []int, x int) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func TimeItContext(ctx context.Context) context.Context {
	value := &TimeItType{
		timers: make(map[string]time.Time),
	}
	return context.WithValue(ctx, TimeItKey(0), value)
}

func TimeIt(ctx context.Context, name string) {
	ctxValue := ctx.Value(TimeItKey(0)).(*TimeItType)
	if ctxValue == nil {
		fmt.Printf("timeIt: Invalid context")
		return
	}
	timerValues := ctxValue.timers
	if _, present := timerValues[name]; present {
		fmt.Printf("timeIt: Timer %s already started\n", name)
		return
	}
	timerValues[name] = time.Now()
}

func TimeEnd(ctx context.Context, name string) int64 {
	ctxValue := ctx.Value(TimeItKey(0)).(*TimeItType)
	if ctxValue == nil {
		return 0
	}
	timerValues := ctxValue.timers
	start, present := timerValues[name]
	if !present {
		fmt.Printf("timeEnd: Timer %s has not started\n", name)
		return 0
	}
	end := time.Now()
	delete(timerValues, name)
	dur := end.Sub(start).Milliseconds()
	if len(ctxValue.results) == 0 {
		ctxValue.results = fmt.Sprintf("%s:%dms", name, dur)
	} else {
		ctxValue.results = fmt.Sprintf("%s %s:%dms", ctxValue.results, name, dur)
	}
	return dur
}

func TimeResults(ctx context.Context) string {
	ctxValue := ctx.Value(TimeItKey(0)).(*TimeItType)
	if ctxValue == nil {
		return ""
	}
	return ctxValue.results
}
