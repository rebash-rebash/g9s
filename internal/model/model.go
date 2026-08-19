package model

type Project struct {
	ID          string
	Name        string
	Number      string
	DisplayName string
}

type VM struct {
	Name        string
	Zone        string
	MachineType string
	Status      string
}

type Cluster struct {
	Name      string
	Location  string
	Status    string
	NodeCount int
}

type Disk struct {
	Name      string
	Zone      string
	SizeGB    int64
	Attached  bool
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

type Finding struct {
	ResourceID    string
	ResourceType  string
	Severity      string
	Reason        string
	MonthlyCost   float64
	PotentialSave float64
	Recommendation string
}
