package parsing

import (
	"context"
	"github.com/lufeed/feed-parser-api/internal/parser"
	"github.com/lufeed/feed-parser-api/internal/proxy"
	"github.com/lufeed/feed-parser-api/internal/types"
	"net/http"
)

type service interface {
	parseUrl(ctx context.Context, inputUrl string, sendHTML bool) (types.APIResponse, error)
	parseSource(ctx context.Context, inputUrl string, sendHTML bool) (types.APIResponse, error)
}

type serviceImpl struct {
	proxyManager *proxy.Manager
}

func newService(proxyManager *proxy.Manager) service {
	return serviceImpl{
		proxyManager: proxyManager,
	}
}

func (s serviceImpl) parseUrl(ctx context.Context, inputUrl string, sendHTML bool) (types.APIResponse, error) {
	urlParser := parser.NewURLParser(ctx, s.proxyManager)

	source, err := urlParser.Exec(inputUrl, sendHTML)
	if err != nil {
		return types.APIResponse{
			Code: http.StatusBadRequest,
		}, err
	}

	return types.APIResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    source,
	}, nil
}

func (s serviceImpl) parseSource(ctx context.Context, inputUrl string, sendHTML bool) (types.APIResponse, error) {
	sourceParser := parser.NewSourceParser(ctx, s.proxyManager)

	feeds, err := sourceParser.Exec(inputUrl, sendHTML)
	if err != nil {
		return types.APIResponse{
			Code: http.StatusInternalServerError,
		}, err
	}

	return types.APIResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    feeds,
	}, nil
}
