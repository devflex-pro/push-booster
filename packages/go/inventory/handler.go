package inventory

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	log     *slog.Logger
	service *Service
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

type createPublisherBody struct {
	Name string `json:"name"`
}

type createSourceBody struct {
	PublisherID string `json:"publisher_id"`
	Name        string `json:"name"`
	Domain      string `json:"domain"`
}

type createVAPIDKeyBody struct {
	Name string `json:"name"`
}

type updateVAPIDKeyStatusBody struct {
	Status string `json:"status"`
}

type attachVAPIDKeyBody struct {
	VAPIDKeyID string `json:"vapid_key_id"`
}

type createCampaignBody struct {
	PublisherID             string         `json:"publisher_id"`
	SourceID                string         `json:"source_id"`
	SourceIDs               []string       `json:"source_ids"`
	AudienceScope           string         `json:"audience_scope"`
	Name                    string         `json:"name"`
	TargetingRules          TargetingRules `json:"targeting_rules"`
	DailyCapPerSubscription int            `json:"daily_cap_per_subscription"`
	TotalCapPerSubscription int            `json:"total_cap_per_subscription"`
}

type updateCampaignStatusBody struct {
	Status string `json:"status"`
}

type createCampaignScheduleBody struct {
	Status           string                           `json:"status"`
	FallbackTimezone string                           `json:"fallback_timezone"`
	GraceMinutes     int                              `json:"grace_minutes"`
	Slots            []createCampaignScheduleSlotBody `json:"slots"`
}

type createCampaignScheduleSlotBody struct {
	LocalTime  string `json:"local_time"`
	DaysOfWeek []int  `json:"days_of_week"`
	Position   int    `json:"position"`
}

type updateCampaignScheduleStatusBody struct {
	Status string `json:"status"`
}

type createCreativeBody struct {
	CampaignID              string `json:"campaign_id"`
	Title                   string `json:"title"`
	Body                    string `json:"body"`
	URL                     string `json:"url"`
	Icon                    string `json:"icon"`
	DailyCapPerSubscription int    `json:"daily_cap_per_subscription"`
	TotalCapPerSubscription int    `json:"total_cap_per_subscription"`
}

type updateCreativeStatusBody struct {
	Status string `json:"status"`
}

type createCreativeProviderConfigBody struct {
	CampaignID     string            `json:"campaign_id"`
	Name           string            `json:"name"`
	ProviderName   string            `json:"provider_name"`
	FetchURL       string            `json:"fetch_url"`
	RequestHeaders map[string]string `json:"request_headers"`
}

type updateCreativeProviderConfigStatusBody struct {
	Status string `json:"status"`
}

func (h *Handler) CreatePublisher(w http.ResponseWriter, r *http.Request) {
	var req createPublisherBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}

	publisher, err := h.service.CreatePublisher(r.Context(), CreatePublisherInput(req))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, publisher)
}

func (h *Handler) ListPublishers(w http.ResponseWriter, r *http.Request) {
	publishers, err := h.service.ListPublishers(r.Context())
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": publishers,
		"total": len(publishers),
	})
}

func (h *Handler) CreateSource(w http.ResponseWriter, r *http.Request) {
	var req createSourceBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}

	source, err := h.service.CreateSource(r.Context(), CreateSourceInput(req))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, source)
}

func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.service.ListSources(r.Context(), r.URL.Query().Get("publisher_id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": sources,
		"total": len(sources),
	})
}

func (h *Handler) SourceSnippet(w http.ResponseWriter, r *http.Request) {
	snippet, err := h.service.SourceSnippet(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"snippet": snippet})
}

func (h *Handler) SourceStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.SourceStats(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, stats)
}

func (h *Handler) CreateVAPIDKey(w http.ResponseWriter, r *http.Request) {
	var req createVAPIDKeyBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}

	key, err := h.service.CreateVAPIDKey(r.Context(), CreateVAPIDKeyInput(req))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, key)
}

func (h *Handler) ListVAPIDKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.service.ListVAPIDKeys(r.Context())
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": keys,
		"total": len(keys),
	})
}

func (h *Handler) UpdateVAPIDKeyStatus(w http.ResponseWriter, r *http.Request) {
	var req updateVAPIDKeyStatusBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}

	key, err := h.service.UpdateVAPIDKeyStatus(r.Context(), chi.URLParam(r, "id"), req.Status)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, key)
}

func (h *Handler) AttachVAPIDKeyToSource(w http.ResponseWriter, r *http.Request) {
	var req attachVAPIDKeyBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}

	source, err := h.service.AttachVAPIDKeyToSource(r.Context(), AttachVAPIDKeyInput{
		SourceID:   chi.URLParam(r, "id"),
		VAPIDKeyID: req.VAPIDKeyID,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, source)
}

func (h *Handler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var req createCampaignBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	campaign, err := h.service.CreateCampaign(r.Context(), CreateCampaignInput(req))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, campaign)
}

func (h *Handler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := h.service.ListCampaigns(r.Context(), r.URL.Query().Get("source_id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": campaigns,
		"total": len(campaigns),
	})
}

