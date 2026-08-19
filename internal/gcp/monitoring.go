package gcp

import (
	"context"
	"fmt"
	"sort"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/rebash-rebash/g9s/internal/model"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const cpuMetricType = "compute.googleapis.com/instance/cpu/utilization"

// MonitoringService provides read-only Cloud Monitoring operations.
type MonitoringService struct {
	metrics   *monitoring.MetricClient
	projectID string
}

func NewMonitoringService(ctx context.Context, client *Client) (*MonitoringService, error) {
	metrics, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Cloud Monitoring client: %w", err)
	}
	return &MonitoringService{metrics: metrics, projectID: client.ProjectID}, nil
}

func (s *MonitoringService) Close() error { return s.metrics.Close() }

// GetCPUUtilization returns CPU utilization for the last 24 hours.
// Current is the newest observed point, Average is the mean of all aligned
// points, and P95 is the 95th percentile of those points. Values are ratios
// from 0.0 to 1.0, matching the Cloud Monitoring metric.
func (s *MonitoringService) GetCPUUtilization(ctx context.Context, instanceID string) (model.Utilization, error) {
	if instanceID == "" || instanceID == "0" {
		return model.Utilization{Unit: "ratio"}, nil
	}

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	filter := fmt.Sprintf(
		`metric.type = %q AND resource.type = "gce_instance" AND resource.labels.instance_id = %q`,
		cpuMetricType,
		instanceID,
	)

	it := s.metrics.ListTimeSeries(ctx, &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + s.projectID,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(now),
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:  durationpb.New(5 * time.Minute),
			PerSeriesAligner: monitoringpb.Aggregation_ALIGN_MEAN,
		},
	})

	var values []float64
	var latestValue float64
	var latestTime time.Time

	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return model.Utilization{}, fmt.Errorf("read CPU utilization: %w", err)
		}
		for _, point := range ts.GetPoints() {
			value := point.GetValue().GetDoubleValue()
			values = append(values, value)
			observed := point.GetInterval().GetEndTime().AsTime()
			if latestTime.IsZero() || observed.After(latestTime) {
				latestTime = observed
				latestValue = value
			}
		}
	}

	if len(values) == 0 {
		return model.Utilization{Unit: "ratio"}, nil
	}

	sort.Float64s(values)
	var sum float64
	for _, value := range values {
		sum += value
	}
	average := sum / float64(len(values))
	p95Index := int(float64(len(values)-1) * 0.95)
	p95 := values[p95Index]

	return model.Utilization{
		Current: &latestValue,
		Average: &average,
		P95:     &p95,
		Unit:    "ratio",
	}, nil
}
