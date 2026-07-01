package auth

import (
	"context"
	"errors"
	"strings"

	"netlog/internal/models"
	"netlog/internal/store"
	"netlog/internal/validate"
)

var (
	// ErrRegistrationClosed is returned when self-registration is attempted but
	// an account already exists (only the very first account may self-register).
	ErrRegistrationClosed = errors.New("registration is closed")
	// ErrCallsignTaken is returned when a callsign already exists.
	ErrCallsignTaken = errors.New("callsign already registered")
	// ErrEmailTaken is returned when an email already belongs to another account.
	// Email uniqueness matters because OIDC linking matches accounts by email.
	ErrEmailTaken = errors.New("email already in use")
	// ErrInvalidCredentials is returned for any failed login (no user
	// enumeration).
	ErrInvalidCredentials = errors.New("invalid callsign or password")
	// ErrLastAdmin is returned when an edit would leave the system with no admin
	// (demoting the only remaining admin), which would lock everyone out of
	// account and net administration.
	ErrLastAdmin = errors.New("cannot demote the last remaining admin")
)

// AccountInput is the data collected when creating an account.
type AccountInput struct {
	Callsign   string `json:"callsign" validate:"required"`
	FirstName  string `json:"firstName" validate:"required,max=80"`
	LastName   string `json:"lastName" validate:"required,max=80"`
	Email      string `json:"email" validate:"required,email,max=254"`
	Password   string `json:"password" validate:"required,min=8,max=200"`
	Timezone   string `json:"timezone" validate:"max=64"`
	TimeFormat string `json:"timeFormat" validate:"omitempty,oneof=24h 12h"`
}

// Service implements account registration, creation, and authentication.
type Service struct {
	store *store.Store
}

// NewService returns an auth Service.
func NewService(s *store.Store) *Service {
	return &Service{store: s}
}

