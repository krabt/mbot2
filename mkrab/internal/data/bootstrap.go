package data

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"mkrab/internal/data/model"
	"mkrab/internal/data/repo"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// EnsureInitialRoot 仅在用户表为空时创建 root、admin 角色及全权限 ACL。
func (d *Database) EnsureInitialRoot() (password string, created bool, err error) {
	err = d.Repo.Transaction(func(repositories *repo.Repositories) error {
		ctx := context.Background()
		count, err := repositories.Users.Count(ctx)
		if err != nil || count > 0 {
			return err
		}
		random := make([]byte, 18)
		if _, err := rand.Read(random); err != nil {
			return fmt.Errorf("generate root password: %w", err)
		}
		password = base64.RawURLEncoding.EncodeToString(random)
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash root password: %w", err)
		}

		roleRecord, err := repositories.Roles.FindByName(ctx, "admin")
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == gorm.ErrRecordNotFound {
			roleRecord = &model.Role{Name: "admin"}
			if err := repositories.Roles.Create(ctx, roleRecord); err != nil {
				return err
			}
		}
		rules := []*model.ACLRule{
			{RoleID: roleRecord.ID, Operation: model.OperationPublish, Effect: model.EffectAllow, TopicFilter: "#"},
			{RoleID: roleRecord.ID, Operation: model.OperationSubscribe, Effect: model.EffectAllow, TopicFilter: "#"},
		}
		if err := repositories.ACLRules.CreateIgnoreConflicts(ctx, rules...); err != nil {
			return err
		}
		root := &model.User{Username: "root", PasswordHash: string(hash), Enabled: true}
		if err := repositories.Users.Create(ctx, root); err != nil {
			return err
		}
		if err := repositories.Users.AddRole(ctx, root, roleRecord); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return "", false, fmt.Errorf("initialize root user: %w", err)
	}
	return password, created, nil
}
