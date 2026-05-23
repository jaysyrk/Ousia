package types

type SidecarConfig struct {
	ServiceID    string
	InboundPort  int
	OutboundPort int
	LocalPort    int
	Upstreams    []SidecarUpstream
}

type SidecarUpstream struct {
	Name    string
	Address string
}
