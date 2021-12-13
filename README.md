# Proxy Service
> A service to proxy requests to a given backend service.

---
## Run (Local)
> Run either of the following from the *same* directory as this README (change values as needed).

### With Flags
```bash
go run main.go \
-debug=true \
-host=localhost \
-port=8080 \
-target-url=http://randomuser.me/ \
-request-delay=2
```

### With `.env` File
```bash
echo \
'DEBUG=true
HOST="localhost"
PORT=8080
TARGET_URL=http://randomuser.me/ 
REQUEST_DELAY=2' > .env
go run main.go
```