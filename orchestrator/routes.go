package main

import "github.com/tedsuo/rata"

const (
	// Start pusher orchestration
	StartPushersRoute = "StartPushers"

	// Receive Pusher Updates
	PostUpdateRoute = "PostUpdate"
)

var Routes = rata.Routes{
	// Start pusher orchestration
	{Path: "/v1/diego-perf/start", Method: "GET", Name: StartPushersRoute},

	// Receive Pusher Updates
	{Path: "/v1/diego-perf/update", Method: "POST", Name: PostUpdateRoute},
}
