package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	user User
}

func (r *fakeRepo) ListUsers(_ context.Context) ([]User, error) {
	return []User{r.user}, nil
}

func (r *fakeRepo) GetUser(_ context.Context, id string) (User, error) {
	if r.user.ID == id {
		return r.user, nil
	}
	return User{}, ErrNotFound
}

func (r *fakeRepo) GetUserByEmail(_ context.Context, email string) (User, error) {
	if r.user.Email == email {
		return r.user, nil
	}
	return User{}, ErrNotFound
}

func (r *fakeRepo) SaveUser(_ context.Context, user User) (User, error) {
	r.user = user
	return user, nil
}

func TestRequestOTPRejectsActiveOTPOutsideDevReturnMode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC)
	repo := &fakeRepo{user: User{
		ID:             "user-id",
		Email:          "admin@example.com",
		OTPHash:        "active",
		OTPExpiresAt:   now.Add(time.Minute),
		OTPRequestedAt: now.Add(-time.Second),
	}}
	service := NewServiceWithDelivery(repo, nil, Config{
		AdminEmail:   "admin@example.com",
		JWTSecret:    "secret",
		DevReturnOTP: false,
	})
	service.now = func() time.Time { return now }

	_, err := service.RequestOTP(context.Background(), "admin@example.com")
	if !errors.Is(err, ErrOTPActive) {
		t.Fatalf("expected ErrOTPActive, got %v", err)
	}
}

func TestRequestOTPReissuesActiveOTPInDevReturnMode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC)
	repo := &fakeRepo{user: User{
		ID:             "user-id",
		Email:          "admin@example.com",
		OTPHash:        "active",
		OTPExpiresAt:   now.Add(time.Minute),
		OTPRequestedAt: now.Add(-time.Second),
	}}
	service := NewServiceWithDelivery(repo, nil, Config{
		AdminEmail:   "admin@example.com",
		JWTSecret:    "secret",
		DevReturnOTP: true,
	})
	service.now = func() time.Time { return now }

	challenge, err := service.RequestOTP(context.Background(), "admin@example.com")
	if err != nil {
		t.Fatalf("request otp: %v", err)
	}
	if challenge.OTP == "" {
		t.Fatal("expected dev otp in response")
	}
	if repo.user.OTPHash == "active" {
		t.Fatal("expected active otp to be replaced")
	}
}
