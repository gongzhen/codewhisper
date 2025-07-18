package config

import (
	"os"
	"strconv"
	"strings"
)

// Environment variables used by CodeWhisper
const (
    EnvUserCodebaseDir      = "CODEWHISPER_USER_CODEBASE_DIR"
    EnvAdditionalExcludeDirs = "CODEWHISPER_ADDITIONAL_EXCLUDE_DIRS"
    EnvAWSProfile           = "CODEWHISPER_AWS_PROFILE"
    EnvModel                = "CODEWHISPER_MODEL"
    EnvEndpoint             = "CODEWHISPER_ENDPOINT"
    EnvMaxDepth             = "CODEWHISPER_MAX_DEPTH"
    EnvLogLevel             = "CODEWHISPER_LOG_LEVEL"
    EnvTemperature          = "CODEWHISPER_TEMPERATURE"
    EnvMaxOutputTokens      = "CODEWHISPER_MAX_OUTPUT_TOKENS"
    EnvTopK                 = "CODEWHISPER_TOP_K"
    EnvThinkingMode         = "CODEWHISPER_THINKING_MODE"
)

// GetEnv retrieves an environment variable with a default value
func GetEnv(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}

// GetEnvInt retrieves an environment variable as an integer with a default value
func GetEnvInt(key string, defaultValue int) int {
    if value, exists := os.LookupEnv(key); exists {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

// GetEnvBool retrieves an environment variable as a boolean
func GetEnvBool(key string, defaultValue bool) bool {
    if value, exists := os.LookupEnv(key); exists {
        return value == "1" || strings.ToLower(value) == "true"
    }
    return defaultValue
}

// SetEnv sets an environment variable (wrapper for os.Setenv)
func SetEnv(key, value string) error {
    return os.Setenv(key, value)
}