package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT
    u.id::text,
    u.email,
    r.code,
    u.status,
    u.email_verified,
    u.approved,
    u.otp_hash,
    u.otp_expires_at,
    u.otp_requested_at,
    u.created_at,
    u.updated_at
FROM users u
JOIN roles r ON r.id = u.role_id
ORDER BY u.created_at ASC
`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users rows: %w", err)
	}
	return users, nil
}

func (r *PostgresRepository) GetUser(ctx context.Context, id string) (User, error) {
	return r.findOne(ctx, "u.id = $1", id)
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	return r.findOne(ctx, "u.email = $1", normalizeEmail(email))
}

func (r *PostgresRepository) SaveUser(ctx context.Context, user User) (User, error) {
	user.Email = normalizeEmail(user.Email)
	if user.Role == "" {
		user.Role = RoleUser
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	row := r.pool.QueryRow(
		ctx,
		`
WITH selected_role AS (
    SELECT id FROM roles WHERE code = $4
)
INSERT INTO users (
    id,
    email,
    password_hash,
    status,
    role_id,
    email_verified,
    approved,
    otp_hash,
    otp_expires_at,
    otp_requested_at,
    created_at,
    updated_at
)
SELECT
    $1::uuid,
    $2,
    '',
    $3,
    selected_role.id,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11
FROM selected_role
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    status = EXCLUDED.status,
    role_id = EXCLUDED.role_id,
    email_verified = EXCLUDED.email_verified,
    approved = EXCLUDED.approved,
    otp_hash = EXCLUDED.otp_hash,
    otp_expires_at = EXCLUDED.otp_expires_at,
    otp_requested_at = EXCLUDED.otp_requested_at,
    updated_at = EXCLUDED.updated_at
RETURNING
    id::text,
    email,
    $4::text,
    status,
    email_verified,
    approved,
    otp_hash,
    otp_expires_at,
    otp_requested_at,
    created_at,
    updated_at
`,
		user.ID,
		user.Email,
		user.Status,
		user.Role,
		user.EmailVerified,
		user.Approved,
		user.OTPHash,
		nullableTime(user.OTPExpiresAt),
		nullableTime(user.OTPRequestedAt),
		user.CreatedAt,
		user.UpdatedAt,
	)

	saved, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, errors.Join(ErrInvalidInput, errors.New("email already exists"))
		}
		return User{}, fmt.Errorf("save user: %w", err)
	}
	return saved, nil
}

func (r *PostgresRepository) findOne(ctx context.Context, condition string, arg any) (User, error) {
	row := r.pool.QueryRow(
		ctx,
		`
SELECT
    u.id::text,
    u.email,
    r.code,
    u.status,
    u.email_verified,
    u.approved,
    u.otp_hash,
    u.otp_expires_at,
    u.otp_requested_at,
    u.created_at,
    u.updated_at
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE `+condition+`
LIMIT 1
`,
		arg,
	)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	var otpExpiresAt *time.Time
	var otpRequestedAt *time.Time
	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.Approved,
		&user.OTPHash,
		&otpExpiresAt,
		&otpRequestedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return User{}, err
	}
	if otpExpiresAt != nil {
		user.OTPExpiresAt = *otpExpiresAt
	}
	if otpRequestedAt != nil {
		user.OTPRequestedAt = *otpRequestedAt
	}
	return user, nil
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
