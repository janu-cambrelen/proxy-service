package proxyserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProxyServer defines the HTTP proxy server.
type ProxyServer struct {
	debug             bool
	targetURL         string
	requestDelay      uint
	bodyMethodsOnly   bool
	rejectWith        string
	rejectExact       bool
	rejectInsensitive bool
	logger            *zap.Logger
	priorRequest      RequestCopy
}

// RequestCopy defines a request representation that is used to compare requests.
// The values are captured from incoming HTTP requests.
type RequestCopy struct {
	Method    string
	TargetURL string
	TargetURI string
	Header    headerCopy
	Body      []byte
}

// headerCopy defines a subset of common, non-auth, related headers.
// These headers will be used when comparing two requests.
// The values are captured from incoming HTTP request headers.
type headerCopy struct {
	Host           []string
	Accept         []string
	UserAgent      []string
	Connection     []string
	ContentType    []string
	ContentLength  []string
	AcceptEncoding []string
}

// proxyErrorResponse defines a server error that is marshalled into JSON and returned to the client.
type proxyErrorResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// NewProxyServer constructor creates a new ProxyServer.
func NewProxyServer(
	d bool,
	turl string,
	rd uint,
	bmo bool,
	rw string,
	re bool,
	ri bool,
	l *zap.Logger,
	pr RequestCopy,
) *ProxyServer {
	s := &ProxyServer{
		debug:             d,
		targetURL:         turl,
		requestDelay:      rd,
		bodyMethodsOnly:   bmo,
		rejectWith:        rw,
		rejectExact:       re,
		rejectInsensitive: ri,
		logger:            l,
		priorRequest:      pr,
	}
	return s
}

// ServeHTTP is the main handler used by the server.
func (s *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// validate request method
	if s.bodyMethodsOnly {
		methodAllowed := false
		allowedMethods := [3]string{"POST", "PUT", "PATCH"}
		for _, m := range allowedMethods {
			if m == r.Method {
				methodAllowed = true
			}
		}
		if !methodAllowed {
			s.writeError(w, 405, "`"+r.Method+"` method not allowed, this proxy server only supports `POST, PUT, PATCH` requests")
			return
		}
	}

	// validate content type
	if r.Header.Get("Content-Type") != "application/json" {
		s.writeError(w, 415, "Content-Type header must be `application/json`")
		return
	}

	// create identical ReadClosers from request body
	r1, r2, err := s.copyBody(r.Body)
	if err != nil {
		s.writeError(w, 400, "invalid request body")
		return
	}

	// set request body again for future use
	r.Body = r1

	// create a byte array representation of the original request body
	cb, err := io.ReadAll(r2)
	if err != nil {
		s.writeError(w, 400, "invalid request body")
		return
	}

	// reject requests with the word/phrase within the string value of `s.RejectWith`
	// whether the check is "exact" or "contains" is determined by the `s.RejectExact` boolean
	// please refer to the method's documentation for additional context
	if s.rejectWith != "" {
		err = s.validateRequestBody(string(cb))
		if err != nil {
			// consider whether `400 BAD REQUEST` or `422 UNPROCESSABLE ENTITY`
			// is more fitting than `401 UNAUTHORIZED`
			s.writeError(w, 401, err.Error())
			return
		}
	}

	// delay response for consecutive requests
	ch := s.copyHeader(r.Header)

	pr := s.priorRequest
	cr := RequestCopy{
		Method:    r.Method,
		TargetURL: s.targetURL,
		TargetURI: r.RequestURI,
		Header:    ch,
		Body:      cb,
	}

	if cmp.Equal(pr, cr) {
		d := time.Duration(s.requestDelay * uint(time.Second))
		s.logger.Info("consecutive requests detected, delaying response", zap.Any("seconds", s.requestDelay))
		time.Sleep(d)
	}

	s.priorRequest = cr

	// prepare request to hit backend service
	req, err := s.prepareRequest(r)
	if err != nil {
		// although the url is already validated in the `Config.validate()` method
		// an additional parse attempt takes place within the the `prepareRequest`
		// method. A 500 is returned since target url is not something the client
		// is able to provide.
		s.writeError(w, 500, err.Error())
		return
	}

	// create a request id that will be set to the `X-Proxy-Request-ID` response
	reqID := uuid.NewString()
	s.logger.Info("processing", zap.String("X-Proxy-Request-ID", reqID))

	// make request backend service and write the result to the client
	code, err := s.requestBackendService(w, req, reqID)
	// the status code is only intended for use when the server encounters an error
	if err != nil {
		s.writeError(w, code, err.Error())
		return
	}

}

// WithRequestLoggerMiddleware is "middleware" that logs every request
func (s *ProxyServer) WithRequestLoggerMiddleware() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rLog, err := httputil.DumpRequest(r, true)
		if err != nil {
			s.writeError(w, 400, "bad request")
			return
		}

		s.logger.Info("request", zap.ByteString("payload", rLog))
		s.ServeHTTP(w, r)
	})
}

// validateRequestBody validates whether the body includes an unwanted word or phrase.
// Whether this validation is concerned with an "exact" match or "contains" is determined by the
// value of the `RejectExact` boolean. Whether this check is case-sensitive is determined by the RejectInsensitive
// boolean. Lastly, the value it validates against is determined by the value of the `RejectWith` string.
func (s *ProxyServer) validateRequestBody(b string) error {
	v := s.rejectWith
	// make validation case-insensitive
	if s.rejectInsensitive {
		b = strings.ToLower(b)
		v = strings.ToLower(v)
	}

	// general contains case
	invalid := []string{v}

	// specific / exact cases, consider regex
	if s.rejectExact {
		c1 := fmt.Sprintf(" %s ", v)   // ` bad_message `
		c2 := fmt.Sprintf("\"%s\"", v) // `"bad_message"`
		c3 := fmt.Sprintf(" %s\"", v)  // ` bad_message"`
		c4 := fmt.Sprintf("\"%s ", v)  // `"bad_message `
		invalid = []string{c1, c2, c3, c4}
	}

	for _, c := range invalid {
		if strings.Contains(b, c) {
			return errors.New("rejected because `" + v + "` found within request body")
		}
	}

	return nil
}

