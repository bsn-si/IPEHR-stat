package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/bsn-si/IPEHR-stat/internal/api/middleware"
	"github.com/bsn-si/IPEHR-stat/internal/queryexecuter"
	"github.com/bsn-si/IPEHR-stat/pkg/config"
	"github.com/bsn-si/IPEHR-stat/pkg/infrastructure"
)

type API struct {
	Stat     *StatHandler
	queryAPI *aqlQueryAPI
}

func New(cfg *config.Config, infra *infrastructure.Infra) *API {
	return &API{
		Stat:     NewStatHandler(infra.DB),
		queryAPI: newAQLQueryAPI(queryexecuter.NewQueryExecuterService(infra.AqlDB)),
	}
}

func (a *API) Build() *gin.Engine {
	return a.setupRouter(
		a.buildStatAPI(),
		a.buildQueryAPI(),
	)
}

type handlerBuilder func(r *gin.RouterGroup)

func (a *API) setupRouter(apiHandlers ...handlerBuilder) *gin.Engine {
	r := gin.New()

	r.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(404)
	})

	r.Use(middleware.RequestID)

	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// your custom format
		return fmt.Sprintf("[GIN] %19s | %3d | %13v | %15s | %-7s %#v %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	statGroup := r.Group("")
	for _, b := range apiHandlers {
		b(statGroup)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

func (a *API) buildStatAPI() handlerBuilder {
	return func(r *gin.RouterGroup) {
		r.GET("", a.Stat.GetTotal)
		r.GET("/:period", a.Stat.GetStat)
	}
}

func (a *API) buildQueryAPI() handlerBuilder {
	return func(r *gin.RouterGroup) {
		r = r.Group("query")

		r.POST("/", a.queryAPI.QueryHandler)
	}
}