package reports

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestServiceCreateCostEntryValidatesInput(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeCostStore{}, &fakeMetricStore{})

	_, err := service.CreateCostEntry(
		context.Background(),
		CreateCostEntryInput{Date: "bad", Amount: 1},
	)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid date error, got %v", err)
	}

	_, err = service.CreateCostEntry(
		context.Background(),
		CreateCostEntryInput{Date: "2026-05-21", Amount: 0},
	)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid amount error, got %v", err)
	}
}

func TestServicePerformanceMergesSpendRevenueAndEvents(t *testing.T) {
	t.Parallel()

	service := NewService(
		&fakeCostStore{
			costs: []CostRow{{Key: "source-1", Spend: 40}},
		},
		&fakeMetricStore{
			metrics: []MetricRow{
				{
					Key:         "source-1",
					Revenue:     100,
					Conversions: 5,
					Sent:        1200,
					Shown:       1000,
					Clicks:      50,
					Closed:      10,
				},
			},
		},
	)

	rows, err := service.Performance(
		context.Background(),
		PerformanceInput{GroupBy: GroupSource},
	)
	if err != nil {
		t.Fatalf("performance failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row.Spend != 40 || row.Revenue != 100 || row.Profit != 60 {
		t.Fatalf("unexpected money metrics: %+v", row)
	}
	if row.ROI != 150 {
		t.Fatalf("unexpected roi: %v", row.ROI)
	}
	if row.Sent != 1200 {
		t.Fatalf("unexpected sent: %v", row.Sent)
	}
	if row.CTR != 5 {
		t.Fatalf("unexpected ctr: %v", row.CTR)
	}
	if row.CR != 10 {
		t.Fatalf("unexpected cr: %v", row.CR)
	}
}

func TestServicePerformanceAggregatesPublisherRows(t *testing.T) {
	t.Parallel()

	service := NewService(
		&fakeCostStore{
			costs: []CostRow{{Key: "publisher-1", Spend: 10}},
			sourcePublishers: map[string]string{
				"source-1": "publisher-1",
				"source-2": "publisher-1",
			},
			names: map[string]string{"publisher-1": "Publisher One"},
		},
		&fakeMetricStore{
			metrics: []MetricRow{
				{Key: "source-1", Revenue: 15, Conversions: 1, Sent: 12, Shown: 10, Clicks: 2},
				{Key: "source-2", Revenue: 20, Conversions: 2, Sent: 24, Shown: 20, Clicks: 4},
			},
		},
	)

	rows, err := service.Performance(
		context.Background(),
		PerformanceInput{GroupBy: GroupPublisher},
	)
	if err != nil {
		t.Fatalf("performance failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row.Key != "publisher-1" || row.Name != "Publisher One" {
		t.Fatalf("unexpected publisher row identity: %+v", row)
	}
	if row.Revenue != 35 || row.Spend != 10 || row.Profit != 25 {
		t.Fatalf("unexpected publisher metrics: %+v", row)
	}
	if row.Sent != 36 {
		t.Fatalf("unexpected publisher sent: %+v", row)
	}
}

func TestServicePerformanceAllocatesSourceSpendToCampaigns(t *testing.T) {
	t.Parallel()

	service := NewService(
		&fakeCostStore{
			allocations: []CostAllocationRow{{SourceID: "source-1", Spend: 90}},
			campaignSources: map[string]string{
				"campaign-1": "source-1",
				"campaign-2": "source-1",
			},
		},
		&fakeMetricStore{
			metrics: []MetricRow{
				{Key: "campaign-1", Revenue: 80, Shown: 100},
				{Key: "campaign-2", Revenue: 40, Shown: 200},
			},
		},
	)

	rows, err := service.Performance(
		context.Background(),
		PerformanceInput{GroupBy: GroupCampaign},
	)
	if err != nil {
		t.Fatalf("performance failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	spend := spendByKey(rows)
	if spend["campaign-1"] != 30 || spend["campaign-2"] != 60 {
		t.Fatalf("unexpected allocated spend: %+v", spend)
	}
}

func TestServicePerformanceAllocatesPublisherSpendToSources(t *testing.T) {
	t.Parallel()

	service := NewService(
		&fakeCostStore{
			allocations: []CostAllocationRow{{PublisherID: "publisher-1", Spend: 100}},
			sourcePublishers: map[string]string{
				"source-1": "publisher-1",
				"source-2": "publisher-1",
			},
		},
		&fakeMetricStore{
			metrics: []MetricRow{
				{Key: "source-1", Revenue: 80, Shown: 25},
				{Key: "source-2", Revenue: 40, Shown: 75},
			},
		},
	)

	rows, err := service.Performance(
		context.Background(),
		PerformanceInput{GroupBy: GroupSource},
	)
	if err != nil {
		t.Fatalf("performance failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	spend := spendByKey(rows)
	if spend["source-1"] != 25 || spend["source-2"] != 75 {
		t.Fatalf("unexpected allocated publisher spend: %+v", spend)
	}
}

func TestServiceImportCostEntriesParsesCSV(t *testing.T) {
	t.Parallel()

	store := &fakeCostStore{}
	service := NewService(store, &fakeMetricStore{})

	result, err := service.ImportCostEntries(
		context.Background(),
		strings.NewReader(
			"date,source_id,amount,currency,note\n"+
				"2026-05-21,source-1,12.5,usd,first row\n",
		),
	)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted row, got %d", result.Inserted)
	}
	if len(store.createdInputs) != 1 {
		t.Fatalf("expected 1 created input, got %d", len(store.createdInputs))
	}
	input := store.createdInputs[0]
	if input.Currency != "USD" || input.Amount != 12.5 || input.SourceID != "source-1" {
		t.Fatalf("unexpected imported input: %+v", input)
	}
}

func TestServiceDashboardTotalsSourceRows(t *testing.T) {
	t.Parallel()

	service := NewService(
		&fakeCostStore{
			costs: []CostRow{
				{Key: "source-1", Spend: 25},
				{Key: "source-2", Spend: 5},
			},
		},
		&fakeMetricStore{
			metrics: []MetricRow{
				{Key: "source-1", Revenue: 50, Conversions: 2, Shown: 100, Clicks: 10},
				{Key: "source-2", Revenue: 10, Conversions: 1, Shown: 50, Clicks: 5},
			},
		},
	)

	dashboard, err := service.Dashboard(context.Background(), DateRange{})
	if err != nil {
		t.Fatalf("dashboard failed: %v", err)
	}
	if dashboard.Spend != 30 || dashboard.Revenue != 60 || dashboard.Profit != 30 {
		t.Fatalf("unexpected dashboard totals: %+v", dashboard)
	}
	if dashboard.ROI != 100 {
		t.Fatalf("unexpected dashboard roi: %v", dashboard.ROI)
	}
	if dashboard.CTR != 10 {
		t.Fatalf("unexpected dashboard ctr: %v", dashboard.CTR)
	}
}

type fakeCostStore struct {
	createdInputs     []CreateCostEntryInput
	allocations       []CostAllocationRow
	costs             []CostRow
	names             map[string]string
	sourcePublishers  map[string]string
	campaignSources   map[string]string
	creativeSources   map[string]string
	creativeCampaigns map[string]string
}

func (s *fakeCostStore) CreateCostEntry(
	_ context.Context,
	input CreateCostEntryInput,
) (CostEntry, error) {
	s.createdInputs = append(s.createdInputs, input)
	return CostEntry{
		ID:       "cost-1",
		Date:     input.Date,
		Amount:   input.Amount,
		Currency: input.Currency,
	}, nil
}

func (s *fakeCostStore) ListCostEntries(
	context.Context,
	ListCostEntriesInput,
) (CostEntryPage, error) {
	return CostEntryPage{}, nil
}

func (s *fakeCostStore) CostsByGroup(
	context.Context,
	string,
	DateRange,
) ([]CostRow, error) {
	return s.costs, nil
}

func (s *fakeCostStore) CostAllocations(
	context.Context,
	DateRange,
) ([]CostAllocationRow, error) {
	if s.allocations == nil {
		allocations := make([]CostAllocationRow, 0, len(s.costs))
		for _, cost := range s.costs {
			allocations = append(allocations, CostAllocationRow{
				SourceID: cost.Key,
				Spend:    cost.Spend,
			})
		}
		return allocations, nil
	}
	return s.allocations, nil
}

func (s *fakeCostStore) DimensionNames(
	context.Context,
	string,
	[]string,
) (map[string]string, error) {
	if s.names == nil {
		return map[string]string{}, nil
	}
	return s.names, nil
}

func (s *fakeCostStore) PublisherIDsBySource(
	context.Context,
	[]string,
) (map[string]string, error) {
	if s.sourcePublishers == nil {
		return map[string]string{}, nil
	}
	return s.sourcePublishers, nil
}

func (s *fakeCostStore) SourceIDsByCampaign(
	context.Context,
	[]string,
) (map[string]string, error) {
	if s.campaignSources == nil {
		return map[string]string{}, nil
	}
	return s.campaignSources, nil
}

func (s *fakeCostStore) SourceIDsByCreative(
	context.Context,
	[]string,
) (map[string]string, error) {
	if s.creativeSources == nil {
		return map[string]string{}, nil
	}
	return s.creativeSources, nil
}

func (s *fakeCostStore) CampaignIDsByCreative(
	context.Context,
	[]string,
) (map[string]string, error) {
	if s.creativeCampaigns == nil {
		return map[string]string{}, nil
	}
	return s.creativeCampaigns, nil
}

func spendByKey(rows []PerformanceRow) map[string]float64 {
	spend := map[string]float64{}
	for _, row := range rows {
		spend[row.Key] = row.Spend
	}
	return spend
}

type fakeMetricStore struct {
	metrics []MetricRow
}

func (s *fakeMetricStore) MetricsByGroup(
	context.Context,
	string,
	DateRange,
) ([]MetricRow, error) {
	return s.metrics, nil
}

func TestNormalizeDateRange(t *testing.T) {
	t.Parallel()

	to := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	dateRange := normalizeDateRange(DateRange{To: to})

	if !dateRange.To.Equal(to) {
		t.Fatalf("unexpected to: %v", dateRange.To)
	}
	if !dateRange.From.Equal(to.AddDate(0, 0, -7)) {
		t.Fatalf("unexpected from: %v", dateRange.From)
	}
}
