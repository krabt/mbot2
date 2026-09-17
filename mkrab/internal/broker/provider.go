package broker

import "github.com/google/wire"

// ProviderSet 定义 Broker 层内部组件的依赖装配关系。
var ProviderSet = wire.NewSet(
	NewAuthACLHook,
	NewClientConnectionHook,
	NewMessageStore,
	NewListeners,
	NewServer,
)
