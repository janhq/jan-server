package interfaces

import (
	"github.com/google/wire"
	"menlo.ai/menlo-platform/internal/interfaces/crontab"
	"menlo.ai/menlo-platform/internal/interfaces/eventconsumers"
	"menlo.ai/menlo-platform/internal/interfaces/eventconsumers/consumers"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver"
)

var InterfacesProvider = wire.NewSet(
	crontab.NewCrontab,
	eventconsumers.NewEventConsumers,
	httpserver.NewHttpServer,
	consumers.ConsumerProvider,
)
