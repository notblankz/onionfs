package core

type OnionState struct {
	LowerDir   string
	UpperDir   string
	MountPoint string

	// status flags
	HideMeta bool
	Debug    bool
}