func (h *Handler) UpdateCampaignStatus(w http.ResponseWriter, r *http.Request) {
	var req updateCampaignStatusBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	campaign, err := h.service.UpdateCampaignStatus(r.Context(), UpdateCampaignStatusInput{
		ID:     chi.URLParam(r, "id"),
		Status: req.Status,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, campaign)
}

func (h *Handler) ListCampaignSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.service.ListCampaignSchedules(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": schedules,
		"total": len(schedules),
	})
}

func (h *Handler) CreateCampaignSchedule(w http.ResponseWriter, r *http.Request) {
	var req createCampaignScheduleBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	slots := make([]CreateCampaignScheduleSlotInput, 0, len(req.Slots))
	for _, slot := range req.Slots {
		slots = append(slots, CreateCampaignScheduleSlotInput(slot))
	}
	schedule, err := h.service.CreateCampaignSchedule(r.Context(), CreateCampaignScheduleInput{
		CampaignID:       chi.URLParam(r, "id"),
		Status:           req.Status,
		FallbackTimezone: req.FallbackTimezone,
		GraceMinutes:     req.GraceMinutes,
		Slots:            slots,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, schedule)
}

func (h *Handler) UpdateCampaignScheduleStatus(w http.ResponseWriter, r *http.Request) {
	var req updateCampaignScheduleStatusBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	schedule, err := h.service.UpdateCampaignScheduleStatus(r.Context(), UpdateCampaignScheduleStatusInput{
		ID:     chi.URLParam(r, "schedule_id"),
		Status: req.Status,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, schedule)
}

func (h *Handler) ListCampaignScheduleRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := h.service.ListCampaignScheduleRuns(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": runs,
		"total": len(runs),
	})
}

func (h *Handler) EstimateCampaignAudience(w http.ResponseWriter, r *http.Request) {
	estimate, err := h.service.EstimateCampaignAudience(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, estimate)
}

func (h *Handler) CreateCampaignLaunch(w http.ResponseWriter, r *http.Request) {
	launch, err := h.service.CreateCampaignLaunch(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, launch)
}

func (h *Handler) ListCampaignLaunches(w http.ResponseWriter, r *http.Request) {
	launches, err := h.service.ListCampaignLaunches(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": launches,
		"total": len(launches),
	})
}

func (h *Handler) EnqueueCampaignLaunch(w http.ResponseWriter, r *http.Request) {
	launch, err := h.service.EnqueueCampaignLaunch(
		r.Context(),
		chi.URLParam(r, "id"),
		chi.URLParam(r, "launch_id"),
	)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, launch)
}

func (h *Handler) CreateCreative(w http.ResponseWriter, r *http.Request) {
	var req createCreativeBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	creative, err := h.service.CreateCreative(r.Context(), CreateCreativeInput(req))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, creative)
}

func (h *Handler) ListCreatives(w http.ResponseWriter, r *http.Request) {
	creatives, err := h.service.ListCreatives(r.Context(), r.URL.Query().Get("campaign_id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": creatives,
		"total": len(creatives),
	})
}

func (h *Handler) UpdateCreativeStatus(w http.ResponseWriter, r *http.Request) {
	var req updateCreativeStatusBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	creative, err := h.service.UpdateCreativeStatus(r.Context(), UpdateCreativeStatusInput{
		ID:     chi.URLParam(r, "id"),
		Status: req.Status,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, creative)
}

func (h *Handler) CreateCreativeProviderConfig(w http.ResponseWriter, r *http.Request) {
	var req createCreativeProviderConfigBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	config, err := h.service.CreateCreativeProviderConfig(
		r.Context(),
		CreateCreativeProviderConfigInput(req),
	)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, config)
}

func (h *Handler) ListCreativeProviderConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.ListCreativeProviderConfigs(r.Context(), r.URL.Query().Get("campaign_id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": configs,
		"total": len(configs),
	})
}

func (h *Handler) SyncCreativeProviderConfig(w http.ResponseWriter, r *http.Request) {
	log, err := h.service.SyncCreativeProviderConfig(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, log)
}

func (h *Handler) UpdateCreativeProviderConfigStatus(w http.ResponseWriter, r *http.Request) {
	var req updateCreativeProviderConfigStatusBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	config, err := h.service.UpdateCreativeProviderConfigStatus(
		r.Context(),
		UpdateCreativeProviderConfigStatusInput{
			ID:     chi.URLParam(r, "id"),
			Status: req.Status,
		},
	)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, config)
}

func (h *Handler) ListCreativeSyncLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.service.ListCreativeSyncLogs(
		r.Context(),
		r.URL.Query().Get("provider_config_id"),
		r.URL.Query().Get("campaign_id"),
	)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": logs,
		"total": len(logs),
	})
}

func (h *Handler) respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		h.respondError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, ErrNotFound):
		h.respondError(w, http.StatusNotFound, "inventory record not found", err)
	default:
		h.respondError(w, http.StatusInternalServerError, "inventory operation failed", err)
	}
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to write inventory response", slog.String("error", err.Error()))
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		h.log.Warn("inventory request failed", slog.String("error", err.Error()))
	}
	h.respondJSON(w, status, map[string]string{"error": message})
}
