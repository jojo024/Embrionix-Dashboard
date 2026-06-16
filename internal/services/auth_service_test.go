package services

import (
	"testing"

	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/models"
)

func TestRoleRBAC(t *testing.T) {
	if !models.RoleAdmin.AtLeast(models.RoleOperator) {
		t.Error("admin should outrank operator")
	}
	if !models.RoleOperator.AtLeast(models.RoleViewer) {
		t.Error("operator should outrank viewer")
	}
	if models.RoleViewer.AtLeast(models.RoleOperator) {
		t.Error("viewer must not meet operator")
	}
	if models.Role("bogus").Valid() {
		t.Error("unknown role must be invalid")
	}
}

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "s3cret" || hash == "" {
		t.Fatal("password was not hashed")
	}
}

func TestJWTIssueAndVerify(t *testing.T) {
	svc := NewAuthService(nil, config.AuthConfig{Enabled: true, JWTSecret: "test-secret", TokenTTLHours: 1})
	token, err := svc.issue(&models.User{ID: 1, Username: "alice", Role: models.RoleOperator})
	if err != nil {
		t.Fatal(err)
	}
	username, role, err := svc.Verify(token)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if username != "alice" || role != models.RoleOperator {
		t.Fatalf("got (%q, %q), want (alice, operator)", username, role)
	}

	// A token signed with a different secret must fail.
	other := NewAuthService(nil, config.AuthConfig{JWTSecret: "different"})
	if _, _, err := other.Verify(token); err == nil {
		t.Error("expected verification to fail with wrong secret")
	}
}
