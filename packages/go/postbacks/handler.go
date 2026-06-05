package postbacks

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	log     *slog.Logger
	service *Service
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateConfigInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	cfg, err := h.service.CreateConfig(r.Context(), req)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, cfg)
}

func (h *Handler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.ListConfigs(r.Context())
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": configs,
		"total": len(configs),
	})
}

type updateStatusBody struct {
	Status string `json:"status"`
}

func (h *Handler) UpdateConfigStatus(w http.ResponseWriter, r *http.Request) {
	var req updateStatusBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	cfg, err := h.service.UpdateConfigStatus(r.Context(), UpdateConfigStatusInput{
		ID:     chi.URLParam(r, "id"),
		Status: req.Status,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, cfg)
}

func (h *Handler) RecentEvents(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 50
	}
	events, err := h.service.RecentEvents(r.Context(), RecentEventsInput{
		PostbackConfigID: r.URL.Query().Get("postback_config_id"),
		Limit:            limit,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": events,
		"total": len(events),
	})
}

func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "read postback body failed", err)
		return
	}
	form := parseBodyValues(r.Header.Get("Content-Type"), string(rawBody))
	event, err := h.service.Ingest(r.Context(), IngestInput{
		ConfigID: chi.URLParam(r, "postback_config_id"),
		Method:   r.Method,
		Query:    r.URL.Query(),
		Form:     form,
		RawBody:  string(rawBody),
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusAccepted, event)
}

func parseBodyValues(contentType string, rawBody string) url.Values {
	if !strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return url.Values{}
	}
	values, err := url.ParseQuery(rawBody)
	if err != nil {
		return url.Values{}
	}
	return values
}

func (h *Handler) respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		h.respondError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, ErrForbidden):
		h.respondError(w, http.StatusForbidden, "postback token validation failed", err)
	case errors.Is(err, ErrDuplicate):
		h.respondError(w, http.StatusConflict, "duplicate postback", err)
	case errors.Is(err, ErrNotFound):
		h.respondError(w, http.StatusNotFound, "postback record not found", err)
	default:
		h.respondError(w, http.StatusInternalServerError, "postback operation failed", err)
	}
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to write postback response", slog.String("error", err.Error()))
	}
}

func (h *Handler) respondError(
	w http.ResponseWriter,
	status int,
	message string,
	err error,
) {
	if err != nil {
		h.log.Warn("postback request failed", slog.String("error", err.Error()))
	}
	h.respondJSON(w, status, map[string]string{"error": message})
}
