package httpclients

import (
	"context"
	"time"

	"jan-server/mono/apps/backend/internal/infrastructure/logger"

	"resty.dev/v3"
)

type RequestID struct{}
type HTTPClientStartsAt struct{}
type HTTPClientRequestBody struct{}

func NewClient(clientName string) *resty.Client {
	client := resty.New()
	client.AddRequestMiddleware(func(c *resty.Client, r *resty.Request) error {
		start := time.Now()
		ctx := context.WithValue(r.Context(), HTTPClientStartsAt{}, start)
		ctx = context.WithValue(ctx, HTTPClientRequestBody{}, r.Body)
		r.SetContext(ctx)
		return nil
	})
	client.AddResponseMiddleware(func(c *resty.Client, r *resty.Response) error {
		log := logger.GetLogger()
		requestID := r.Request.Context().Value(RequestID{})
		startTime, _ := r.Request.Context().Value(HTTPClientStartsAt{}).(time.Time)
		requestBody := r.Request.Context().Value(HTTPClientRequestBody{})
		latency := time.Since(startTime)
		var responseBody any
		if !r.Request.DoNotParseResponse {
			responseBody = r.Result()
		}

		requestIDStr := ""
		if reqID, ok := requestID.(string); ok {
			requestIDStr = reqID
		}

		log.Debug().
			Str("request_id", requestIDStr).
			Str("client", clientName).
			Int("status", r.StatusCode()).
			Str("method", r.Request.RawRequest.Method).
			Str("path", r.Request.RawRequest.URL.Path).
			Str("query", r.Request.RawRequest.URL.RawQuery).
			Interface("req_body", requestBody).
			Interface("resp_body", responseBody).
			Dur("latency", latency).
			Msg("HTTP client request")
		return nil
	})
	return client
}
