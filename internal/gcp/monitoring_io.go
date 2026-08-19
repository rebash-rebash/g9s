package gcp

import (
	"context"
	"fmt"
	"sort"
	"time"

	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/rebash-rebash/g9s/internal/model"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	networkReceivedBytesMetric = "compute.googleapis.com/instance/network/received_bytes_count"
	networkSentBytesMetric     = "compute.googleapis.com/instance/network/sent_bytes_count"
	diskReadBytesMetric        = "compute.googleapis.com/instance/disk/read_bytes_count"
	diskWriteBytesMetric       = "compute.googleapis.com/instance/disk/write_bytes_count"
)

// GetIOStats returns 24-hour network and disk throughput for a VM.
// Values are bytes per second. Network data is aggregated across NICs and
// disk data is aggregated across attached disks.
func (s *MonitoringService) GetIOStats(ctx context.Context, instanceID string) (model.IOStats, error) {
	if instanceID == "" || instanceID == "0" {
		return model.IOStats{}, nil
	}

	in, err := s.getRateStats(ctx, instanceID, networkReceivedBytesMetric)
	if err != nil {
		return model.IOStats{}, fmt.Errorf("read network receive metrics: %w", err)
	}
	out, err := s.getRateStats(ctx, instanceID, networkSentBytesMetric)
	if err != nil {
		return model.IOStats{}, fmt.Errorf("read network send metrics: %w", err)
	}
	read, err := s.getRateStats(ctx, instanceID, diskReadBytesMetric)
	if err != nil {
		return model.IOStats{}, fmt.Errorf("read disk read metrics: %w", err)
	}
	write, err := s.getRateStats(ctx, instanceID, diskWriteBytesMetric)
	if err != nil {
		return model.IOStats{}, fmt.Errorf("read disk write metrics: %w", err)
	}

	return model.IOStats{
		NetworkInCurrent: in.current, NetworkInAverage: in.average, NetworkInP95: in.p95,
		NetworkOutCurrent: out.current, NetworkOutAverage: out.average, NetworkOutP95: out.p95,
		DiskReadCurrent: read.current, DiskReadAverage: read.average, DiskReadP95: read.p95,
		DiskWriteCurrent: write.current, DiskWriteAverage: write.average, DiskWriteP95: write.p95,
	}, nil
}

type ratePoint struct {
	value float64
	time  time.Time
}

type rateStats struct {
	current *float64
	average *float64
	p95     *float64
}

func (s *MonitoringService) getRateStats(ctx context.Context, instanceID, metricType string) (rateStats, error) {
	now := time.Now()
	start := now.Add(-24 * time.Hour)
	filter := fmt.Sprintf(
		`metric.type = %q AND resource.type = "gce_instance" AND resource.labels.instance_id = %q`,
		metricType, instanceID,
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
			AlignmentPeriod:   durationpb.New(5 * time.Minute),
			PerSeriesAligner:  monitoringpb.Aggregation_ALIGN_RATE,
			CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_SUM,
		},
	})

	var points []ratePoint
	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return rateStats{}, err
		}
		for _, point := range ts.GetPoints() {
			value := point.GetValue().GetDoubleValue()
			if value == 0 {
				value = float64(point.GetValue().GetInt64Value())
			}
			points = append(points, ratePoint{
				value: value,
				time:  point.GetInterval().GetEndTime().AsTime(),
			})
		}
	}

	if len(points) == 0 {
		return rateStats{}, nil
	}

	var sum float64
	latest := points[0]
	values := make([]float64, 0, len(points))
	for _, point := range points {
		values = append(values, point.value)
		sum += point.value
		if point.time.After(latest.time) {
			latest = point
		}
	}

	sort.Float64s(values)
	average := sum / float64(len(values))
	p95 := values[int(float64(len(values)-1)*0.95)]
	current := latest.value

	return rateStats{current: &current, average: &average, p95: &p95}, nil
}
