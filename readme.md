# PeeringMon Controller

This golang code will be a bgp client that will cycle bgp announcements to a
defined list of peers and provide the expected state of these bgp routes in a
prom exporter.

this is meant to be in conjunction with peeringmon_exporter and a prom server
to monitor bgp routes for the peering testbed.

each announcement has the site's configured id value injected in the community
to test for zombie routes

### known issues
Memory leaks: We suggest that if you wish to deploy this, you configure the
peers to not export any routes to the controller as there is a memory leak.

