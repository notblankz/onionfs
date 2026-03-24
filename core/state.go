package core

type OnionState struct {
	LowerDir   string
	UpperDir   string
	MountPoint string

	// status flags
	CoW        bool
	HideMeta   bool
	Foreground bool
}
