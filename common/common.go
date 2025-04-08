package common

import "os"

// version is set at build time using ldflags
var Version = "0.0.1"

const (
	CochlSenseProjectKeyEnv = "COCHL_SENSE_PROJECT_KEY"
	CochlSenseBaseURLEnv    = "COCHL_SENSE_BASE_URL"
)

func GetCochlSenseProjectKey() string {
	return os.Getenv(CochlSenseProjectKeyEnv)
}

func GetCochlSenseBaseURL() string {
	baseUrl := os.Getenv(CochlSenseBaseURLEnv)
	if baseUrl != "" {
		return baseUrl
	}
	return "https://api.cochl.ai"
}
