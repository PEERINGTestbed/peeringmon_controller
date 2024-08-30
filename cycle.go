package main

var lastSite *ConfigSite

func (p *Prefix) update(site *ConfigSite) {
	if p.lastAdvSite != nil {
		p.bgpWithdraw()
	}
	p.bgpAnnounce(site)
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
