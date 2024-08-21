package main

import (
	"github.com/rs/zerolog/log"
)

type Prefix struct {
	prefix string
	site   string
}

var prefixes []*Prefix

var lastSite string

func initPrefixes() {
	for _, prefix := range Config.Prefixes {
		prefixes = append(prefixes, &Prefix{prefix, ""})
	}
}

func (p *Prefix) update(site string) {
	// set prom entry
	log.Debug().
		Str("site", site).
		Str("prefix", p.prefix).
		Msg("updating prom entry")
	// call bgp update
	if p.site != "" {
		log.Info().
			Str("site", p.site).
			Str("prefix", p.prefix).
			Msg("withdrawing")
	}
	log.Info().
		Str("site", site).
		Str("prefix", p.prefix).
		Msg("announcing")
	p.site = site
	return
}

func cycle() {
	for _, prefix := range prefixes {
		site := nextSite()
		prefix.update(site)
		lastSite = site
	}
}

func nextSite() string {
	if lastSite == "" {
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
