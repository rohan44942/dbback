package googleauth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/idtoken"
)

var (
	ErrMissingClientID = errors.New("GOOGLE_CLIENT_ID is not configured")
	ErrInvalidToken    = errors.New("invalid google id token")
	ErrEmailUnverified = errors.New("google email is not verified")
)

type Profile struct {
	Subject string
	Email   string
	Name    string
	Picture string
}

// VerifyIDToken validates a Google Sign-In ID token against the configured Web client ID.
func VerifyIDToken(ctx context.Context, rawToken, clientID string) (Profile, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return Profile{}, ErrMissingClientID
	}
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Profile{}, ErrInvalidToken
	}

	payload, err := idtoken.Validate(ctx, rawToken, clientID)
	if err != nil {
		return Profile{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	email, _ := payload.Claims["email"].(string)
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return Profile{}, ErrInvalidToken
	}

	if verified, ok := payload.Claims["email_verified"].(bool); ok && !verified {
		return Profile{}, ErrEmailUnverified
	}
	// Some tokens encode email_verified as a string.
	if verified, ok := payload.Claims["email_verified"].(string); ok && !strings.EqualFold(verified, "true") {
		return Profile{}, ErrEmailUnverified
	}

	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	return Profile{
		Subject: payload.Subject,
		Email:   email,
		Name:    strings.TrimSpace(name),
		Picture: picture,
	}, nil
}
