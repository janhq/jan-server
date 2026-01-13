package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Plan execution metrics
var (
	// Plan status counters
	PlansTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "plans_total",
			Help:      "Total number of plans created",
		},
		[]string{"agent_type", "status"},
	)

	// Plan execution duration histogram
	PlanDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "plan_duration_seconds",
			Help:      "Plan execution duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800},
		},
		[]string{"agent_type"},
	)

	// Plan steps counters
	PlanStepsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "plan_steps_total",
			Help:      "Total number of plan steps executed",
		},
		[]string{"action", "status"},
	)

	// Plan step duration histogram
	PlanStepDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "plan_step_duration_seconds",
			Help:      "Plan step execution duration in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"action"},
	)

	// Plan retry counters
	PlanRetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "plan_retries_total",
			Help:      "Total number of step retries",
		},
		[]string{"action", "error_code"},
	)

	// Active plans gauge
	PlansActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "plans_active",
			Help:      "Number of currently active plans",
		},
		[]string{"agent_type"},
	)

	// Plans waiting for user input gauge
	PlansWaitingForUser = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "plans_waiting_for_user",
			Help:      "Number of plans waiting for user input",
		},
	)

	// Plan progress histogram
	PlanProgress = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "plan_progress_on_completion",
			Help:      "Plan progress percentage when reaching terminal state",
			Buckets:   []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		},
		[]string{"status"},
	)
)

// RecordPlanCreated records a new plan creation
func RecordPlanCreated(agentType string) {
	PlansTotal.WithLabelValues(agentType, "created").Inc()
	PlansActive.WithLabelValues(agentType).Inc()
}

// RecordPlanCompleted records a plan reaching terminal state
func RecordPlanCompleted(agentType, status string, durationSec, progress float64) {
	PlansTotal.WithLabelValues(agentType, status).Inc()
	PlansActive.WithLabelValues(agentType).Dec()
	PlanDuration.WithLabelValues(agentType).Observe(durationSec)
	PlanProgress.WithLabelValues(status).Observe(progress)
}

// RecordPlanStep records a step execution
func RecordPlanStep(action, status string, durationSec float64) {
	PlanStepsTotal.WithLabelValues(action, status).Inc()
	PlanStepDuration.WithLabelValues(action).Observe(durationSec)
}

// RecordPlanRetry records a step retry
func RecordPlanRetry(action, errorCode string) {
	PlanRetriesTotal.WithLabelValues(action, errorCode).Inc()
}

// SetPlansWaitingForUser sets the number of plans waiting for user input
func SetPlansWaitingForUser(count int) {
	PlansWaitingForUser.Set(float64(count))
}
