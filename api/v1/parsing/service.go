package parsing

import (
	"context"
	"github.com/lufeed/feed-parser-api/internal/parser"
	"github.com/lufeed/feed-parser-api/internal/types"
	"github.com/mmcdole/gofeed"
	"net"
	"net/http"
	"time"
)

type service interface {
	parseUrl(ctx context.Context, inputUrl string) (types.APIResponse, error)
	parseSource(ctx context.Context, inputUrl string) (types.APIResponse, error)
}

type serviceImpl struct {
}

func newService() service {
	return serviceImpl{}
}

func (s serviceImpl) parseUrl(ctx context.Context, inputUrl string) (types.APIResponse, error) {
	fp := gofeed.NewParser()
	fp.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 Edg/134.0.0.0"
	cl := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   60 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
		},
	}
	urlParser := parser.NewURLParser(ctx, fp, cl)

	source, err := urlParser.Exec(inputUrl)
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

func (s serviceImpl) parseSource(ctx context.Context, inputUrl string) (types.APIResponse, error) {
	fp := gofeed.NewParser()
	fp.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 Edg/134.0.0.0"
	cl := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   60 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
		},
	}
	sourceParser := parser.NewSourceParser(ctx, fp, cl)

	feeds, err := sourceParser.Exec(inputUrl)
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
