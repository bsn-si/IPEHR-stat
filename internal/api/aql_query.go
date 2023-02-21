package api

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/bsn-si/IPEHR-gateway/src/pkg/docs/model"
	"github.com/bsn-si/IPEHR-gateway/src/pkg/errors"
	"github.com/gin-gonic/gin"
)

type AQLQuerier interface {
	ExecQuery(ctx context.Context, query *model.QueryRequest) (*model.QueryResponse, error)
}

type aqlQueryAPI struct {
	querier AQLQuerier
}

func newAQLQueryAPI(querier AQLQuerier) *aqlQueryAPI {
	return &aqlQueryAPI{
		querier: querier,
	}
}

func (api *aqlQueryAPI) QueryHandler(c *gin.Context) {
	var q model.QueryRequest

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("Request body read error: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "request validation error"})
		return
	}

	err = q.FromBytes(data)
	if err != nil {
		log.Println("QueryRequest FromBytes error: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "request validation error"})
		return
	}

	resp, err := api.querier.ExecQuery(c.Request.Context(), &q)
	if err != nil {
		log.Printf("cannot exec query: %v", err)

		if errors.Is(err, errors.ErrTimeout) {
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "timeout exceeded"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	respData, err := resp.Bytes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "response serialization error"})
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", respData)
}
