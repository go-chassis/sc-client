package client

import (
	"github.com/cenkalti/backoff"
	"log"
	"net/url"
	"time"
)

func getBackOff(backoffType string) backoff.BackOff {
	switch backoffType {
	case "Exponential":
		return &backoff.ExponentialBackOff{
			InitialInterval:     1000 * time.Millisecond,
			RandomizationFactor: backoff.DefaultRandomizationFactor,
			Multiplier:          backoff.DefaultMultiplier,
			MaxInterval:         30000 * time.Millisecond,
			MaxElapsedTime:      10000 * time.Millisecond,
			Clock:               backoff.SystemClock,
		}
	case "Constant":
		return backoff.NewConstantBackOff(DefaultRetryTimeout * time.Millisecond)
	case "Zero":
		return &backoff.ZeroBackOff{}
	default:
		return backoff.NewConstantBackOff(DefaultRetryTimeout * time.Millisecond)
	}
}

func getProtocolMap(eps []string) map[string]string {
	m := make(map[string]string)
	for _, ep := range eps {
		u, err := url.Parse(ep)
		if err != nil {
			log.Println("URL error" + err.Error())
			continue
		}
		m[u.Scheme] = u.Host
	}
	return m
}
