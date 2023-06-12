package lambdify

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Lambdify accepts a mux and returns a closure
func Lambdify(mux http.Handler) func(events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	return func(ev events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
		var queries string
		for k, v := range ev.QueryStringParameters {
			queries += fmt.Sprintf("&%s=%s", k, v)
		}
		queries = strings.Replace(queries, "&", "?", 1)

		path := ev.Path[strings.Index(strings.TrimLeft(ev.Path, "/"), "/")+1:]

		var body io.Reader
		var contentType string
		if ev.IsBase64Encoded {
			data, err := base64.StdEncoding.DecodeString(ev.Body)
			if err != nil {
				return lambdifyError(err), nil
			}
			body = bytes.NewReader(data)
			contentType = "application/x-www-form-urlencoded"
		} else {
			body = strings.NewReader(ev.Body)
			contentType = "application/json"
		}

		req, err := http.NewRequest(ev.HTTPMethod, path+queries, body)
		if err != nil {
			return lambdifyError(err), nil
		}
		req.Header.Add("Content-Type", contentType)
		for k, v := range ev.Headers {
			req.Header.Add(k, v)
		}
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		headers := map[string]string{"Content-Type": "application/json"}

		multiValueHeaders := make(map[string][]string)
		for k, v := range rr.Result().Header {
			if len(v) > 1 {
				multiValueHeaders[k] = v
			} else if len(v) == 1 {
				headers[k] = v[0]
			}
		}
		return events.ALBTargetGroupResponse{
			Body:              rr.Body.String(),
			IsBase64Encoded:   false,
			StatusCode:        http.StatusOK,
			StatusDescription: "200 OK",
			Headers:           headers,
			MultiValueHeaders: multiValueHeaders,
		}, nil
	}
}

func lambdifyError(err error) events.ALBTargetGroupResponse {
	return events.ALBTargetGroupResponse{
		StatusCode:        http.StatusInternalServerError,
		StatusDescription: "500 Internal Server Error",
		Body:              err.Error(),
		IsBase64Encoded:   false,
	}
}
