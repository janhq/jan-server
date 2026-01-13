package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Artifact metrics
var (
	// Artifact creation counters
	ArtifactsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "artifacts_total",
			Help:      "Total number of artifacts created",
		},
		[]string{"content_type", "retention_policy"},
	)

	// Artifact size histogram
	ArtifactSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "artifact_size_bytes",
			Help:      "Artifact size in bytes",
			Buckets:   []float64{1024, 10240, 102400, 1048576, 10485760, 104857600}, // 1KB to 100MB
		},
		[]string{"content_type"},
	)

	// Artifact version counters
	ArtifactVersionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "artifact_versions_total",
			Help:      "Total number of artifact versions created",
		},
		[]string{"content_type"},
	)

	// Active artifacts gauge (non-expired)
	ArtifactsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "artifacts_active",
			Help:      "Number of active (non-expired) artifacts",
		},
		[]string{"content_type"},
	)

	// Expired artifacts deleted counter
	ArtifactsExpiredDeleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "artifacts_expired_deleted_total",
			Help:      "Total number of expired artifacts deleted",
		},
	)

	// Artifact downloads counter
	ArtifactDownloadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "artifact_downloads_total",
			Help:      "Total number of artifact downloads",
		},
		[]string{"content_type"},
	)

	// Total artifact storage bytes gauge
	ArtifactStorageTotalBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "jan",
			Subsystem: "response_api",
			Name:      "artifact_storage_total_bytes",
			Help:      "Total bytes stored in artifacts",
		},
	)
)

// RecordArtifactCreated records a new artifact creation
func RecordArtifactCreated(contentType, retentionPolicy string, sizeBytes int64) {
	ArtifactsTotal.WithLabelValues(contentType, retentionPolicy).Inc()
	ArtifactSizeBytes.WithLabelValues(contentType).Observe(float64(sizeBytes))
	ArtifactsActive.WithLabelValues(contentType).Inc()
}

// RecordArtifactVersion records a new artifact version
func RecordArtifactVersion(contentType string, sizeBytes int64) {
	ArtifactVersionsTotal.WithLabelValues(contentType).Inc()
	ArtifactSizeBytes.WithLabelValues(contentType).Observe(float64(sizeBytes))
}

// RecordArtifactDeleted records artifact deletion
func RecordArtifactDeleted(contentType string) {
	ArtifactsActive.WithLabelValues(contentType).Dec()
}

// RecordArtifactsExpiredDeleted records expired artifacts deletion
func RecordArtifactsExpiredDeleted(count int64) {
	ArtifactsExpiredDeleted.Add(float64(count))
}

// RecordArtifactDownload records an artifact download
func RecordArtifactDownload(contentType string) {
	ArtifactDownloadsTotal.WithLabelValues(contentType).Inc()
}

// SetArtifactStorageTotal sets the total storage used by artifacts
func SetArtifactStorageTotal(bytes int64) {
	ArtifactStorageTotalBytes.Set(float64(bytes))
}
