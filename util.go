package sc

import (
	"net/url"

	"github.com/go-chassis/openlog"
)

func getProtocolMap(eps []string) map[string]string {
	m := make(map[string]string)
	for _, ep := range eps {
		u, err := url.Parse(ep)
		if err != nil {
			openlog.Error("url err: " + err.Error())
			continue
		}
		m[u.Scheme] = u.Host
	}
	return m
}
