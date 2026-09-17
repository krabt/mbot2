package data

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestEnsureInitialRoot(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	password, created, err := db.EnsureInitialRoot()
	if err != nil || !created || password == "" {
		t.Fatalf("created=%v err=%v", created, err)
	}
	root, err := db.Repo.Users.FindByUsernameWithRoles(context.Background(), "root")
	if err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(root.PasswordHash), []byte(password)) != nil {
		t.Fatal("password mismatch")
	}
	if len(root.Roles) != 1 || len(root.Roles[0].ACLRules) != 2 {
		t.Fatalf("unexpected ACL: %+v", root.Roles)
	}
	if root.CreatedAt.IsZero() || root.UpdatedAt.IsZero() {
		t.Fatal("user timestamps were not populated")
	}
	userRole, err := db.Repo.UserRoles.First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if userRole.CreatedAt.IsZero() || userRole.UpdatedAt.IsZero() {
		t.Fatal("user role timestamps were not populated")
	}
	if password, created, err = db.EnsureInitialRoot(); err != nil || created || password != "" {
		t.Fatal("initialized twice")
	}
}
