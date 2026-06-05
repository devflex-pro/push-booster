package reports

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidInput = errors.New("invalid report input")

const (
	GroupDate      = "date"
	GroupPublisher = "publisher"
	GroupSource    = "source"
	GroupCampaign  = "campaign"
	GroupCreative  = "creative"
)

type DateRange struct {
	From time.Time
	To   time.Time
}

type CostEntry struct {
	ID            string    `json:"id"`
	Date          string    `json:"date"`
	PublisherID   string    `json:"publisher_id,omitempty"`
	PublisherName string    `json:"publisher_name,omitempty"`
	SourceID      string    `json:"source_id,omitempty"`
	SourceName    string    `json:"source_name,omitempty"`
	CampaignID    string    `json:"campaign_id,omitempty"`
	CampaignName  string    `json:"campaign_name,omitempty"`
	CreativeID    string    `json:"creative_id,omitempty"`
	CreativeName  string    `json:"creative_name,omitempty"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateCostEntryInput struct {
	Date        string  `json:"date"`
	PublisherID string  `json:"publisher_id"`
	SourceID    string  `json:"source_id"`
	CampaignID  string  `json:"campaign_id"`
	CreativeID  string  `json:"creative_id"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Note        string  `json:"note"`
}

type ListCostEntriesInput struct {
	Limit  int
	Offset int
}

type CostEntryPage struct {
	Items []CostEntry
	Total int
}

type MetricRow struct {
	Key         string
	Revenue     float64
	Conversions int64
	Sent        int64
	Shown       int64
	Clicks      int64
	Closed      int64
}

type CostRow struct {
	Key   string
	Spend float64
}

type CostAllocationRow struct {
	PublisherID string
	SourceID    string
	Spend       float64
}

type PerformanceRow struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	GroupBy     string  `json:"group_by"`
	Spend       float64 `json:"spend"`
	Revenue     float64 `json:"revenue"`
	Profit      float64 `json:"profit"`
	ROI         float64 `json:"roi"`
	Conversions int64   `json:"conversions"`
	Sent        int64   `json:"sent"`
	Shown       int64   `json:"shown"`
	Clicks      int64   `json:"clicks"`
	Closed      int64   `json:"closed"`
	CTR         float64 `json:"ctr"`
	CR          float64 `json:"cr"`
}

type PerformanceInput struct {
	GroupBy     string
	DateRange   DateRange
	SortBy      string
	Limit       int
	PublisherID string
	CampaignID  string
}

type Dashboard struct {
	Spend       float64          `json:"spend"`
	Revenue     float64          `json:"revenue"`
	Profit      float64          `json:"profit"`
	ROI         float64          `json:"roi"`
	Conversions int64            `json:"conversions"`
	Sent        int64            `json:"sent"`
	Shown       int64            `json:"shown"`
	Clicks      int64            `json:"clicks"`
	Closed      int64            `json:"closed"`
	CTR         float64          `json:"ctr"`
	CR          float64          `json:"cr"`
	Rows        []PerformanceRow `json:"rows"`
}

type CostImportResult struct {
	Inserted int         `json:"inserted"`
	Entries  []CostEntry `json:"entries"`
}

type CostStore interface {
	CreateCostEntry(ctx context.Context, input CreateCostEntryInput) (CostEntry, error)
	ListCostEntries(ctx context.Context, input ListCostEntriesInput) (CostEntryPage, error)
	CostAllocations(ctx context.Context, dateRange DateRange) ([]CostAllocationRow, error)
	CostsByGroup(ctx context.Context, groupBy string, dateRange DateRange) ([]CostRow, error)
	DimensionNames(ctx context.Context, groupBy string, keys []string) (map[string]string, error)
	PublisherIDsBySource(ctx context.Context, sourceIDs []string) (map[string]string, error)
	SourceIDsByCampaign(ctx context.Context, campaignIDs []string) (map[string]string, error)
	SourceIDsByCreative(ctx context.Context, creativeIDs []string) (map[string]string, error)
	CampaignIDsByCreative(ctx context.Context, creativeIDs []string) (map[string]string, error)
}

type MetricStore interface {
	MetricsByGroup(ctx context.Context, groupBy string, dateRange DateRange) ([]MetricRow, error)
}

type Service struct {
	costs   CostStore
	metrics MetricStore
}

func NewService(costs CostStore, metrics MetricStore) *Service {
	return &Service{costs: costs, metrics: metrics}
}

