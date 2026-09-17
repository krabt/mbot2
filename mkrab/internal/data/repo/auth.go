package repo

import (
	"context"

	"mkrab/internal/data/model"
	"mkrab/internal/data/query"

	"gorm.io/gorm/clause"
)

type UserRepository struct{ query *query.Query }

func (r *UserRepository) FindEnabled(ctx context.Context, username string) (*model.User, error) {
	u := r.query.User
	return u.WithContext(ctx).Where(u.Username.Eq(username), u.Enabled.Is(true)).First()
}

func (r *UserRepository) FindEnabledWithRoles(ctx context.Context, username string) (*model.User, error) {
	u := r.query.User
	return u.WithContext(ctx).Preload(u.Roles.ACLRules).Where(u.Username.Eq(username), u.Enabled.Is(true)).First()
}

func (r *UserRepository) FindByUsernameWithRoles(ctx context.Context, username string) (*model.User, error) {
	u := r.query.User
	return u.WithContext(ctx).Preload(u.Roles.ACLRules).Where(u.Username.Eq(username)).First()
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return r.query.User.WithContext(ctx).Count()
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.query.User.WithContext(ctx).Create(user)
}

func (r *UserRepository) AddRole(ctx context.Context, user *model.User, role *model.Role) error {
	return r.query.User.Roles.WithContext(ctx).Model(user).Append(role)
}

type RoleRepository struct{ query *query.Query }

func (r *RoleRepository) FindByName(ctx context.Context, name string) (*model.Role, error) {
	role := r.query.Role
	return role.WithContext(ctx).Where(role.Name.Eq(name)).First()
}

func (r *RoleRepository) Create(ctx context.Context, role *model.Role) error {
	return r.query.Role.WithContext(ctx).Create(role)
}

type ACLRuleRepository struct{ query *query.Query }

func (r *ACLRuleRepository) CreateIgnoreConflicts(ctx context.Context, rules ...*model.ACLRule) error {
	return r.query.ACLRule.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(rules...)
}

func (r *ACLRuleRepository) Create(ctx context.Context, rules ...*model.ACLRule) error {
	return r.query.ACLRule.WithContext(ctx).Create(rules...)
}
