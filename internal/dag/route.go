package dag

import (
	"time"
)

// RouteOptions defines optional route defaults
type RouteOptions struct {
	ResponseTimeout time.Duration
}

// RouteLimits defines optional route defaults
type RouteLimits struct {
	ResponseTimeout time.Duration
}