func (s *Service) CreateCostEntry(
	ctx context.Context,
	input CreateCostEntryInput,
) (CostEntry, error) {
	input, err := normalizeCostEntry(input)
	if err != nil {
		return CostEntry{}, err
	}
	return s.costs.CreateCostEntry(ctx, input)
}

func (s *Service) ImportCostEntries(
	ctx context.Context,
	reader io.Reader,
) (CostImportResult, error) {
	inputs, err := parseCostCSV(reader)
	if err != nil {
		return CostImportResult{}, err
	}
	entries := make([]CostEntry, 0, len(inputs))
	for _, input := range inputs {
		entry, err := s.costs.CreateCostEntry(ctx, input)
		if err != nil {
			return CostImportResult{}, fmt.Errorf("import cost entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return CostImportResult{
		Inserted: len(entries),
		Entries:  entries,
	}, nil
}

func (s *Service) ListCostEntries(
	ctx context.Context,
	input ListCostEntriesInput,
) (CostEntryPage, error) {
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 50
	}
	if input.Offset < 0 {
		input.Offset = 0
	}
	return s.costs.ListCostEntries(ctx, input)
}

func (s *Service) Performance(
	ctx context.Context,
	input PerformanceInput,
) ([]PerformanceRow, error) {
	groupBy, err := normalizeGroup(input.GroupBy)
	if err != nil {
		return nil, err
	}
	dateRange := normalizeDateRange(input.DateRange)
	metricRows, err := s.metricRows(ctx, groupBy, dateRange)
	if err != nil {
		return nil, err
	}
	metricRows, err = s.filterMetricRows(ctx, groupBy, metricRows, input)
	if err != nil {
		return nil, err
	}
	costRows, err := s.costRows(ctx, groupBy, dateRange, metricRows)
	if err != nil {
		return nil, err
	}
	costRows, err = s.filterCostRows(ctx, groupBy, costRows, input)
	if err != nil {
		return nil, err
	}
	rows := map[string]PerformanceRow{}
	for _, metric := range metricRows {
		if emptyMetric(metric) {
			continue
		}
		row := rows[metric.Key]
		row.Key = metric.Key
		row.GroupBy = groupBy
		row.Revenue = metric.Revenue
		row.Conversions = metric.Conversions
		row.Sent = metric.Sent
		row.Shown = metric.Shown
		row.Clicks = metric.Clicks
		row.Closed = metric.Closed
		rows[metric.Key] = calculate(row)
	}
	for _, cost := range costRows {
		if cost.Spend == 0 {
			continue
		}
		row := rows[cost.Key]
		row.Key = cost.Key
		row.GroupBy = groupBy
		row.Spend = cost.Spend
		rows[cost.Key] = calculate(row)
	}
	result := make([]PerformanceRow, 0, len(rows))
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Key == "" || row.Key == "null" {
			continue
		}
		keys = append(keys, row.Key)
		result = append(result, row)
	}
	names, err := s.costs.DimensionNames(ctx, groupBy, keys)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Name = names[result[i].Key]
	}
	sortPerformanceRows(result, input.SortBy)
	if input.Limit > 0 && len(result) > input.Limit {
		result = result[:input.Limit]
	}
	return result, nil
}

func (s *Service) filterMetricRows(
	ctx context.Context,
	groupBy string,
	rows []MetricRow,
	input PerformanceInput,
) ([]MetricRow, error) {
	switch {
	case input.PublisherID != "" && groupBy == GroupSource:
		keys := metricKeys(rows)
		publishers, err := s.costs.PublisherIDsBySource(ctx, keys)
		if err != nil {
			return nil, err
		}
		return filterMetricRows(rows, func(row MetricRow) bool {
			return publishers[row.Key] == input.PublisherID
		}), nil
	case input.CampaignID != "" && groupBy == GroupCreative:
		keys := metricKeys(rows)
		campaigns, err := s.costs.CampaignIDsByCreative(ctx, keys)
		if err != nil {
			return nil, err
		}
		return filterMetricRows(rows, func(row MetricRow) bool {
			return campaigns[row.Key] == input.CampaignID
		}), nil
	default:
		return rows, nil
	}
}

