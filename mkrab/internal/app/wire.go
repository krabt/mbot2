//go:build wireinject

package app

import (
	"mkrab/internal/broker"
	"mkrab/internal/config"
	"mkrab/internal/data"

	"github.com/google/wire"
)

func Initialize(configPath string) (*Application, func(), error) {
	wire.Build(config.Load, data.ProviderSet, broker.ProviderSet, New)
	return nil, nil, nil
}
