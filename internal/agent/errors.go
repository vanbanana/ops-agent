package agent

// ErrorCode defines typed error codes for LLM/Agent operations.
// Naming: <DOMAIN>_<CATEGORY>_<SEQ> (design.md §4.6)
type ErrorCode string

const (
	ErrLLMNetwork ErrorCode = "LLM_NETWORK_001"
	ErrLLMAuth    ErrorCode = "LLM_AUTH_001"
	ErrLLMQuota   ErrorCode = "LLM_QUOTA_001"
	ErrLLMService ErrorCode = "LLM_SERVICE_001"
	ErrLLMTimeout ErrorCode = "LLM_TIMEOUT_001"
	ErrLLMParse   ErrorCode = "LLM_PARSE_001"
)

// LLMError wraps an error with a typed error code.
type LLMError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *LLMError) Error() string {
	if e.Err != nil {
		return string(e.Code) + ": " + e.Message + ": " + e.Err.Error()
	}
	return string(e.Code) + ": " + e.Message
}

func (e *LLMError) Unwrap() error {
	return e.Err
}
