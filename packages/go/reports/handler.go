package reports

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	log     *slog.Logger
	service *Service
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := h.service.Dashboard(
		r.Context(),
		dateRangeFromRequest(r),
	)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, dashboard)
}

func (h *Handler) Performance(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.Performance(
		r.Context(),
		PerformanceInput{
			GroupBy:     r.URL.Query().Get("group_by"),
			DateRange:   dateRangeFromRequest(r),
			SortBy:      r.URL.Query().Get("sort_by"),
			Limit:       intFromQuery(r, "limit", 0),
			PublisherID: r.URL.Query().Get("publisher_id"),
			CampaignID:  r.URL.Query().Get("campaign_id"),
		},
	)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": rows,
		"total": len(rows),
	})
}

func (h *Handler) CreateCostEntry(w http.ResponseWriter, r *http.Request) {
	var req CreateCostEntryInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	entry, err := h.service.CreateCostEntry(r.Context(), req)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, entry)
}

func (h *Handler) ImportCostEntries(w http.ResponseWriter, r *http.Request) {
	reader, closeReader, err := costImportReader(r)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid import file", err)
		return
	}
	defer func() {
		if err := closeReader(); err != nil {
			h.log.Warn("failed to close import file", slog.String("error", err.Error()))
		}
	}()
	result, err := h.service.ImportCostEntries(r.Context(), reader)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, result)
}

func costImportReader(r *http.Request) (io.Reader, func() error, error) {
	if err := r.ParseMultipartForm(2 * 1024 * 1024); err == nil && r.MultipartForm != nil {
		file, _, err := r.FormFile("file")
		if err != nil {
			return nil, func() error { return nil }, err
		}
		return io.LimitReader(file, 2*1024*1024), file.Close, nil
	}
	return io.LimitReader(r.Body, 2*1024*1024), func() error { return nil }, nil
}

func (h *Handler) ListCostEntries(w http.ResponseWriter, r *http.Request) {
	page, err := h.service.ListCostEntries(r.Context(), ListCostEntriesInput{
		Limit:  intFromQuery(r, "limit", 50),
		Offset: intFromQuery(r, "offset", 0),
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"items": page.Items,
		"total": page.Total,
	})
}

func intFromQuery(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}

func dateRangeFromRequest(r *http.Request) DateRange {
	return DateRange{
		From: parseDate(r.URL.Query().Get("date_from")),
		To:   parseDate(r.URL.Query().Get("date_to")),
	}
}

func parseDate(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func (h *Handler) respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		h.respondError(w, http.StatusBadRequest, err.Error(), err)
	default:
		h.respondError(w, http.StatusInternalServerError, "report operation failed", err)
	}
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to write report response", slog.String("error", err.Error()))
	}
}

func (h *Handler) respondError(
	w http.ResponseWriter,
	status int,
	message string,
	err error,
) {
	if err != nil {
		h.log.Warn("report request failed", slog.String("error", err.Error()))
	}
	h.respondJSON(w, status, map[string]string{"error": message})
}
