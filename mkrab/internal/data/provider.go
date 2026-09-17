package data

import (
	"mkrab/internal/data/repo"

	"github.com/google/wire"
)

// ProviderSet 向 Wire 暴露数据库、仓储集合及各细分仓储。
var ProviderSet = wire.NewSet(
	OpenFromConfig,
	NewQuery,
	repo.New,
	NewDatabase,
	wire.FieldsOf(
		new(*repo.Repositories),
		"Users",
		"Roles",
		"ACLRules",
		"Messages",
		"Topics",
		"Connections",
		"UserRoles",
	),
)
