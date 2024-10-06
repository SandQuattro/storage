![](https://community.alteryx.com/t5/image/serverpage/image-id/256457i72705B6D0ABAE7D6?v=v2)

# Storage API

#### Version 1.0

## Common

### About service
The Storage API (here and after referred to as *API*) provides a set of methods necessary for multiple file uploading using Websocket following by transferring files chunks to AWS S3-like object storage or multipart file upload processing.


### Technologies:

Microservice core: Echo web microframework v4

Authorization: JWT token, public key verification, jwt parsing / validation

File Storage: MINIO S3 Object Storage

Configuration: Hocon config

Logging: LogDoc logging subsystem

Database: File Metadata storage using Postgres DB, sqlx

Can be deployed to Docker, Dockerfile included

Observalibity: 

- Opentracing to Jaeger UI or my custom trace collector with LogDoc trace processing
- Prometheus metrics (golang standart + custom business metrics) with Grafana visualization
- LogDoc logging visualization
- Asynq queue monitoring using asynqmon

### Middlewares, Features

Custom middlewares for Authorization header processing, custom CORS processing, multipart body validation

Rate limiter middleware, rate limit: 20 rps/sec, burst: 20 (maximum number of requests to pass at the same moment)

LogDoc logging subsystem, ClickHouse-based high performance logging collector https://logdoc.org/en/

Graceful shutdown

### Building

Using Makefile:  make rebuild, restart, run, etc

### TODO
- [x] combine backend and frontend into one application
- [ ] add docker-compose multistage projct building 
- [ ] migrate to fiber framework
- [ ] Communication Bus: Asynq (Redis-based async queue) for incidents notification by telegram, sending emails, etc.
- [ ] Migrations: golang-migrate
- [ ] pprof profiling in debug mode
- [ ] SIGHUP signal config reloading
- [ ] Teler WAF (Intrusion Detection Middleware) https://github.com/kitabisa/teler-waf.git
