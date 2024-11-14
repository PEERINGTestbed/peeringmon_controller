package main

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var lastSite *ConfigSite

var (
	routesGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "current_announcements",
		Help: "current announcements",
	}, []string{"prefix", "site", "id"})
)

func (p *Prefix) update(site *ConfigSite) {
	if p.announcing {
		routesGauge.DeleteLabelValues(
			p.prefix,
			p.lastAdvSite.Name,
			strconv.Itoa(p.lastAdvSite.Id+2000),
		)
		p.bgpWithdraw()
		return
	}

	p.bgpAnnounce(site)
	routesGauge.WithLabelValues(
		p.prefix,
		p.lastAdvSite.Name,
		strconv.Itoa(p.lastAdvSite.Id+2000),
	).Set(1)

	return
}

func cycle() {
	for i := range prefixes {
		prefix := prefixes[i]
		site := nextSite()
		prefix.update(site)
		lastSite = site
	}
}

func nextSite() *ConfigSite {
	if lastSite == nil {
		return Config.Sites[0]
	}
	for i, site := range Config.Sites {
		if site == lastSite {
			if i == len(Config.Sites)-1 {
				return Config.Sites[0]
			}
			return Config.Sites[i+1]
		}
	}
	panic("unreachable, maybe Config.Sites empty")
}
