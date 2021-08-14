package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
)

const (
	EncryptionAlgorithmDefault = jwa.RS512
	IssuerDefault              = "hospital_booking"
	AudienceDefault            = "hospital_booking"
	AccessTokenType            = "access"
	RefreshTokenType           = "refresh"
	AccessTokenExpiration      = 5 * time.Minute
	RefreshTokenExpiration     = 24 * time.Hour
)

// TokenOption determines the Functional Options used to create a new Token.
type TokenOption func(token jwt.Token) error

// GetDefaultAccessTokenOptions returns the common TokenOption used to create a new Access Token plus
// the given new ones options.
func GetDefaultAccessTokenOptions(opts ...TokenOption) []TokenOption {
	return append([]TokenOption{
		WithIssuer(IssuerDefault),
		WithType(AccessTokenType),
		WithAudience([]string{AudienceDefault}),
		WithJTI(),
		WithIssuedAt(),
		WithExpiration(AccessTokenExpiration),
	}, opts...)
}

// GetDefaultRefreshTokenOptions returns the common TokenOption used to create a new Refresh Token plus
// the given new ones options.
func GetDefaultRefreshTokenOptions(opts ...TokenOption) []TokenOption {
	return append([]TokenOption{
		WithIssuer(IssuerDefault),
		WithType(RefreshTokenType),
		WithAudience([]string{AudienceDefault}),
		WithJTI(),
		WithIssuedAt(),
		WithExpiration(RefreshTokenExpiration),
	}, opts...)
}

// NewJwtToken creates a new Token using the given options.
func NewJwtToken(opts ...TokenOption) (jwt.Token, error) {
	jwtToken := jwt.New()
	for _, opt := range opts {
		if err := opt(jwtToken); err != nil {
			return nil, err
		}
	}
	return jwtToken, nil
}

// WithIssuer determines the issuer of the token.
func WithIssuer(issuer string) TokenOption {
	return func(token jwt.Token) error {
		return token.Set(jwt.IssuerKey, issuer)
	}
}

// WithSubject determines the subject of the token.
func WithSubject(subject string) TokenOption {
	return func(token jwt.Token) error {
		return token.Set(jwt.SubjectKey, subject)
	}
}

// WithType determines the token type.
func WithType(typ string) TokenOption {
	return func(token jwt.Token) error {
		return token.Set("typ", typ)
	}
}

// WithExpiration determines the token expiration time.
func WithExpiration(duration time.Duration) TokenOption {
	return func(token jwt.Token) error {
		return token.Set(jwt.ExpirationKey, time.Now().Add(duration))
	}
}

// WithJTI sets a unique UUID to the token.
func WithJTI() TokenOption {
	return func(token jwt.Token) error {
		genUUID, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		return token.Set("jti", genUUID.String())
	}
}

// WithAudience determines the token audience.
func WithAudience(audience []string) TokenOption {
	return func(token jwt.Token) error {
		return token.Set(jwt.AudienceKey, audience)
	}
}

// WithIssuedAt sets the current date to token.
func WithIssuedAt() TokenOption {
	return func(token jwt.Token) error {
		return token.Set(jwt.IssuedAtKey, time.Now())
	}
}

// WithRole sets the subject's role.
func WithRole(role Role) TokenOption {
	return func(token jwt.Token) error {
		return token.Set("role", role)
	}
}

// getThumbprint gets the thumbprint of the private key in order to generate the token headers.
func getThumbprint(privateKey rsa.PrivateKey) (string, error) {
	jwKey, err := jwk.New(privateKey)
	if err != nil {
		return "", err
	}
	thumbprint, err := jwKey.Thumbprint(crypto.SHA256)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(thumbprint), nil
}

// generateTokenHeaders generates the token headers based on the given private key.
func generateTokenHeaders(privateKey rsa.PrivateKey) (jws.Headers, error) {
	thumbprint, err := getThumbprint(privateKey)
	if err != nil {
		return nil, err
	}
	headers := jws.NewHeaders()
	err = headers.Set(jws.KeyIDKey, thumbprint)
	if err != nil {
		return nil, err
	}
	return headers, nil
}

// SignToken signs the given token using the given private key.
func SignToken(token jwt.Token, privateKey rsa.PrivateKey) (string, error) {
	headers, err := generateTokenHeaders(privateKey)
	if err != nil {
		return "", err
	}
	signedToken, err := jwt.Sign(token, EncryptionAlgorithmDefault, privateKey, jwt.WithHeaders(headers))
	if err != nil {
		return "", err
	}
	return string(signedToken), err
}

// ParseToken parses the token using the public key and returns the parsed token, otherwise an error.
func ParseToken(token string, publicKey rsa.PublicKey) (jwt.Token, error) {
	parsedToken, err := jwt.Parse([]byte(token), jwt.WithVerify(EncryptionAlgorithmDefault, publicKey))
	if err != nil {
		return nil, err
	}
	return parsedToken, nil
}

// GenerateTokens generates Tokens for the given user.
func GenerateTokens(ctx context.Context, privateKey rsa.PrivateKey, user User, opts... TokenOption) (*Tokens, error) {
	opts = append(opts, WithSubject(user.UUID.String()), WithRole(user.Role))
	accessToken, err := NewJwtToken(GetDefaultAccessTokenOptions(opts...)...)
	if err != nil {
		return nil, err
	}
	signedAccessToken, err := SignToken(accessToken, privateKey)
	if err != nil {
		return nil, err
	}
	refreshToken, err := NewJwtToken(GetDefaultRefreshTokenOptions(opts...)...)
	if err != nil {
		return nil, err
	}
	signedRefreshToken, err := SignToken(refreshToken, privateKey)
	if err != nil {
		return nil, err
	}
	return &Tokens{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}, nil
}

// MustGenerateTokens generates Tokens for the given user and if any error occurs, will panic.
func MustGenerateTokens(ctx context.Context, privateKey rsa.PrivateKey, user User, opts... TokenOption) *Tokens {
	tokens, err := GenerateTokens(ctx, privateKey, user, opts...)
	if err != nil {
		panic(err)
	}
	return tokens
}