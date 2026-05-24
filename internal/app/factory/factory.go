package factory

import (
	"github.com/ivanSaichkin/wb-search-top/internal/domain/ports/usecases"
	infraRedis "github.com/ivanSaichkin/wb-search-top/internal/infrastructure/persistence/redis"
	"github.com/ivanSaichkin/wb-search-top/internal/infrastructure/service"
	"github.com/redis/go-redis/v9"
)

// содержит все инициализированные бизнес-сервисы (use cases) приложения
type Services struct {
	Search   usecases.SearchUseCase
	StopList usecases.StoplistUseCase
}

// отвечает за создание и внедрение зависимостей
type ServiceFactory struct {
	redisClient *redis.Client
}

func NewServiceFactory(redisClient *redis.Client) *ServiceFactory {
	return &ServiceFactory{
		redisClient: redisClient,
	}
}

func (f *ServiceFactory) Build() *Services {
	searchRepo := infraRedis.NewSearchRepo(f.redisClient)
	stopListRepo := infraRedis.NewStopListRepo(f.redisClient)

	searchUC := service.NewSearchService(searchRepo, stopListRepo)
	stopListUC := service.NewStopListService(stopListRepo)

	return &Services{
		Search:   searchUC,
		StopList: stopListUC,
	}
}