func (s *Service) filterCostRows(
	ctx context.Context,
	groupBy string,
	rows []CostRow,
	input PerformanceInput,
) ([]CostRow, error) {
	switch {
	case input.PublisherID != "" && groupBy == GroupSource:
		keys := costKeys(rows)
		publishers, err := s.costs.PublisherIDsBySource(ctx, keys)
		if err != nil {
			return nil, err
		}
		return filterCostRows(rows, func(row CostRow) bool {
			return publishers[row.Key] == input.PublisherID
		}), nil
	case input.CampaignID != "" && groupBy == GroupCreative:
		keys := costKeys(rows)
		campaigns, err := s.costs.CampaignIDsByCreative(ctx, keys)
		if err != nil {
			return nil, err
		}
		return filterCostRows(rows, func(row CostRow) bool {
			return campaigns[row.Key] == input.CampaignID
		}), nil
	default:
		return rows, nil
	}
}

func (s *Service) Dashboard(ctx context.Context, dateRange DateRange) (Dashboard, error) {
	rows, err := s.Performance(ctx, PerformanceInput{
		GroupBy:   GroupSource,
		DateRange: dateRange,
		SortBy:    "profit",
		Limit:     50,
	})
	if err != nil {
		return Dashboard{}, err
	}
	dashboard := Dashboard{Rows: rows}
	for _, row := range rows {
		dashboard.Spend += row.Spend
		dashboard.Revenue += row.Revenue
		dashboard.Conversions += row.Conversions
		dashboard.Sent += row.Sent
		dashboard.Shown += row.Shown
		dashboard.Clicks += row.Clicks
		dashboard.Closed += row.Closed
	}
	dashboard.Profit = dashboard.Revenue - dashboard.Spend
	dashboard.ROI = ratio(dashboard.Profit, dashboard.Spend) * 100
	dashboard.CTR = ratio(float64(dashboard.Clicks), float64(dashboard.Shown)) * 100
	dashboard.CR = ratio(float64(dashboard.Conversions), float64(dashboard.Clicks)) * 100
	return dashboard, nil
}

func (s *Service) metricRows(
	ctx context.Context,
	groupBy string,
	dateRange DateRange,
) ([]MetricRow, error) {
	if groupBy != GroupPublisher {
		return s.metrics.MetricsByGroup(ctx, groupBy, dateRange)
	}
	sourceRows, err := s.metrics.MetricsByGroup(ctx, GroupSource, dateRange)
	if err != nil {
		return nil, err
	}
	sourceIDs := make([]string, 0, len(sourceRows))
	for _, row := range sourceRows {
		sourceIDs = append(sourceIDs, row.Key)
	}
	publishers, err := s.costs.PublisherIDsBySource(ctx, sourceIDs)
	if err != nil {
		return nil, err
	}
	rows := map[string]MetricRow{}
	for _, sourceRow := range sourceRows {
		publisherID := publishers[sourceRow.Key]
		if publisherID == "" {
			continue
		}
		row := rows[publisherID]
		row.Key = publisherID
		row.Revenue += sourceRow.Revenue
		row.Conversions += sourceRow.Conversions
		row.Sent += sourceRow.Sent
		row.Shown += sourceRow.Shown
		row.Clicks += sourceRow.Clicks
		row.Closed += sourceRow.Closed
		rows[publisherID] = row
	}
	result := make([]MetricRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	return result, nil
}

func (s *Service) costRows(
	ctx context.Context,
	groupBy string,
	dateRange DateRange,
	metricRows []MetricRow,
) ([]CostRow, error) {
	if groupBy == GroupDate {
		return s.costs.CostsByGroup(ctx, groupBy, dateRange)
	}
	if groupBy == GroupPublisher {
		return s.costs.CostsByGroup(ctx, groupBy, dateRange)
	}
	allocations, err := s.costs.CostAllocations(ctx, dateRange)
	if err != nil {
		return nil, err
	}
	sourceSpend, err := s.allocatedSourceSpend(ctx, allocations, dateRange)
	if err != nil {
		return nil, err
	}
	if groupBy == GroupSource {
		return costRowsFromSpend(sourceSpend), nil
	}
	return s.allocateSpendToMetricRows(ctx, groupBy, sourceSpend, metricRows)
}

func (s *Service) allocatedSourceSpend(
	ctx context.Context,
	allocations []CostAllocationRow,
	dateRange DateRange,
) (map[string]float64, error) {
	sourceSpend := map[string]float64{}
	publisherSpend := map[string]float64{}
	for _, allocation := range allocations {
		if allocation.SourceID != "" {
			sourceSpend[allocation.SourceID] += allocation.Spend
			continue
		}
		if allocation.PublisherID != "" {
			publisherSpend[allocation.PublisherID] += allocation.Spend
		}
	}
	if len(publisherSpend) == 0 {
		return sourceSpend, nil
	}
	sourceMetrics, err := s.metrics.MetricsByGroup(ctx, GroupSource, dateRange)
	if err != nil {
		return nil, err
	}
	sourceIDs := make([]string, 0, len(sourceMetrics))
	for _, metric := range sourceMetrics {
		sourceIDs = append(sourceIDs, metric.Key)
	}
	sourcePublishers, err := s.costs.PublisherIDsBySource(ctx, sourceIDs)
	if err != nil {
		return nil, err
	}
	weightsByPublisher := map[string]float64{}
	for _, metric := range sourceMetrics {
		publisherID := sourcePublishers[metric.Key]
		if publisherSpend[publisherID] == 0 {
			continue
		}
		weightsByPublisher[publisherID] += allocationWeight(metric)
	}
	for _, metric := range sourceMetrics {
		publisherID := sourcePublishers[metric.Key]
		spend := publisherSpend[publisherID]
		totalWeight := weightsByPublisher[publisherID]
		if spend == 0 || totalWeight == 0 {
			continue
		}
		sourceSpend[metric.Key] += spend * allocationWeight(metric) / totalWeight
	}
	return sourceSpend, nil
}

func (s *Service) allocateSpendToMetricRows(
	ctx context.Context,
	groupBy string,
	sourceSpend map[string]float64,
	metricRows []MetricRow,
) ([]CostRow, error) {
	rowSources, err := s.sourceIDsForRows(ctx, groupBy, metricRows)
	if err != nil {
		return nil, err
	}
	weightsBySource := map[string]float64{}
	for _, row := range metricRows {
		sourceID := rowSources[row.Key]
		if sourceSpend[sourceID] == 0 {
			continue
		}
		weightsBySource[sourceID] += allocationWeight(row)
	}
	result := []CostRow{}
	for _, row := range metricRows {
		sourceID := rowSources[row.Key]
		spend := sourceSpend[sourceID]
		totalWeight := weightsBySource[sourceID]
		if sourceID == "" || spend == 0 || totalWeight == 0 {
			continue
		}
		result = append(result, CostRow{
			Key:   row.Key,
			Spend: spend * allocationWeight(row) / totalWeight,
		})
	}
	return result, nil
}

func (s *Service) sourceIDsForRows(
	ctx context.Context,
	groupBy string,
	rows []MetricRow,
) (map[string]string, error) {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.Key)
	}
	switch groupBy {
	case GroupCampaign:
		return s.costs.SourceIDsByCampaign(ctx, keys)
	case GroupCreative:
		return s.costs.SourceIDsByCreative(ctx, keys)
	default:
		return map[string]string{}, nil
	}
}

