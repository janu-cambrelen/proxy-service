package e2e

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// REF: https://jsonplaceholder.typicode.com/
// All HTTP methods are supported. You can use http or https for your requests.

// GET 		/posts
// GET 		/posts/1
// GET 		/posts/1/comments
// GET 		/comments?postId=1
// POST 	/posts
// PUT 		/posts/1
// PATCH 	/posts/1
// DELETE 	/posts/1

// End-to-End test to test the core functionality of the proxy-service.
func TestProxyService(t *testing.T) {
	assert := assert.New(t)

	t.Run("ProxyServerE2ETest", func(t *testing.T) {

		newReq := func(method, url string, body io.Reader) *http.Request {
			r, err := http.NewRequest(method, url, body)
			r.Header.Set("Content-Type", "application/json")
			if err != nil {
				t.Fatal(err)
			}
			return r
		}

		type e2eTestCase struct {
			name          string
			req           *http.Request
			code          int  // expected status code
			methodAllowed bool // whether the request method is allowed
			xProxyReqId   bool // whether the response contains a `X-Proxy-Request-ID`
		}

		// whether non PUT, POST, PATCH methods are allowed
		alw := tc.cfg.BodyMethodsOnly

		for _, tCase := range []e2eTestCase{
			// valid payloads (should respond with 201 or 200)
			{name: "test valid post", req: newReq("POST", tc.server.URL+"/posts", bytes.NewBuffer([]byte(`{"body": "good_message"}`))), code: 201, methodAllowed: true, xProxyReqId: true},
			{name: "test valid put", req: newReq("PUT", tc.server.URL+"/posts/1", bytes.NewBuffer([]byte(`{"body": "good_message"}`))), code: 200, methodAllowed: true, xProxyReqId: true},
			{name: "test valid patch", req: newReq("PATCH", tc.server.URL+"/posts/1", bytes.NewBuffer([]byte(`{"body": "good_message"}`))), code: 200, methodAllowed: true, xProxyReqId: true},
			// invalid payloads (should respond with status code 401)
			{name: "test invalid post", req: newReq("POST", tc.server.URL+"/posts", bytes.NewBuffer([]byte(`{"body": "bad_message"}`))), code: 401, methodAllowed: true, xProxyReqId: false},
			{name: "test invalid put", req: newReq("PUT", tc.server.URL+"/posts/1", bytes.NewBuffer([]byte(`{"body": "bad_message"}`))), code: 401, methodAllowed: true, xProxyReqId: false},
			{name: "test invalid patch", req: newReq("PATCH", tc.server.URL+"/posts/1", bytes.NewBuffer([]byte(`{"body": "bad_message"}`))), code: 401, methodAllowed: true, xProxyReqId: false},
			// GET and DELETE tests (expected status code and whether proxy ID should be present depends on whether the methods are allowed)
			{name: "test get", req: newReq("GET", tc.server.URL+"/posts/1", nil), code: 200, methodAllowed: !alw, xProxyReqId: !alw},
			{name: "test delete", req: newReq("DELETE", tc.server.URL+"/posts/1", nil), code: 200, methodAllowed: !alw, xProxyReqId: !alw},
		} {
			t.Run(tCase.name, func(t *testing.T) {
				// sleep to be nice to the remote test server
				// also, it may be necessary to account for the built-in consecutive request delay
				time.Sleep(time.Duration(tc.cfg.RequestDelay+1) * time.Second)

				resp, err := http.DefaultClient.Do(tCase.req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()

				t.Run("Status Code", func(t *testing.T) {
					if tCase.methodAllowed {
						assert.Equal(tCase.code, resp.StatusCode)
					} else {
						assert.Equal(405, resp.StatusCode)
					}
				})

				t.Run("X-Proxy-Request-ID Header", func(t *testing.T) {
					if tCase.xProxyReqId && tCase.methodAllowed {
						id := resp.Header.Get("X-Proxy-Request-ID")
						assert.NotEmpty(id)
					}
				})
			})
		}
	})

	t.Run("ConsecutiveRequestTest", func(t *testing.T) {
		// this test should take longer than or, roughly, equal to `minTime`
		minTime := tc.cfg.RequestDelay * 2
		start := time.Now()
		for _, uri := range []string{
			"/posts", // < 1 sec
			"/posts", // >= 2 secs
			"/posts", // >= 2 secs
		} {
			t.Run("Run @"+start.String(), func(t *testing.T) {
				resp, err := http.DefaultClient.Post(tc.server.URL+uri, "application/json", bytes.NewBuffer([]byte(`{"body": "good_message"}`)))
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
			})
		}
		duration := time.Since(start).Seconds()
		assert.GreaterOrEqual(uint(duration), minTime)
	})

	t.Run("ContentTypeTest", func(t *testing.T) {

		type contentTypeTest struct {
			ct   string // content-type
			code int    // expected status code
		}

		for _, tCase := range []contentTypeTest{
			// only valid content type
			{ct: "application/json", code: 201},
			// invalid content types
			{ct: "application/gzip", code: 415},
			{ct: "application/zip", code: 415},
			{ct: "application/pdf", code: 415},
			{ct: "text/javascript", code: 415},
			{ct: "text/html", code: 415},
		} {
			t.Run(tCase.ct, func(t *testing.T) {
				// sleep to be nice to the remote test server
				// also, it may be necessary to account for the built-in consecutive request delay
				time.Sleep(time.Duration(tc.cfg.RequestDelay+1) * time.Second)

				resp, err := http.DefaultClient.Post(tc.server.URL+"/posts", tCase.ct, bytes.NewBuffer([]byte(`{"body": "good_message"}`)))
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()

				assert.Equal(tCase.code, resp.StatusCode)
			})
		}
	})

}
