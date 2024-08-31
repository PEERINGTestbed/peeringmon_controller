package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	api "github.com/osrg/gobgp/v3/api"
	bgpLog "github.com/osrg/gobgp/v3/pkg/log"
	"github.com/osrg/gobgp/v3/pkg/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/anypb"
	apb "google.golang.org/protobuf/types/known/anypb"
)

type Prefix struct {
	prefix      string
	pathObj     *api.Path
	lastAdvUuid []byte
	lastAdvSite *ConfigSite
}

var ctx = context.Background()
var s *server.BgpServer

func bgpInit() {
	s = server.NewBgpServer(server.LoggerOption(&myLogger{logger: &log.Logger}))
	go s.Serve()

	if err := s.StartBgp(ctx, &api.StartBgpRequest{
		Global: &api.Global{
			Asn:             Config.ASN,
			RouterId:        Config.RouterID,
			ListenPort:      Config.ListenPort,
			ListenAddresses: Config.ListenAddr,
		},
	}); err != nil {
		log.Fatal().Err(err).Msg("Failed to start BGP server")
	}

	if err := s.WatchEvent(ctx, &api.WatchEventRequest{Peer: &api.WatchEventRequest_Peer{}}, func(r *api.WatchEventResponse) {
		if p := r.GetPeer(); p != nil && p.Type == api.WatchEventResponse_PeerEvent_STATE {
			log.Debug().
				Str("src", "gobgp.peer").
				Msg(p.String())
		}
	}); err != nil {
		log.Fatal().Err(err).Msg("Failed to install watchEvent hook")
	}
}

func bgpStop(ctx context.Context) error {
	return s.StopBgp(ctx, &api.StopBgpRequest{})
}

func prefixesInit() (prefixes []*Prefix) {
	for i := range Config.Prefixes {
		configPrefix := Config.Prefixes[i]
		cidr := configPrefix.Prefix

		cidrSplit := strings.Split(cidr, "/")
		prefix := cidrSplit[0]
		prefixLenStr := cidrSplit[1]
		// api.IPAddressPrefix.PrefixLen takes a uint32
		prefixLen, err := strconv.ParseUint(prefixLenStr, 10, 32)
		if err != nil {
			log.Fatal().Err(err).
				Str("prefixLen", prefixLenStr).
				Msg("cant convert prefixLen to int")
		}

		nlri, _ := apb.New(&api.IPAddressPrefix{
			Prefix:    prefix,
			PrefixLen: uint32(prefixLen),
		})

		a1, _ := apb.New(&api.OriginAttribute{
			Origin: 0,
		})
		a2, _ := apb.New(&api.NextHopAttribute{
			NextHop: "0.0.0.0",
		})
		attrs := []*apb.Any{a1, a2}

		newPrefix := Prefix{
			prefix: prefix,
			pathObj: &api.Path{
				Family: &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
				Nlri:   nlri,
				Pattrs: attrs,
			},
			lastAdvUuid: nil,
			lastAdvSite: nil,
		}
		prefixes = append(prefixes, &newPrefix)
	}
	return
}