func allocationWeight(row MetricRow) float64 {
	switch {
	case row.Shown > 0:
		return float64(row.Shown)
	case row.Clicks > 0:
		return float64(row.Clicks)
	case row.Conversions > 0:
		return float64(row.Conversions)
	default:
		return 0
	}
}

func costRowsFromSpend(spendByKey map[string]float64) []CostRow {
	rows := make([]CostRow, 0, len(spendByKey))
	for key, spend := range spendByKey {
		rows = append(rows, CostRow{Key: key, Spend: spend})
	}
	return rows
}

func metricKeys(rows []MetricRow) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.Key)
	}
	return keys
}

func costKeys(rows []CostRow) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.Key)
	}
	return keys
}

func filterMetricRows(rows []MetricRow, keep func(MetricRow) bool) []MetricRow {
	result := make([]MetricRow, 0, len(rows))
	for _, row := range rows {
		if keep(row) {
			result = append(result, row)
		}
	}
	return result
}

func filterCostRows(rows []CostRow, keep func(CostRow) bool) []CostRow {
	result := make([]CostRow, 0, len(rows))
	for _, row := range rows {
		if keep(row) {
			result = append(result, row)
		}
	}
	return result
}

func normalizeDateRange(dateRange DateRange) DateRange {
	if dateRange.To.IsZero() {
		dateRange.To = time.Now().UTC()
	}
	if dateRange.From.IsZero() {
		dateRange.From = dateRange.To.AddDate(0, 0, -7)
	}
	return dateRange
}

func normalizeGroup(groupBy string) (string, error) {
	groupBy = strings.TrimSpace(groupBy)
	if groupBy == "" {
		groupBy = GroupSource
	}
	switch groupBy {
	case GroupDate, GroupPublisher, GroupSource, GroupCampaign, GroupCreative:
		return groupBy, nil
	default:
		return "", errors.Join(ErrInvalidInput, errors.New("invalid group_by"))
	}
}

