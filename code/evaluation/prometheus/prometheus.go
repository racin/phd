package prometheus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
	ErrType = errors.New("wrong value type")
)

func getValue(api v1.API, query string) (value float64, err error) {
	result, warnings, err := api.Query(context.TODO(), query, time.Now())
	if err != nil {
		return 0, err
	}

	for _, warning := range warnings {
		log.Printf("prometheus warning: %s", warning)
	}

	scalar, ok := result.(*model.Scalar)
	if !ok {
		return 0, fmt.Errorf("%w: got %T, want %T", ErrType, result, scalar)
	}

	return float64(scalar.Value), nil
}

const measureInterval = 250 * time.Millisecond
const equalMaxCntr = 480

func MeasureRateFallTime(ctx context.Context, api v1.API, waitRise time.Duration, maxWait time.Duration, query string) (time.Duration, error) {
	now := time.Now()
	durationSinceGotValue := time.Duration(0)
	didRise := false
	var oldVal model.SampleValue
	var equalCntr int
	for {
		result, _, err := api.Query(context.TODO(), query, time.Now())
		if err != nil {
			return 0, err
		}

		scalar := result.(model.Vector)
		if len(scalar) != 1 {
			return 0, fmt.Errorf("expected 1 result, got %d", len(scalar))
		}

		if scalar[0].Value.Equal(oldVal) {
			if didRise && equalCntr == equalMaxCntr {
				return durationSinceGotValue, nil
				// return time.Since(now) - time.Duration(measureInterval*equalMaxCntr), nil
			} else if !didRise && time.Since(now) > waitRise {
				return 0, fmt.Errorf("did not rise after %s", waitRise)
			} else if durationSinceGotValue > maxWait {
				return 0, fmt.Errorf("did fall down after %s", maxWait)
			}
			equalCntr++
		} else {
			if !didRise {
				didRise = true
			}
			oldVal = scalar[0].Value
			equalCntr = 0
			durationSinceGotValue = time.Since(now)
		}

		<-time.After(measureInterval)
	}
}

// func MeasureRateFallTime(ctx context.Context, api v1.API, waitRise time.Duration, query string) (time.Duration, error) {
// 	now := time.Now()
// 	didRise := false
// 	for {
// 		result, _, err := api.Query(context.TODO(), query, time.Now())
// 		if err != nil {
// 			return 0, err
// 		}

// 		scalar := result.(model.Vector)
// 		if len(scalar) != 1 {
// 			return 0, fmt.Errorf("expected 1 result, got %d", len(scalar))
// 		}

// 		if scalar[0].Value == 0 {
// 			if didRise {
// 				return time.Since(now), nil
// 			} else if time.Since(now) > waitRise {
// 				return 0, fmt.Errorf("did not rise after %s", waitRise)
// 			}
// 		} else if !didRise {
// 			didRise = true
// 		}
// 		<-time.After(100 * time.Millisecond)
// 	}
// }

func RatePoller(ctx context.Context, api v1.API, query string) (time.Duration, error) {
	now := time.Now()

	result, warnings, err := api.Query(context.TODO(), query, time.Now())
	if err != nil {
		return 0, err
	}

	for _, warning := range warnings {
		log.Printf("prometheus warning: %s", warning)
	}

	scalar := result.(model.Vector)
	if len(scalar) == 1 {
		fmt.Printf("Got scalar: %v, result: %+#v\n", scalar[0].Value, result)
	}

	return time.Since(now), nil
}

func StartMeasurement(api v1.API, query string) (finish func() (delta float64, err error), err error) {
	startValue, err := getValue(api, query)
	if err != nil {
		return nil, err
	}

	return func() (delta float64, err error) {
		endValue, err := getValue(api, query)
		if err != nil {
			return 0, err
		}
		return endValue - startValue, nil
	}, nil
}
