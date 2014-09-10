package teamweek

import (
  "testing"
)

func TestNewClient(t *testing.T) {
  c := NewClient(nil)

  if c.BaseURL.String() != defaultBaseURL {
    t.Errorf("NewClient BaseURL = %v, want %v", c.BaseURL.String(), defaultBaseURL)
  }
  if c.UserAgent != userAgent {
    t.Errorf("NewClient UserAgent = %v, want %v", c.UserAgent, userAgent)
  }
}
