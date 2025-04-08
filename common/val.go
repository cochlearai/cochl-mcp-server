package common

import "os"

// version is set at build time using ldflags
var Version = "0.0.1"

const (
	CochlSenseProjectKeyEnv = "COCHL_SENSE_PROJECT_KEY"
)

func GetCochlSenseProjectKey() string {
	return os.Getenv(CochlSenseProjectKeyEnv)
}
