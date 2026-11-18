package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// @Summary metricsHistory
// @ID metricsHistory
// @Description Read persisted utilization history. scope is host, vm, or interface; target is host, VM ID (or all for global VM statistics), or interface name. VM CPU is normalized to host CPU capacity; global VM CPU is the sum of individual VM CPU shares and VM memory is QEMU resident memory. Rates are bytes per second. Missing baselines are null. Retention is seven days.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags host
// @Security BearerAuth
// @Produce json
// @Param scope query string true "Metric scope" Enums(host,vm,interface)
// @Param target query string true "Metric target"
// @Param range query string false "History range" Enums(1h,6h,24h,7d)
// @Success 200 {array} metrics.Point
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/metrics [get]
func (s *Server) metricsHistory(w http.ResponseWriter, r *http.Request) {
	scope, target := r.URL.Query().Get("scope"), r.URL.Query().Get("target")
	if (scope != "host" && scope != "vm" && scope != "interface") || target == "" || (scope == "host" && target != "host") {
		writeError(w, http.StatusBadRequest, "invalid metric scope or target")
		return
	}

	duration := time.Hour
	switch r.URL.Query().Get("range") {
	case "", "1h":
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	default:
		writeError(w, http.StatusBadRequest, "invalid history range")
		return
	}

	points, err := s.engine.MetricsHistory(r.Context(), scope, target, duration)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, points)
}

// @Summary metricInterfaces
// @ID metricInterfaces
// @Description List host interfaces with persisted network counters.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags host
// @Security BearerAuth
// @Produce json
// @Success 200 {array} string
// @Failure 500 {object} ErrorResponse
// @Router /api/metrics/interfaces [get]
func (s *Server) metricInterfaces(w http.ResponseWriter, r *http.Request) {
	database, err := s.engine.OpenDB(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	names, err := database.MetricInterfaces(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, names)
}

func (s *Server) mountMetrics(r chi.Router) {
	r.Get("/metrics", s.metricsHistory)
	r.Get("/metrics/interfaces", s.metricInterfaces)
}
