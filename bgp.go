package main

import (
	"context"

	api "github.com/osrg/gobgp/v3/api"
	apb "google.golang.org/protobuf/types/known/anypb"
)

func bgpAnnounce(site string, prefix string) error {
	// establish connection with peer
	n := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: "172.17.0.2",
			PeerAsn:         65002,
		},
	}

	if err := s.AddPeer(context.Background(), &api.AddPeerRequest{
		Peer: n,
	}); err != nil {
		return err
	}

	// add routes
	nlri, _ := apb.New(&api.IPAddressPrefix{
		Prefix:    "10.0.0.0",
		PrefixLen: 24,
	})

	a1, _ := apb.New(&api.OriginAttribute{
		Origin: 0,
	})
	a2, _ := apb.New(&api.NextHopAttribute{
		NextHop: "10.0.0.1",
	})
	a3, _ := apb.New(&api.AsPathAttribute{
		Segments: []*api.AsSegment{
			{
				Type:    2,
				Numbers: []uint32{6762, 39919, 65000, 35753, 65000},
			},
		},
	})
	attrs := []*apb.Any{a1, a2, a3}

	_, err := s.AddPath(context.Background(), &api.AddPathRequest{
		Path: &api.Path{
			Family: &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
			Nlri:   nlri,
			Pattrs: attrs,
		},
	})
	if err != nil {
		return err
	}

	return nil

}

func bgpWithdraw() error {
	// make withdraw
	return nil

}
