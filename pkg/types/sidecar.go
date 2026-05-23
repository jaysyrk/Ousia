package types

import "time"

type SidecarConfig struct {
	ServiceID       string
	InboundPort     int
	OutboundPort    int
	LocalPort       int
	AdminURL        string
	RefreshInterval time.Duration
}

type SidecarUpstream struct {
	Name    string
	Address string
}
