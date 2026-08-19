package model

type Project struct {
	ID          string
	Name        string
	Number      string
	DisplayName string
}

type VM struct {
	Name         string
	InstanceID   string
	Zone         string
	MachineType  string
	Status       string
	InternalIP   string
	ExternalIP   string
	CreationTime string
	BootDisk     string
	DiskCount    int
	NetworkCount int
}

type Cluster struct {
	Name      string
	Location  string
	Status    string
	NodeCount int
}

type Disk struct {
	Name     string
	Zone     string
	SizeGB   int64
	Attached bool
}

type Cost struct {
	AmountUSD float64
	Period    string
}

type Metric struct {
	Name       string
	Value      float64
	Unit       string
	ObservedAt string
}

type Utilization struct {
	Current *float64
	Average *float64
	P95     *float64
	Unit    string
}

// IOStats contains 24-hour average/current throughput for a VM.
// Network and disk values are bytes per second.
type IOStats struct {
	NetworkInCurrent  *float64
	NetworkInAverage  *float64
	NetworkInP95      *float64
	NetworkOutCurrent *float64
	NetworkOutAverage *float64
	NetworkOutP95     *float64
	DiskReadCurrent   *float64
	DiskReadAverage   *float64
	DiskReadP95       *float64
	DiskWriteCurrent  *float64
	DiskWriteAverage  *float64
	DiskWriteP95      *float64
}

type Finding struct {
	ResourceID     string
	ResourceType   string
	Severity       string
	Reason         string
	MonthlyCost    float64
	PotentialSave  float64
	Recommendation string
}
