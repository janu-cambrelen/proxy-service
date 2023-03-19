# Proxy Service

> A service to proxy requests to a given backend service.

---
Go 1.17+

## Clone

```
git clone git@github.com:janu-cambrelen/proxy-service.git
```

## Run (Local)
*Run either of the following from the **same** directory as this README (change values as needed).*

### With Flags
```bash
go run main.go \
-debug=true \
-host=localhost \
-port=8080 \
-target-url=http://jsonplaceholder.typicode.com/ \
-body-methods-only=true \
-reject-with=bad_message \
-reject-exact=true \
-reject-insensitive=false \
-request-delay=2
```

### With `.env` File
```bash
echo \
'DEBUG=true
HOST="localhost"
PORT=8080
TARGET_URL=http://jsonplaceholder.typicode.com/
BODY_METHODS_ONLY=true
REJECT_WITH=bad_message
REJECT_EXACT=true
REJECT_INSENSITIVE=false
REQUEST_DELAY=2' > .env
go run main.go
```
---
## Docker
Build
```bash
docker build -t proxy-scratch . 
```
Run
```bash
docker run -p 8080:8080 -it --rm proxy-scratch \
-debug=true \
-host=0.0.0.0 \
-port=8080 \
-target-url=http://jsonplaceholder.typicode.com/ \
-body-methods-only=false \
-reject-with=bad_message \
-reject-exact=true \
-reject-insensitive=false \
-request-delay=2
```
---
## Overview / Usage

#### **Target URL:**
The target backend "service" is set using the `TARGET_URL` environment file setting or by the `-target-url` CLI flag. The server will fail to initialize if the target url is not provided.

If, for example, we wanted to hit the `http://backend-service.com/user` endpoint of our backend service we would set `http://backend-service.com/` as our target URL.

The proxy server will handle routing and map the URI accordingly. Therefore, to hit the `/user` endpoint from the proxy server, the request should be made to `http://proxy-service.com/user`.

---
#### **Supported HTTP Methods and Content-Type:**
This service only accepts `POST`, `PUT`, and `PATCH` requests if the `BODY_METHODS_ONLY` environment file setting or the `body-methods-only` CLI flag is set to `true`.  Otherwise, this service is able to support other methods (including `GET` requests with query params etc.).

Also, all requests and responses should be of content type `application/json`.

The client will receive an error, detailing the issue, if the aforementioned is not conformed to.


---
#### **Request Filtering:**
This service can reject requests from ever reaching the backend / target based on a specfied word or phrase contained within the `REJECT_WITH` environment file setting or by the `reject-with` CLI flag.

Whether this validation is concerned with an "exact" match or a "contains" match is determined by the `REJECT_EXACT` environment file setting or by the `reject-exact` CLI flag.

Finally, whether this check is case-sensitive is determined by the `REJECT_INSENSITIVE` environment file setting or by the `reject-insensitive` CLI flag.

---
#### **Consecutive Request Delay:**

The service will delay its response by *two seconds* if consective requests, containing the same content (i.e., request body) and common set of headers, are received.

The delay can be changed via the `REQUEST_DELAY` environment file setting or by the `-request-delay` CLI flag. Only positive integer values are supported.

The server will fail to initialize if the `REQUEST_DELAY` value is negative; otherwise, if not set, it will default to two seconds.

---
> Sample Backend Service: https://jsonplaceholder.typicode.com/

### Sample Request (successful)
```bash
curl --location --include --request PATCH 'http://localhost:8080/posts/1' \
--header 'Content-Type: application/json' \
--data-raw '{
    "body": "good_message"
}'
```
### Sample Response (successful)
```bash
HTTP/1.1 200 OK
Content-Type: application/json
X-Proxy-Request-Id: 6ad80a7c-c132-4417-8323-fdfcaf7365e8
Date: Tue, 14 Dec 2021 02:06:27 GMT
Content-Length: 138

{
  "userId": 1,
  "id": 1,
  "title": "sunt aut facere repellat provident occaecati excepturi optio reprehenderit",
  "body": "good_message"
}
```
### Sample Request (unsuccessful)
```bash

curl --location --include --request PATCH 'http://localhost:8080/posts/1' \
--header 'Content-Type: application/json' \
--data-raw '{
    "body": "bad_message"
}'
```

### Sample Response (unsuccessful)
```bash

HTTP/1.1 401 Unauthorized
Content-Length: 62
Content-Type: application/json
Date: Tue, 14 Dec 2021 02:08:21 GMT

{"code":"401","msg":"rejected because `bad_message` found within request body"}
```
---
## Test
Unit Tests
```bash
go test ./internal/proxyserver -v 
```
End-to-End Tests
```bash
bash scripts/e2e.sh
```
or
```bash
./scripts/e2e.sh 
```
All
```bash
go test ./internal/proxyserver -v && ./scripts/e2e.sh 
```
> NOTE: May need to `chmod +x ./scripts/e2e.sh` if you encounter permissions issue.
---
## Docs
Update documentation within the `doc` directory:
```bash
go doc --all cmd  > doc/cmd.txt 
go doc --all internal/proxyserver > doc/proxyserver.txt
```
---