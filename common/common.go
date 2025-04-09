package common

import "os"

// Version is set at build time using ldflags
var Version = "0.0.0"

const (
	CochlSenseProjectKeyEnv = "COCHL_SENSE_PROJECT_KEY"
	CochlSenseBaseURLEnv    = "COCHL_SENSE_BASE_URL"

	_defaultBaseURL = "https://api.cochl.ai"
)

func GetCochlSenseProjectKey() string {
	return os.Getenv(CochlSenseProjectKeyEnv)
}

func GetCochlSenseBaseURL() string {
	baseUrl := os.Getenv(CochlSenseBaseURLEnv)
	if baseUrl != "" {
		return baseUrl
	}
	return _defaultBaseURL
}