// NeedsFirstAdmin reports whether the system has no accounts yet (so the forced
// first-admin registration should be shown).
func (s *Service) NeedsFirstAdmin(ctx context.Context) (bool, error) {
	n, err := s.store.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

// RegisterFirstAdmin creates the first account, which is always an admin. The
// "no accounts exist" check and the insert are atomic, so concurrent requests
// can't both create an admin; it fails with ErrRegistrationClosed otherwise.
func (s *Service) RegisterFirstAdmin(ctx context.Context, in AccountInput) (models.User, error) {
	u, err := s.prepareAccount(in, models.RoleAdmin)
	if err != nil {
		return models.User{}, err
	}
	ok, err := s.store.CreateFirstAdmin(ctx, u)
	if err != nil {
		return models.User{}, err
	}
	if !ok {
		return models.User{}, ErrRegistrationClosed
	}
	return u, nil
}

// CreateUser creates a non-admin account. Authorization (admin-only) is enforced
// by the caller/middleware.
func (s *Service) CreateUser(ctx context.Context, in AccountInput) (models.User, error) {
	u, err := s.prepareAccount(in, models.RoleUser)
	if err != nil {
		return models.User{}, err
	}
	if _, err := s.store.GetUserByCallsign(ctx, u.Callsign); err == nil {
		return models.User{}, ErrCallsignTaken
	} else if !errors.Is(err, store.ErrNotFound) {
		return models.User{}, err
	}
	if _, err := s.store.GetUserByEmail(ctx, u.Email); err == nil {
		return models.User{}, ErrEmailTaken
	} else if !errors.Is(err, store.ErrNotFound) {
		return models.User{}, err
	}
	if err := s.store.CreateUser(ctx, u); err != nil {
		return models.User{}, err
	}
	return u, nil
}

// prepareAccount validates and normalizes input and returns a populated User
// (with a hashed password) ready for insertion.
func (s *Service) prepareAccount(in AccountInput, role string) (models.User, error) {
	in.Callsign = validate.NormalizeCallsign(in.Callsign)
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.TrimSpace(in.Email)

	if err := validate.Struct(&in); err != nil {
		return models.User{}, err
	}
	if !validate.ValidCallsign(in.Callsign) {
		return models.User{}, errors.New("invalid callsign")
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return models.User{}, err
	}

	timeFormat := in.TimeFormat
	if timeFormat == "" {
		timeFormat = "24h"
	}

	now := models.Now()
	return models.User{
		ID:           NewID(),
		Callsign:     in.Callsign,
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		Email:        in.Email,
		PasswordHash: hash,
		Role:         role,
		Timezone:     strings.TrimSpace(in.Timezone),
		TimeFormat:   timeFormat,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// ProfileInput is the editable profile data.
type ProfileInput struct {
	Callsign   string `json:"callsign" validate:"required"`
	FirstName  string `json:"firstName" validate:"required,max=80"`
	LastName   string `json:"lastName" validate:"required,max=80"`
	Email      string `json:"email" validate:"required,email,max=254"`
	Timezone   string `json:"timezone" validate:"max=64"`
	TimeFormat string `json:"timeFormat" validate:"omitempty,oneof=24h 12h"`
}

// UpdateProfile updates the user's editable fields, enforcing callsign validity
// and uniqueness.
func (s *Service) UpdateProfile(ctx context.Context, userID string, in ProfileInput) (models.User, error) {
	in.Callsign = validate.NormalizeCallsign(in.Callsign)
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.TrimSpace(in.Email)
	if err := validate.Struct(&in); err != nil {
		return models.User{}, err
	}
	if !validate.ValidCallsign(in.Callsign) {
		return models.User{}, errors.New("invalid callsign")
	}

	u, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return models.User{}, err
	}

	// If the callsign changed, ensure it isn't taken by another account.
	if in.Callsign != u.Callsign {
		if other, err := s.store.GetUserByCallsign(ctx, in.Callsign); err == nil && other.ID != userID {
			return models.User{}, ErrCallsignTaken
		} else if err != nil && !errors.Is(err, store.ErrNotFound) {
			return models.User{}, err
		}
	}
	// Likewise for the email, which OIDC linking keys on.
	if !strings.EqualFold(in.Email, u.Email) {
		if other, err := s.store.GetUserByEmail(ctx, in.Email); err == nil && other.ID != userID {
			return models.User{}, ErrEmailTaken
		} else if err != nil && !errors.Is(err, store.ErrNotFound) {
			return models.User{}, err
		}
	}

	u.Callsign = in.Callsign
	u.FirstName = in.FirstName
	u.LastName = in.LastName
	u.Email = in.Email
	u.Timezone = strings.TrimSpace(in.Timezone)
	if in.TimeFormat != "" {
		u.TimeFormat = in.TimeFormat
	}
	u.UpdatedAt = models.Now()
	if err := s.store.UpdateProfile(ctx, u); err != nil {
		return models.User{}, err
	}
	return u, nil
}

// AdminUserInput is the data an admin may change on another account: identity
// fields plus role. Display preferences and password are not touched here.
type AdminUserInput struct {
	Callsign  string `json:"callsign" validate:"required"`
	FirstName string `json:"firstName" validate:"required,max=80"`
	LastName  string `json:"lastName" validate:"required,max=80"`
	Email     string `json:"email" validate:"required,email,max=254"`
	Role      string `json:"role" validate:"required,oneof=admin user"`
}

// AdminUpdateUser updates targetID's identity + role on behalf of an admin. It
// enforces callsign validity/uniqueness, email uniqueness, and refuses to demote
// the final admin (ErrLastAdmin). Authorization (admin-only) is the caller's job.
func (s *Service) AdminUpdateUser(ctx context.Context, targetID string, in AdminUserInput) (models.User, error) {
	in.Callsign = validate.NormalizeCallsign(in.Callsign)
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.TrimSpace(in.Email)
	if err := validate.Struct(&in); err != nil {
		return models.User{}, err
	}
	if !validate.ValidCallsign(in.Callsign) {
		return models.User{}, errors.New("invalid callsign")
	}

	u, err := s.store.GetUserByID(ctx, targetID)
	if err != nil {
		return models.User{}, err
	}

	// Guard against locking everyone out: don't demote the last admin.
	if u.IsAdmin() && in.Role != models.RoleAdmin {
		admins, err := s.store.CountAdmins(ctx)
		if err != nil {
			return models.User{}, err
		}
		if admins <= 1 {
			return models.User{}, ErrLastAdmin
		}
	}

	if in.Callsign != u.Callsign {
		if other, err := s.store.GetUserByCallsign(ctx, in.Callsign); err == nil && other.ID != targetID {
			return models.User{}, ErrCallsignTaken
		} else if err != nil && !errors.Is(err, store.ErrNotFound) {
			return models.User{}, err
		}
	}
	if !strings.EqualFold(in.Email, u.Email) {
		if other, err := s.store.GetUserByEmail(ctx, in.Email); err == nil && other.ID != targetID {
			return models.User{}, ErrEmailTaken
		} else if err != nil && !errors.Is(err, store.ErrNotFound) {
			return models.User{}, err
		}
	}

	u.Callsign = in.Callsign
	u.FirstName = in.FirstName
	u.LastName = in.LastName
	u.Email = in.Email
	u.Role = in.Role
	u.UpdatedAt = models.Now()
	if err := s.store.UpdateUserAdmin(ctx, u); err != nil {
		return models.User{}, err
	}
	return u, nil
}

// ChangePassword verifies the current password and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, userID, current, next string) error {
	if len(next) < 8 {
		return errors.New("new password must be at least 8 characters")
	}
	u, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	ok, err := VerifyPassword(current, u.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}
	hash, err := HashPassword(next)
	if err != nil {
		return err
	}
	return s.store.SetPassword(ctx, userID, hash, models.Now())
}

// dummyHash equalizes login timing: when a callsign doesn't exist we still run a
// full argon2id verification against this hash so the response time doesn't leak
// whether the account exists.
var dummyHash, _ = HashPassword("netlog-timing-equalizer")

// Authenticate verifies a callsign/password pair, returning the user on success
// or ErrInvalidCredentials otherwise (without revealing which part failed).
func (s *Service) Authenticate(ctx context.Context, callsign, password string) (models.User, error) {
	callsign = validate.NormalizeCallsign(callsign)
	u, err := s.store.GetUserByCallsign(ctx, callsign)
	if errors.Is(err, store.ErrNotFound) {
		// Run a throwaway verification to keep timing constant, then fail.
		_, _ = VerifyPassword(password, dummyHash)
		return models.User{}, ErrInvalidCredentials
	}
	if err != nil {
		return models.User{}, err
	}
	ok, err := VerifyPassword(password, u.PasswordHash)
	if err != nil {
		return models.User{}, err
	}
	if !ok {
		return models.User{}, ErrInvalidCredentials
	}
	return u, nil
}
