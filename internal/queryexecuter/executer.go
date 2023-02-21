package queryexecuter

import (
	"context"

	"github.com/bsn-si/IPEHR-gateway/src/pkg/aqlprocessor"
	"github.com/bsn-si/IPEHR-gateway/src/pkg/docs/model"
	"github.com/bsn-si/IPEHR-gateway/src/pkg/errors"
	"github.com/bsn-si/IPEHR-gateway/src/pkg/storage/treeindex"
	"github.com/bsn-si/IPEHR-stat/internal/aqlquerier"

	"github.com/jmoiron/sqlx"
)

type queryCallback func(columns []string, rows []any, err error)

type Query struct {
	ctx      context.Context
	query    *aqlprocessor.Query
	args     map[string]any
	callback queryCallback
}

type ExecuterService struct {
	db          *sqlx.DB
	queriesChan chan Query
	index       *treeindex.EHRIndex
}

func NewQueryExecuterService(db *sqlx.DB) *ExecuterService {
	svc := &ExecuterService{
		db:          db,
		queriesChan: make(chan Query, 100),
		index:       treeindex.DefaultEHRIndex,
	}

	go svc.Run()

	return svc
}

func (svc *ExecuterService) Run() {
	for q := range svc.queriesChan {
		if q.ctx.Err() != nil {
			continue
		}

		exec := aqlquerier.Executer{
			Query:  q.query,
			Params: q.args,
			Index:  svc.index,
		}

		rows, err := exec.Run()
		if err != nil {
			q.callback(nil, nil, errors.Wrap(err, "cannot exec.Run()"))
			continue
		}

		q.callback(rows.Columns(), rows.Rows(), nil)
	}
}

func (svc *ExecuterService) Close() {
	close(svc.queriesChan)
}

func (svc *ExecuterService) ExecQuery(ctx context.Context, query *model.QueryRequest) (*model.QueryResponse, error) {
	columns, result, err := svc.runQuery(ctx, query.QueryParsed, query.QueryParameters)
	if err != nil {
		return nil, errors.Wrap(err, "cannot exec query")
	}

	resp := &model.QueryResponse{
		Query: query.Query,
		Rows:  result,
	}

	for _, c := range columns {
		resp.Columns = append(resp.Columns, model.QueryColumn{Name: c})
	}

	return resp, nil
}

func (svc *ExecuterService) runQuery(ctx context.Context, query *aqlprocessor.Query, params map[string]any) ([]string, []any, error) {
	var (
		resultColumns []string
		resultRows    []any
		resultErr     error
		done          = make(chan bool)
	)

	q := Query{
		ctx:   ctx,
		query: query,
		args:  params,
		callback: func(columns []string, rows []any, err error) {
			resultColumns = columns
			resultRows = rows
			resultErr = err
			done <- true
		},
	}

	svc.queriesChan <- q
	select {
	case <-ctx.Done():
		return nil, nil, errors.ErrTimeout
	case <-done:
		if resultErr != nil {
			return nil, nil, resultErr
		}
	}

	return resultColumns, resultRows, nil
}
