package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PRCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pr_created_total",
		Help: "Total number of created PRs",
	})

	TeamCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "team_created_total",
		Help: "Total number of created teams",
	})

	TeamDeactivatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "team_deactivated_total",
		Help: "Total number of team deactivations",
	})

	PRReassignedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pr_reassigned_total",
		Help: "Total number of PR reviewer reassignments",
	})
)