func (p *Prefix) bgpAnnounce(site *ConfigSite) {
	log.Info().
		Str("site", site.Name).
		Str("prefix", p.prefix).
		Msg("Announcing")

	rd, _ := apb.New(&api.RouteDistinguisherFourOctetASN{
		Admin:    65000,
		Assigned: 100,
	})

	a, err := apb.New(&api.IPv4AddressSpecificExtended{
		IsTransitive: true,
		SubType:      0x02,
		Address:      p.prefix,
		LocalAdmin:   100,
	})

	if err := s.AddVrf(ctx, &api.AddVrfRequest{
		Vrf: &api.Vrf{
			Name:     site.Name,
			Rd:       rd,
			ExportRt: []*anypb.Any{a},
			ImportRt: []*anypb.Any{a},
		}}); err != nil {
		log.Error().Err(err).
			Str("site", site.Name).
			Str("prefix", p.prefix).
			Msg("AddVrf")
	}

	n := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: site.Neighbor,
			PeerAsn:         uint32(site.ASN),
			Vrf:             site.Name,
		},
	}

	if err := s.AddPeer(ctx, &api.AddPeerRequest{
		Peer: n,
	}); err != nil {
		log.Error().Err(err).
			Str("site", site.Name).
			Str("prefix", p.prefix).
			Msg("AddPeer")
		return
	}

	resp, err := s.AddPath(ctx, &api.AddPathRequest{
		Path:      p.pathObj,
		VrfId:     site.Name,
		TableType: api.TableType_VRF,
	})
	if err != nil {
		log.Error().Err(err).
			Str("site", site.Name).
			Str("prefix", p.prefix).
			Msg("AddPath")
		return
	}

	p.lastAdvUuid = resp.Uuid
	p.lastAdvSite = site

	return
}

func (p *Prefix) bgpWithdraw() {
	log.Info().
		Str("neighbor", p.lastAdvSite.Name).
		Str("prefix", p.prefix).
		Msg("withdrawing")

	// make withdraw
	if err := s.DeletePath(ctx, &api.DeletePathRequest{
		Path:  p.pathObj,
		VrfId: p.lastAdvSite.Name,
	}); err != nil {
		log.Error().Err(err).
			Str("neighbor", p.lastAdvSite.Name).
			Str("prefix", p.prefix).
			Msg("DeletePath")
	}
	if err := s.ShutdownPeer(ctx, &api.ShutdownPeerRequest{
		Address: p.lastAdvSite.Neighbor,
	}); err != nil {
		log.Error().Err(err).
			Str("neighbor", p.lastAdvSite.Name).
			Str("prefix", p.prefix).
			Msg("ShutdownPeer")
	}
	if err := s.DeletePeer(ctx, &api.DeletePeerRequest{
		Address: p.lastAdvSite.Neighbor,
	}); err != nil {
		log.Error().Err(err).
			Str("neighbor", p.lastAdvSite.Name).
			Str("prefix", p.prefix).
			Msg("DeletePeer")
	}
	if err := s.DeleteVrf(ctx, &api.DeleteVrfRequest{
		Name: p.lastAdvSite.Name,
	}); err != nil {
		log.Error().Err(err).
			Str("neighbor", p.lastAdvSite.Name).
			Str("prefix", p.prefix).
			Msg("DeleteVrf")
	}
	return
}

type myLogger struct {
	logger *zerolog.Logger
}

func (l *myLogger) log(level zerolog.Level, msg string, fields bgpLog.Fields) {
	event := l.logger.WithLevel(level).Str("src", "gobgp.server")
	for key, value := range fields {
		event = event.Str(key, fmt.Sprint(value))
	}
	event.Msg(msg)
}

func (l *myLogger) Panic(msg string, fields bgpLog.Fields) {
	l.log(zerolog.PanicLevel, msg, fields)
}

func (l *myLogger) Fatal(msg string, fields bgpLog.Fields) {
	l.log(zerolog.FatalLevel, msg, fields)
}

func (l *myLogger) Error(msg string, fields bgpLog.Fields) {
	l.log(zerolog.ErrorLevel, msg, fields)
}

func (l *myLogger) Warn(msg string, fields bgpLog.Fields) {
	l.log(zerolog.WarnLevel, msg, fields)
}

func (l *myLogger) Info(msg string, fields bgpLog.Fields) {
	l.log(zerolog.InfoLevel, msg, fields)
}

func (l *myLogger) Debug(msg string, fields bgpLog.Fields) {
	l.log(zerolog.DebugLevel, msg, fields)
}

func (l *myLogger) SetLevel(level bgpLog.LogLevel) {
	l.logger.Level(zerolog.Level(level))
}

func (l *myLogger) GetLevel() bgpLog.LogLevel {
	return bgpLog.LogLevel(l.logger.GetLevel())
}
