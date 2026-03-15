package api

// Request represents an HTTP request definition with optional auth,
// headers, body, and variable interpolation support.
type Request struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	BodyType    string            `json:"body_type,omitempty"` // json, xml, form, text, graphql
	Auth        *AuthConfig       `json:"auth,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	Folder      string            `json:"folder,omitempty"`
}

// AuthConfig holds authentication details for a request.
type AuthConfig struct {
	Type  string `json:"type"` // bearer, basic, api_key
	Token string `json:"token,omitempty"`
	User  string `json:"user,omitempty"`
	Pass  string `json:"pass,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	In    string `json:"in,omitempty"` // header, query
}

// Response holds the result of an executed HTTP request.
type Response struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Duration   int64             `json:"duration_ms"`
	Size       int64             `json:"size_bytes"`
}

// Collection groups related endpoints with shared configuration.
type Collection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	BaseURL     string    `json:"base_url,omitempty"`
	Endpoints   []Request `json:"endpoints"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// Environment holds named variable sets for request interpolation.
type Environment struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}

// HistoryEntry records a request-response pair with a timestamp.
type HistoryEntry struct {
	Timestamp string   `json:"timestamp"`
	Request   Request  `json:"request"`
	Response  Response `json:"response"`
}

// WSConnection tracks a live WebSocket connection and its messages.
type WSConnection struct {
	ID        string
	URL       string
	Connected bool
	Messages  []WSMessage
}

// WSMessage represents a single WebSocket message with direction metadata.
type WSMessage struct {
	Direction string `json:"direction"` // sent, received
	Data      string `json:"data"`
	Timestamp string `json:"timestamp"`
}
