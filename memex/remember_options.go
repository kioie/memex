package memex

// RememberOption configures Remember (mem0-style metadata and scoping).
type RememberOption func(*rememberConfig)

type rememberConfig struct {
	UserID   string
	Metadata map[string]any
}

func applyRememberOptions(opts []RememberOption) rememberConfig {
	cfg := rememberConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

// WithUserID scopes the memory to a user_id (defaults to MEMEX_USER_ID).
func WithUserID(userID string) RememberOption {
	return func(c *rememberConfig) {
		c.UserID = userID
	}
}

// WithMetadata attaches arbitrary JSON metadata (mem0 metadata bag).
func WithMetadata(metadata map[string]any) RememberOption {
	return func(c *rememberConfig) {
		c.Metadata = metadata
	}
}
