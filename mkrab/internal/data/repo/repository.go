package repo

import "mkrab/internal/data/query"

// Repositories 汇总所有数据仓储；上层业务不直接依赖 GORM Gen Query。
type Repositories struct {
	Users       *UserRepository
	Roles       *RoleRepository
	ACLRules    *ACLRuleRepository
	Messages    *MessageRepository
	Topics      *TopicRepository
	Connections *ConnectionRepository
	UserRoles   *UserRoleRepository
	query       *query.Query
}

func New(q *query.Query) *Repositories {
	return &Repositories{
		Users: &UserRepository{query: q}, Roles: &RoleRepository{query: q},
		ACLRules: &ACLRuleRepository{query: q}, Messages: &MessageRepository{query: q},
		Topics:      &TopicRepository{query: q},
		Connections: &ConnectionRepository{query: q}, UserRoles: &UserRoleRepository{query: q},
		query: q,
	}
}

func (r *Repositories) Transaction(fn func(*Repositories) error) error {
	return r.query.Transaction(func(tx *query.Query) error { return fn(New(tx)) })
}
