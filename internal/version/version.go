package version

var (
	Version = "0.0.0-dev"
	Commit  = "unknown"
)

func String() string {
	return "disk-smi " + Version + " (" + Commit + ")"
}
