package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"ipehr/stat/pkg/config"
	"ipehr/stat/pkg/infrastructure"
)

// @title        IPEHR Stat API
// @version      0.1
// @description  IPEHR Stat is an open API service for providing public statistics from the IPEHR system.

// @contact.name   API Support
// @contact.url    https://bsn.si/blockchain
// @contact.email  support@bsn.si

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      stat.ipehr.org
// host      localhost:8080
// @BasePath  /

type API struct {
	Stat *StatHandler
}

func New(cfg *config.Config, infra *infrastructure.Infra) *API {
	return &API{
		Stat: NewStatHandler(infra.DB),
	}
}

func (a *API) Build() *gin.Engine {
	return a.setupRouter(
		a.buildStatAPI(),
	)
}

type handlerBuilder func(r *gin.RouterGroup)

func (a *API) setupRouter(apiHandlers ...handlerBuilder) *gin.Engine {
	r := gin.New()

	r.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(404)
	})

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
