package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/devflex-pro/push-booster/packages/go/auth"
	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/postbacks"
	"github.com/devflex-pro/push-booster/packages/go/reports"
	"github.com/devflex-pro/push-booster/packages/go/subscribers"
	"github.com/go-chi/chi/v5"
)

type Options struct {
	AuthHandler       *auth.Handler
	InventoryHandler  *inventory.Handler
	PostbackHandler   *postbacks.Handler
	ReportHandler     *reports.Handler
	SubscriberHandler *subscribers.Handler
}

type PublicOptions struct {
	SubscriberHandler *subscribers.Handler
	PostbackHandler   *postbacks.Handler
}

type PayloadOptions struct {
	SubscriberHandler *subscribers.Handler
}

func NewAdminRouter(opts Options) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", healthHandler)
	r.Get("/readyz", readyHandler)
	if opts.AuthHandler != nil {
		r.Route("/api", func(r chi.Router) {
			r.Post("/auth/request-otp", opts.AuthHandler.RequestOTP)
			r.Post("/auth/verify-otp", opts.AuthHandler.VerifyOTP)
			r.Group(func(r chi.Router) {
				r.Use(opts.AuthHandler.Middleware)
				r.Get("/me", opts.AuthHandler.Me)
				r.Get("/users", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.AuthHandler.ListUsers)).ServeHTTP)
				r.Post("/users/{id}/approve", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.AuthHandler.ApproveUser)).ServeHTTP)
				if opts.InventoryHandler != nil {
					r.Get("/publishers", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListPublishers)).ServeHTTP)
					r.Post("/publishers", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.CreatePublisher)).ServeHTTP)
					r.Get("/sources", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListSources)).ServeHTTP)
					r.Post("/sources", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.CreateSource)).ServeHTTP)
					r.Get("/sources/{id}/snippet", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.SourceSnippet)).ServeHTTP)
					r.Get("/sources/{id}/stats", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.SourceStats)).ServeHTTP)
					r.Post("/sources/{id}/vapid-key", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.AttachVAPIDKeyToSource)).ServeHTTP)
					r.Get("/vapid-keys", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListVAPIDKeys)).ServeHTTP)
					r.Post("/vapid-keys", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.CreateVAPIDKey)).ServeHTTP)
					r.Post("/vapid-keys/{id}/status", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.UpdateVAPIDKeyStatus)).ServeHTTP)
					r.Get("/campaigns", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListCampaigns)).ServeHTTP)
					r.Post("/campaigns", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.CreateCampaign)).ServeHTTP)
					r.Post("/campaigns/{id}/status", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.UpdateCampaignStatus)).ServeHTTP)
					r.Get("/campaigns/{id}/schedules", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListCampaignSchedules)).ServeHTTP)
					r.Post("/campaigns/{id}/schedules", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.CreateCampaignSchedule)).ServeHTTP)
					r.Post("/campaigns/{id}/schedules/{schedule_id}/status", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.UpdateCampaignScheduleStatus)).ServeHTTP)
					r.Get("/campaigns/{id}/schedule-runs", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListCampaignScheduleRuns)).ServeHTTP)
					r.Get("/campaigns/{id}/launches", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListCampaignLaunches)).ServeHTTP)
					r.Post("/campaigns/{id}/launches", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.CreateCampaignLaunch)).ServeHTTP)
					r.Post("/campaigns/{id}/launches/estimate", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.EstimateCampaignAudience)).ServeHTTP)
					r.Post("/campaigns/{id}/launches/{launch_id}/enqueue", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.EnqueueCampaignLaunch)).ServeHTTP)
					r.Get("/creatives", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListCreatives)).ServeHTTP)
					r.Post("/creatives", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.CreateCreative)).ServeHTTP)
					r.Post("/creatives/{id}/status", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.UpdateCreativeStatus)).ServeHTTP)
					r.Get("/creative-provider-configs", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListCreativeProviderConfigs)).ServeHTTP)
					r.Post("/creative-provider-configs", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.CreateCreativeProviderConfig)).ServeHTTP)
					r.Post("/creative-provider-configs/{id}/sync", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.SyncCreativeProviderConfig)).ServeHTTP)
					r.Post("/creative-provider-configs/{id}/status", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.UpdateCreativeProviderConfigStatus)).ServeHTTP)
					r.Get("/creative-sync-logs", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.InventoryHandler.ListCreativeSyncLogs)).ServeHTTP)
				}
				if opts.SubscriberHandler != nil {
					r.Get("/reports/campaigns", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.SubscriberHandler.CampaignReport)).ServeHTTP)
				}
				if opts.ReportHandler != nil {
					r.Get("/reports/dashboard", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.ReportHandler.Dashboard)).ServeHTTP)
					r.Get("/reports/performance", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.ReportHandler.Performance)).ServeHTTP)
					r.Get("/costs", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.ReportHandler.ListCostEntries)).ServeHTTP)
					r.Post("/costs", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.ReportHandler.CreateCostEntry)).ServeHTTP)
					r.Post("/costs/import", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.ReportHandler.ImportCostEntries)).ServeHTTP)
				}
				if opts.PostbackHandler != nil {
					r.Get("/postback-configs", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.PostbackHandler.ListConfigs)).ServeHTTP)
					r.Post("/postback-configs", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.PostbackHandler.CreateConfig)).ServeHTTP)
					r.Post("/postback-configs/{id}/status", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.PostbackHandler.UpdateConfigStatus)).ServeHTTP)
					r.Get("/postbacks", opts.AuthHandler.AdminOnly(http.HandlerFunc(opts.PostbackHandler.RecentEvents)).ServeHTTP)
				}
			})
		})
	}

	return r
}

func NewPublicRouter(opts PublicOptions) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", healthHandler)
	r.Get("/readyz", readyHandler)
	if opts.SubscriberHandler != nil {
		r.Get("/api/web-push/vapid-public-key", opts.SubscriberHandler.VAPIDPublicKey)
		r.Get("/api/sdk/sources/{source_id}.js", opts.SubscriberHandler.SDKScript)
		r.Options("/api/sdk/config", opts.SubscriberHandler.Preflight)
		r.Get("/api/sdk/config", opts.SubscriberHandler.SDKConfig)
		r.Options("/api/subscribe", opts.SubscriberHandler.Preflight)
		r.Post("/api/subscribe", opts.SubscriberHandler.Subscribe)
		r.Options("/api/sw/events", opts.SubscriberHandler.Preflight)
		r.Post("/api/sw/events", opts.SubscriberHandler.ServiceWorkerEvent)
		r.Get("/api/click/{delivery_id}", opts.SubscriberHandler.ClickRedirect)
	}
	if opts.PostbackHandler != nil {
		r.Get("/v1/postbacks/{postback_config_id}", opts.PostbackHandler.Ingest)
		r.Post("/v1/postbacks/{postback_config_id}", opts.PostbackHandler.Ingest)
	}

	return r
}

func NewPayloadRouter(opts PayloadOptions) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", healthHandler)
	r.Get("/readyz", readyHandler)
	if opts.SubscriberHandler != nil {
		r.Post("/api/push/triggers", opts.SubscriberHandler.CreatePushTrigger)
		r.Post("/api/push/payload", opts.SubscriberHandler.PushPayload)
	}

	return r
}

func NewRouter(opts Options) http.Handler {
	return NewAdminRouter(opts)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	if err := writeJSON(w, http.StatusOK, map[string]string{"status": "ready"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}
