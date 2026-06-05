package subscribers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	log     *slog.Logger
	service *Service
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w, r.Header.Get("Origin"))
	var req SubscribeInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	req.UserAgent = r.UserAgent()
	req.RequestOrigin = r.Header.Get("Origin")
	req.RequestReferrer = r.Header.Get("Referer")
	req.Targeting = requestTargeting(r)
	req.Targeting.Timezone = req.Timezone
	result, err := h.service.Subscribe(r.Context(), req)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusAccepted, result)
}

func requestTargeting(r *http.Request) Targeting {
	return Targeting{
		IP:                requestIP(r),
		Language:          r.Header.Get("Accept-Language"),
		UAPlatform:        r.Header.Get("Sec-CH-UA-Platform"),
		UAPlatformVersion: r.Header.Get("Sec-CH-UA-Platform-Version"),
		UAMobile:          strings.TrimSpace(r.Header.Get("Sec-CH-UA-Mobile")) == "?1",
		UAFullVersion:     firstFullVersion(r.Header.Get("Sec-CH-UA-Full-Version-List")),
		UAArch:            r.Header.Get("Sec-CH-UA-Arch"),
		UABitness:         r.Header.Get("Sec-CH-UA-Bitness"),
	}
}

func requestIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func firstFullVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	first := strings.TrimSpace(parts[0])
	versionIndex := strings.LastIndex(first, "v=")
	if versionIndex < 0 {
		return first
	}
	return strings.Trim(first[versionIndex+2:], `"`)
}

func (h *Handler) ServiceWorkerEvent(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w, r.Header.Get("Origin"))
	var req ServiceWorkerEventInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	req.UserAgent = r.UserAgent()
	req.RequestOrigin = r.Header.Get("Origin")
	req.RequestReferrer = r.Header.Get("Referer")
	if err := h.service.TrackServiceWorkerEvent(r.Context(), req); err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *Handler) ClickRedirect(w http.ResponseWriter, r *http.Request) {
	deliveryID := strings.TrimSpace(chi.URLParam(r, "delivery_id"))
	targetURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if deliveryID == "" {
		h.respondError(w, http.StatusBadRequest, "delivery_id is required", nil)
		return
	}
	if !validRedirectURL(targetURL) {
		targetURL = "/"
	}
	http.Redirect(
		w,
		r,
		targetURL,
		http.StatusFound,
	)
}

func (h *Handler) VAPIDPublicKey(w http.ResponseWriter, _ *http.Request) {
	publicKey, err := h.service.VAPIDPublicKey()
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"public_key": publicKey})
}

func (h *Handler) SDKConfig(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	h.setCORS(w, origin)
	cfg, err := h.service.SDKConfig(r.Context(), SDKConfigInput{
		SourceID:        r.URL.Query().Get("source_id"),
		RequestOrigin:   origin,
		RequestReferrer: r.Header.Get("Referer"),
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, cfg)
}

func (h *Handler) SDKScript(w http.ResponseWriter, r *http.Request) {
	script, err := h.service.SDKScript(r.Context(), SDKScriptInput{
		SourceID:        chi.URLParam(r, "source_id"),
		RequestOrigin:   r.Header.Get("Origin"),
		RequestReferrer: r.Header.Get("Referer"),
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(script))
}

func (h *Handler) Preflight(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w, r.Header.Get("Origin"))
	w.WriteHeader(http.StatusNoContent)
}

type pushPayloadBody struct {
	TriggerID string `json:"trigger_id"`
}

func (h *Handler) PushPayload(w http.ResponseWriter, r *http.Request) {
	var req pushPayloadBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	payload, err := h.service.PushPayload(r.Context(), PushPayloadRequest{
		TriggerID: req.TriggerID,
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, payload)
}

type createTriggerBody struct {
	SubscriptionID string `json:"subscription_id"`
	SourceID       string `json:"source_id"`
	CampaignID     string `json:"campaign_id"`
	TTLSeconds     int64  `json:"ttl_seconds"`
}

func (h *Handler) CreatePushTrigger(w http.ResponseWriter, r *http.Request) {
	var req createTriggerBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	var ttl time.Duration
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}
	trigger, err := h.service.CreateTrigger(r.Context(), CreateTriggerInput{
		SubscriptionID: req.SubscriptionID,
		SourceID:       req.SourceID,
		CampaignID:     req.CampaignID,
		TTL:            ttl,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, trigger)
}

func (h *Handler) CampaignReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.CampaignReport(r.Context(), r.URL.Query().Get("campaign_id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, report)
}

func (h *Handler) respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		h.respondError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, ErrForbiddenOrigin):
		h.respondError(w, http.StatusForbidden, "source origin is not allowed", err)
	case errors.Is(err, ErrVAPIDNotReady):
		h.respondError(w, http.StatusServiceUnavailable, "vapid public key is not configured", err)
	case errors.Is(err, ErrTriggerStoreNotReady):
		h.respondError(w, http.StatusServiceUnavailable, "delivery trigger store is not configured", err)
	case errors.Is(err, ErrPayloadSuppressed):
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, inventory.ErrNotFound):
		h.respondError(w, http.StatusNotFound, "subscription or source not found", err)
	default:
		h.respondError(w, http.StatusInternalServerError, "subscriber operation failed", err)
	}
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to write subscriber response", slog.String("error", err.Error()))
	}
}

func (h *Handler) setCORS(w http.ResponseWriter, origin string) {
	w.Header().Add("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		h.log.Warn("subscriber request failed", slog.String("error", err.Error()))
	}
	h.respondJSON(w, status, map[string]string{"error": message})
}

func validRedirectURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.IsAbs() {
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}
	return strings.HasPrefix(value, "/")
}
