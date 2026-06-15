package main

import (
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/yimsoijoi/rama-chatbot/internal/domain/repository"
	"github.com/yimsoijoi/rama-chatbot/internal/infrastructure/config"
	"github.com/yimsoijoi/rama-chatbot/internal/infrastructure/sqlite"
	httpHandler "github.com/yimsoijoi/rama-chatbot/internal/interface/http"
	"github.com/yimsoijoi/rama-chatbot/internal/observability"
	"github.com/yimsoijoi/rama-chatbot/internal/usecase"

	"github.com/line/line-bot-sdk-go/v8/linebot"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := getenv("BOT_CONFIG_PATH", "configs/bot.yaml")
	port := getenv("PORT", "8080")
	enablePprof := getenv("ENABLE_PPROF", "true")
	dedupTTL := getenvDuration("EVENT_DEDUP_TTL", 24*time.Hour)
	dbPath := os.Getenv("DB_PATH") // optional; empty = no persistence

	logger := observability.NewLogger()

	cfg, err := config.LoadBotConfig(configPath)
	if err != nil {
		logger.Error("startup_config_failed", "error", err.Error(), "config_path", configPath)
		log.Fatalf("failed to load bot config: %v", err)
	}

	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_TOKEN"),
	)
	if err != nil {
		logger.Error("startup_line_client_failed", "error", err.Error())
		log.Fatalf("failed to create line bot client: %v", err)
	}

	repo := config.NewKnowledgeRepo(cfg)

	var userDiagRepo repository.UserDiagnosisRepository
	var faqRepo repository.FAQRepository
	if dbPath != "" {
		sqliteRepo, err := sqlite.New(dbPath)
		if err != nil {
			logger.Error("startup_sqlite_failed", "error", err.Error(), "db_path", dbPath)
			log.Fatalf("failed to open sqlite db: %v", err)
		}
		defer sqliteRepo.Close()
		userDiagRepo = sqliteRepo
		faqRepo = sqliteRepo
		logger.Info("startup_sqlite_ready", "db_path", dbPath)
	} else {
		logger.Info("startup_sqlite_skipped", "reason", "DB_PATH not set; user diagnosis will not be persisted")
	}

	uc := usecase.NewReplyUsecaseWithRepos(repo, userDiagRepo, faqRepo)
	dedup := observability.NewInMemoryEventDedupCache(dedupTTL)
	h := httpHandler.NewLineHandlerWithDedup(bot, uc, logger, dedup)

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	mw := observability.NewMiddleware(logger, registry)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/webhook", h.Webhook)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	if enablePprof == "true" {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mw.Wrap(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("startup_server_ready", "port", port, "metrics_path", "/metrics", "pprof_enabled", enablePprof, "event_dedup_ttl", dedupTTL.String())
	log.Printf("line chatbot server listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		logger.Error("server_stopped", "error", err.Error())
		log.Fatalf("server error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
