package model

// VM represents the normalized read-only view of a Compute Engine instance.
type VM struct {
	Name        string
	Zone        string
	MachineType string
	Status      string
	InternalIP  string
	ExternalIP  string
}