// prepareRequest modifies the client's request and routes URLs to the scheme,
// host, and base path provided in target. If the target's path is "/base" and
// the incoming request was for "/dir", the target request will be for /base/dir.
func (s *ProxyServer) prepareRequest(r *http.Request) (*http.Request, error) {
	target, err := url.Parse(s.targetURL)
	if err != nil {
		return nil, errors.New("unable to parse target url")
	}

	req := r
	req.Host = ""
	req.RequestURI = ""
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.URL.Path, req.URL.RawPath = func(a *url.URL, b *url.URL) (path, rawpath string) {
		if a.RawPath == "" && b.RawPath == "" {
			return func(a string, b string) string {
				aslash := strings.HasSuffix(a, "/")
				bslash := strings.HasPrefix(b, "/")
				switch {
				case aslash && bslash:
					return a + b[1:]
				case !aslash && !bslash:
					return a + "/" + b
				}
				return a + b
			}(a.Path, b.Path), ""
		}

		apath := a.EscapedPath()
		bpath := b.EscapedPath()

		aslash := strings.HasSuffix(apath, "/")
		bslash := strings.HasPrefix(bpath, "/")

		switch {
		case aslash && bslash:
			return a.Path + b.Path[1:], apath + bpath[1:]
		case !aslash && !bslash:
			return a.Path + "/" + b.Path, apath + "/" + bpath
		}
		return a.Path + b.Path, apath + bpath

	}(target, req.URL)
	// support query params
	targetQuery := target.RawQuery
	if targetQuery == "" || req.URL.RawQuery == "" {
		req.URL.RawQuery = targetQuery + req.URL.RawQuery
	} else {
		req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
	}

	req = s.sanitizeHeader(r)

	return req, nil
}

// sanitizeHeader takes in a request and removes hop-by-hop headers and those that are
// unnecessary or could get in the way of processing the proxied request or consuming the proxied
// response (e.g., Accept-Encoding). This method is used within the `prepareRequest` method.
func (s *ProxyServer) sanitizeHeader(r *http.Request) *http.Request {
	r.Header.Del("Te")
	r.Header.Del("Trailer")
	r.Header.Del("Upgrade")
	r.Header.Del("Keep-Alive")
	r.Header.Del("Connection")
	r.Header.Del("Accept-Encoding")
	r.Header.Del("Transfer-Encoding")
	r.Header.Del("Proxy-Connection")
	r.Header.Del("Proxy-Authenticate")
	r.Header.Del("Proxy-Authorization")
	return r
}

// requestBackendService is the method that actually makes the request to the backend service.
// It also adds the `X-Proxy-Request-ID`, which is a UUID v4 string, to the header of every response
// from the backend.
func (s *ProxyServer) requestBackendService(w http.ResponseWriter, r *http.Request, reqID string) (code int, err error) {
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return 502, errors.New("bad gateway")
	}
	defer client.CloseIdleConnections()

	w.Header().Add("X-Proxy-Request-ID", reqID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	newResp, err := io.Copy(w, resp.Body)
	if err := resp.Body.Close(); err != nil {
		s.logger.Error("failed to close response", zap.Error(err))
	}
	s.logger.Debug("copied bytes to client", zap.Int64("body", newResp))

	// the status code is only used when the server encounters an error
	// returning 418 I'M A TEAPOT status code as a place holder.
	return 418, nil
}

// writeError writes and logs a JSON HTTP response error that conforms to the `proxyErrorResponse` struct defined above.
func (s *ProxyServer) writeError(w http.ResponseWriter, code int, msg string) {
	codeString := strconv.Itoa(code)
	errJSON := proxyErrorResponse{
		Code: codeString,
		Msg:  msg,
	}

	s.logger.Info("response", zap.Any("error", errJSON))

	buf, err := json.Marshal(&errJSON)
	if err != nil {
		buf = []byte("{\"code\": \"" + codeString + "\"\"msg\": \"There was a response that could not be serialized into JSON\"}")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.WriteHeader(code)

	_, writeErr := w.Write(buf)
	if writeErr != nil {
		// avoid unnecessary log / stderr messages in the unlikely event `w.Write()` returns an error
		_ = writeErr
	}
}

// copyBody returns two ReadClosers that yield the same bytes.
func (s *ProxyServer) copyBody(b io.ReadCloser) (r1 io.ReadCloser, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		return http.NoBody, http.NoBody, nil
	}

	buf, err := ioutil.ReadAll(b)
	if err != nil {
		return nil, nil, err
	}

	r1 = ioutil.NopCloser(bytes.NewBuffer(buf))
	r2 = ioutil.NopCloser(bytes.NewBuffer(buf))
	return r1, r2, nil
}

// copyHeader creates and returns a HeaderCopy.
func (s *ProxyServer) copyHeader(h http.Header) (hc headerCopy) {
	for k, vv := range h {
		switch k {
		case "Host":
			hc.Host = vv
		case "Accept":
			hc.Accept = vv
		case "User-Agent":
			hc.UserAgent = vv
		case "Connection":
			hc.Connection = vv
		case "Content-Type":
			hc.ContentType = vv
		case "Content-Length":
			hc.ContentLength = vv
		case "AcceptEncoding":
			hc.AcceptEncoding = vv
		default:
			// do nothing
		}
	}
	return hc
}
