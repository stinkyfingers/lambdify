package lambdify

import (
	"fmt"
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

		// path := strings.TrimPrefix(ev.Path, "/badlibs")
		path := ev.Path

		req, err := http.NewRequest(ev.HTTPMethod, path+queries, strings.NewReader(ev.Body))
		if err != nil {
			return events.ALBTargetGroupResponse{
				StatusCode:        http.StatusInternalServerError,
				StatusDescription: "500 Internal Server Error",
				Body:              err.Error(),
				IsBase64Encoded:   false,
			}, nil
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
