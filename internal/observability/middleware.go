package observability

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Middleware struct {
	logger          *slog.Logger
	httpRequests    *prometheus.CounterVec
	httpDurationSec *prometheus.HistogramVec
}

func NewMiddleware(logger *slog.Logger, registerer prometheus.Registerer) *Middleware {
	httpRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_http_requests_total",
			Help: "Total HTTP requests handled by endpoint, method, and status.",
		},
		[]string{"path", "method", "status"},
	)
	httpDurationSec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chatbot_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds by endpoint and method.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	registerer.MustRegister(httpRequests, httpDurationSec)

	return &Middleware{
		logger:          logger,
		httpRequests:    httpRequests,
		httpDurationSec: httpDurationSec,
	}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := incomingOrGeneratedRequestID(r)
		ctx := WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		ww := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		ww.Header().Set("X-Request-ID", requestID)

		defer func() {
			elapsed := time.Since(start).Seconds()
			path := r.URL.Path
			status := strconv.Itoa(ww.statusCode)
			clientIP := extractClientIP(r)

			m.httpRequests.WithLabelValues(path, r.Method, status).Inc()
			m.httpDurationSec.WithLabelValues(path, r.Method).Observe(elapsed)

			m.logger.Info("http_request",
				slog.String("request_id", requestID),
				slog.String("client_ip", clientIP),
				slog.String("user_agent", r.UserAgent()),
				slog.String("method", r.Method),
				slog.String("path", path),
				slog.Int("status", ww.statusCode),
				slog.Float64("duration_sec", elapsed),
			)
		}()

		defer func() {
			if rec := recover(); rec != nil {
				m.logger.Error("panic_recovered",
					slog.String("request_id", requestID),
					slog.Any("panic", rec),
				)
				http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(ww, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func incomingOrGeneratedRequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buf)
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
