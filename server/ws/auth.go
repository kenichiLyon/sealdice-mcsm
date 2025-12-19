package ws

import (
	"errors"
	"net/http"
)

// ValidateToken validates the authentication token.
// Currently a placeholder as per requirements.
func ValidateToken(r *http.Request) error {
	// TODO: Implement actual token validation logic if needed.
	// For now, we assume plugin is trusted as per constraints.
	return nil
}

func checkAuth(r *http.Request) error {
	if err := ValidateToken(r); err != nil {
		return errors.New("unauthorized")
	}
	return nil
}