func normalizeCostEntry(input CreateCostEntryInput) (CreateCostEntryInput, error) {
	input.Date = strings.TrimSpace(input.Date)
	input.PublisherID = strings.TrimSpace(input.PublisherID)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.CampaignID = ""
	input.CreativeID = ""
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Note = strings.TrimSpace(input.Note)
	if input.Date == "" {
		input.Date = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", input.Date); err != nil {
		return CreateCostEntryInput{}, errors.Join(ErrInvalidInput, errors.New("invalid date"))
	}
	if input.Amount <= 0 {
		return CreateCostEntryInput{}, errors.Join(ErrInvalidInput, errors.New("amount must be greater than zero"))
	}
	if input.Currency == "" {
		input.Currency = "USD"
	}
	return input, nil
}

func parseCostCSV(reader io.Reader) ([]CreateCostEntryInput, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, errors.Join(ErrInvalidInput, fmt.Errorf("read csv: %w", err))
	}
	if len(records) < 2 {
		return nil, errors.Join(ErrInvalidInput, errors.New("csv must include header and rows"))
	}
	header := csvHeader(records[0])
	inputs := make([]CreateCostEntryInput, 0, len(records)-1)
	for index, record := range records[1:] {
		input, err := costInputFromCSVRecord(header, record)
		if err != nil {
			return nil, errors.Join(
				ErrInvalidInput,
				fmt.Errorf("row %d: %w", index+2, err),
			)
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func csvHeader(record []string) map[string]int {
	header := map[string]int{}
	for index, column := range record {
		header[strings.TrimSpace(column)] = index
	}
	return header
}

func costInputFromCSVRecord(
	header map[string]int,
	record []string,
) (CreateCostEntryInput, error) {
	amount, err := strconv.ParseFloat(csvValue(header, record, "amount"), 64)
	if err != nil {
		return CreateCostEntryInput{}, fmt.Errorf("invalid amount: %w", err)
	}
	input, err := normalizeCostEntry(CreateCostEntryInput{
		Date:        csvValue(header, record, "date"),
		PublisherID: csvValue(header, record, "publisher_id"),
		SourceID:    csvValue(header, record, "source_id"),
		CampaignID:  csvValue(header, record, "campaign_id"),
		CreativeID:  csvValue(header, record, "creative_id"),
		Amount:      amount,
		Currency:    csvValue(header, record, "currency"),
		Note:        csvValue(header, record, "note"),
	})
	if err != nil {
		return CreateCostEntryInput{}, err
	}
	return input, nil
}

func csvValue(header map[string]int, record []string, column string) string {
	index, ok := header[column]
	if !ok || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func emptyMetric(row MetricRow) bool {
	return row.Revenue == 0 &&
		row.Conversions == 0 &&
		row.Sent == 0 &&
		row.Shown == 0 &&
		row.Clicks == 0 &&
		row.Closed == 0
}

func sortPerformanceRows(rows []PerformanceRow, sortBy string) {
	sortBy = strings.TrimSpace(sortBy)
	if sortBy == "" {
		sortBy = "profit"
	}
	if sortBy == "date" {
		sort.SliceStable(rows, func(i int, j int) bool {
			return rows[i].Key < rows[j].Key
		})
		return
	}
	sort.SliceStable(rows, func(i int, j int) bool {
		left := sortValue(rows[i], sortBy)
		right := sortValue(rows[j], sortBy)
		if left == right {
			return rows[i].Key < rows[j].Key
		}
		return left > right
	})
}

func sortValue(row PerformanceRow, sortBy string) float64 {
	switch sortBy {
	case "spend":
		return row.Spend
	case "revenue":
		return row.Revenue
	case "roi":
		return row.ROI
	case "conversions":
		return float64(row.Conversions)
	case "sent":
		return float64(row.Sent)
	case "shown":
		return float64(row.Shown)
	case "clicks":
		return float64(row.Clicks)
	default:
		return row.Profit
	}
}

func calculate(row PerformanceRow) PerformanceRow {
	row.Profit = row.Revenue - row.Spend
	row.ROI = ratio(row.Profit, row.Spend) * 100
	row.CTR = ratio(float64(row.Clicks), float64(row.Shown)) * 100
	row.CR = ratio(float64(row.Conversions), float64(row.Clicks)) * 100
	return row
}

func ratio(numerator float64, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}
