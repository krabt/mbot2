package data

import (
	"fmt"

	"mkrab/internal/config"
	"mkrab/internal/data/model"
	"mkrab/internal/data/query"
	"mkrab/internal/data/repo"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB   *gorm.DB
	Repo *repo.Repositories
}

func Open(path string) (*Database, error) {
	db, err := openGORM(path)
	if err != nil {
		return nil, err
	}
	return NewDatabase(db, repo.New(NewQuery(db))), nil
}

func openGORM(path string) (*gorm.DB, error) {
	dsn := path + "?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Error)})
	if err != nil {
		return nil, err
	}
	// 使用显式关联模型，使 mqtt_user_role 也拥有时间戳和软删除字段。
	if err := db.SetupJoinTable(&model.User{}, "Roles", &model.UserRole{}); err != nil {
		return nil, fmt.Errorf("setup user role join table: %w", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Role{}, &model.ACLRule{}, &model.ReceivedMessage{}, &model.Topic{}, &model.ClientConnection{}, &model.UserRole{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}

// OpenFromConfig 是 Wire provider，负责创建底层 GORM DB 及其清理函数。
func OpenFromConfig(cfg *config.Config) (*gorm.DB, func(), error) {
	db, err := openGORM(cfg.Database)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	}
	return db, cleanup, nil
}

func NewQuery(db *gorm.DB) *query.Query { return query.Use(db) }

func NewDatabase(db *gorm.DB, repositories *repo.Repositories) *Database {
	return &Database{DB: db, Repo: repositories}
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
