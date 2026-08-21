# URL shortener service

## pprof tool application results

`cmd/load_tester/main.c` was used for optimization effort evaluation (GET and 
POST operations 1:1, set via `writeRatio` parameter in `cmd/load_tester/main.c`)

```shell
➜  profiles git:(iter17) ✗ go tool pprof -top -diff_base=base.pprof result.pprof            
File: main
Type: alloc_space
Time: 2026-08-14 23:37:04 MSK
Showing nodes accounting for -194.82MB, 58.77% of 331.50MB total
Dropped 154 nodes (cum <= 1.66MB)
      flat  flat%   sum%        cum   cum%
  -37.90MB 11.43% 11.43%   -50.28MB 15.17%  compress/flate.NewWriter (inline)
  -26.51MB  8.00% 19.43%   -26.51MB  8.00%  github.com/rs/zerolog.Logger.With (partial-inline)
  -13.50MB  4.07% 23.50%   -13.50MB  4.07%  net/http.Header.Clone (inline)
  -11.75MB  3.54% 27.05%   -11.75MB  3.54%  compress/flate.(*compressor).initDeflate (inline)
     -10MB  3.02% 30.07%   -11.50MB  3.47%  context.(*cancelCtx).propagateCancel
   -9.50MB  2.87% 32.93%    -9.50MB  2.87%  net/textproto.readMIMEHeader
   -6.50MB  1.96% 34.89%    -6.50MB  1.96%  github.com/labstack/echo/v5.(*Context).Set
      -6MB  1.81% 36.70%       -6MB  1.81%  net/textproto.MIMEHeader.Set (inline)
      -6MB  1.81% 38.51%       -6MB  1.81%  net/url.parse
   -5.50MB  1.66% 40.17%    -5.50MB  1.66%  encoding/json.NewDecoder (inline)
   -5.50MB  1.66% 41.83%      -23MB  6.94%  net/http.(*conn).readRequest
      -5MB  1.51% 43.34%       -5MB  1.51%  github.com/golang-jwt/jwt/v5.NewWithClaims (inline)
   -4.50MB  1.36% 44.70%      -16MB  4.83%  net/http.readRequest
   -4.50MB  1.36% 46.06%    -4.50MB  1.36%  github.com/jackc/pgx/v5.(*Conn).getRows
   -4.50MB  1.36% 47.41%   -18.50MB  5.58%  github.com/apomazanov/shortener/internal/jwt.(*Data).CreateCookieWithUserID
   -4.50MB  1.36% 48.77%    -9.50MB  2.87%  github.com/jackc/pgx/v5/pgconn/ctxwatch.(*ContextWatcher).Watch
      -4MB  1.21% 49.98%       -4MB  1.21%  encoding/json.(*Decoder).refill
   -2.50MB  0.75% 50.73%    -2.50MB  0.75%  time.newTimer
      -2MB   0.6% 51.34%    -8.50MB  2.56%  github.com/golang-jwt/jwt/v5.(*Token).SignedString
      -2MB   0.6% 51.94%   -11.50MB  3.47%  context.WithDeadlineCause
      -2MB   0.6% 52.54%    -3.50MB  1.06%  github.com/golang-jwt/jwt/v5.(*SigningMethodHMAC).Sign
      -2MB   0.6% 53.15%       -2MB   0.6%  github.com/labstack/echo/v5/middleware.GzipConfig.ToMiddleware.func1
      -2MB   0.6% 53.75%       -2MB   0.6%  github.com/labstack/echo/v5/middleware.DecompressWithConfig.toMiddlewareOrPanic.DecompressConfig.ToMiddleware.func1
   -1.50MB  0.45% 54.20%    -1.50MB  0.45%  net/url.(*URL).joinPath
   -1.50MB  0.45% 54.66%    -2.50MB  0.75%  crypto/internal/fips140/hmac.New[go.shape.interface { BlockSize int; Reset; Size int; Sum []uint8; Write  }]
   -1.50MB  0.45% 55.11%    -1.50MB  0.45%  context.withCancel (inline)
   -1.50MB  0.45% 55.56%    -1.50MB  0.45%  net.(*conn).Read
   -1.50MB  0.45% 56.01%      -15MB  4.53%  github.com/apomazanov/shortener/internal/repository.(*PostgresStorage).Save
   -1.50MB  0.45% 56.47%    -1.50MB  0.45%  github.com/jackc/pgx/v5/pgtype.scanPlanString.Scan
   -1.50MB  0.45% 56.92%    -1.50MB  0.45%  reflect.unsafe_New
      -1MB   0.3% 57.22%       -2MB   0.6%  encoding/json.Marshal
      -1MB   0.3% 57.52%  -157.31MB 47.45%  github.com/labstack/echo/v5/middleware.GzipConfig.ToMiddleware.func1.1
      -1MB   0.3% 57.82%   -20.50MB  6.18%  github.com/apomazanov/shortener/internal/handlers.handleCookie
      -1MB   0.3% 58.12%    -2.50MB  0.75%  github.com/jackc/pgx/v5.(*baseRows).Scan
   -0.64MB  0.19% 58.32%   -12.38MB  3.74%  compress/flate.(*compressor).init
   -0.50MB  0.15% 58.47%       -5MB  1.51%  context.AfterFunc
   -0.50MB  0.15% 58.62%       -2MB   0.6%  context.WithCancel
   -0.50MB  0.15% 58.77%   -62.02MB 18.71%  github.com/apomazanov/shortener/internal/handlers.(*Handler).CreateJSON
         0     0% 58.77%    -5.50MB  1.66%  bytes.(*Buffer).WriteTo
         0     0% 58.77%   -49.65MB 14.98%  compress/gzip.(*Writer).Close
         0     0% 58.77%   -50.28MB 15.17%  compress/gzip.(*Writer).Write
         0     0% 58.77%   -11.50MB  3.47%  context.WithDeadline (inline)
         0     0% 58.77%   -11.50MB  3.47%  context.WithTimeout
         0     0% 58.77%    -2.50MB  0.75%  crypto/hmac.New
         0     0% 58.77%    -4.50MB  1.36%  encoding/json.(*Decoder).Decode
         0     0% 58.77%    -4.50MB  1.36%  encoding/json.(*Decoder).readValue
         0     0% 58.77%   -28.01MB  8.45%  github.com/apomazanov/shortener/internal/handlers.(*Handler).Get
         0     0% 58.77%      -14MB  4.22%  github.com/apomazanov/shortener/internal/repository.(*PostgresStorage).Get
         0     0% 58.77%   -14.50MB  4.37%  github.com/apomazanov/shortener/internal/service.(*Service).CreateURLAlias
         0     0% 58.77%      -14MB  4.22%  github.com/apomazanov/shortener/internal/service.(*Service).GetOriginalURL
         0     0% 58.77%    -4.50MB  1.36%  github.com/apomazanov/shortener/internal/validator.(*MyValidator).Validate
         0     0% 58.77%    -4.50MB  1.36%  github.com/go-playground/validator/v10.(*Validate).Struct (inline)
         0     0% 58.77%    -4.50MB  1.36%  github.com/go-playground/validator/v10.(*Validate).StructCtx
         0     0% 58.77%       -4MB  1.21%  github.com/go-playground/validator/v10.(*validate).traverseField
         0     0% 58.77%       -4MB  1.21%  github.com/go-playground/validator/v10.(*validate).validateStruct
         0     0% 58.77%       -3MB  0.91%  github.com/go-playground/validator/v10.New.wrapFunc.func3
         0     0% 58.77%       -3MB  0.91%  github.com/go-playground/validator/v10.isURL
         0     0% 58.77%       -3MB  0.91%  github.com/golang-jwt/jwt/v5.(*Token).SigningString
         0     0% 58.77%   -12.50MB  3.77%  github.com/jackc/pgx/v5.(*Conn).Query
         0     0% 58.77%   -12.50MB  3.77%  github.com/jackc/pgx/v5.(*Conn).QueryRow (inline)
         0     0% 58.77%    -2.50MB  0.75%  github.com/jackc/pgx/v5.(*connRow).Scan
         0     0% 58.77%       -9MB  2.72%  github.com/jackc/pgx/v5/pgconn.(*PgConn).ExecStatement
         0     0% 58.77%    -9.50MB  2.87%  github.com/jackc/pgx/v5/pgconn.(*PgConn).execExtendedPrefix
         0     0% 58.77%   -12.50MB  3.77%  github.com/jackc/pgx/v5/pgxpool.(*Conn).QueryRow
         0     0% 58.77%   -13.50MB  4.07%  github.com/jackc/pgx/v5/pgxpool.(*Pool).QueryRow
         0     0% 58.77%    -2.50MB  0.75%  github.com/jackc/pgx/v5/pgxpool.(*poolRow).Scan
         0     0% 58.77%    -1.14MB  0.34%  github.com/labstack/echo-contrib/v5/pprof.Register.handler.func2
         0     0% 58.77%      -10MB  3.02%  github.com/labstack/echo/v5.(*Context).Bind (inline)
         0     0% 58.77%    -4.50MB  1.36%  github.com/labstack/echo/v5.(*Context).Validate (inline)
         0     0% 58.77%      -10MB  3.02%  github.com/labstack/echo/v5.(*DefaultBinder).Bind
         0     0% 58.77%  -179.32MB 54.09%  github.com/labstack/echo/v5.(*Echo).ServeHTTP
         0     0% 58.77%  -179.32MB 54.09%  github.com/labstack/echo/v5.(*Echo).serveHTTP
         0     0% 58.77%    -5.50MB  1.66%  github.com/labstack/echo/v5.(*Response).Write
         0     0% 58.77%   -13.50MB  4.07%  github.com/labstack/echo/v5.(*Response).WriteHeader
         0     0% 58.77%      -10MB  3.02%  github.com/labstack/echo/v5.BindBody
         0     0% 58.77%      -10MB  3.02%  github.com/labstack/echo/v5.DefaultJSONSerializer.Deserialize
         0     0% 58.77%    -6.50MB  1.96%  github.com/labstack/echo/v5.applyMiddleware
         0     0% 58.77%   -93.16MB 28.10%  github.com/labstack/echo/v5/middleware.BodyLimitWithConfig.toMiddlewareOrPanic.BodyLimitConfig.ToMiddleware.func2.1
         0     0% 58.77%   -93.16MB 28.10%  github.com/labstack/echo/v5/middleware.DecompressWithConfig.toMiddlewareOrPanic.DecompressConfig.ToMiddleware.func1.1
         0     0% 58.77%   -62.65MB 18.90%  github.com/labstack/echo/v5/middleware.GzipConfig.ToMiddleware.func1.1.1
         0     0% 58.77%  -172.32MB 51.98%  github.com/labstack/echo/v5/middleware.RecoverWithConfig.toMiddlewareOrPanic.RecoverConfig.ToMiddleware.func1.1
         0     0% 58.77%  -172.32MB 51.98%  github.com/labstack/echo/v5/middleware.RequestIDWithConfig.toMiddlewareOrPanic.RequestIDConfig.ToMiddleware.func1.1
         0     0% 58.77%   -90.02MB 27.16%  main.run.AuditRecorder.func8.1
         0     0% 58.77%   -64.02MB 19.31%  main.run.Authenticator.func7.1
         0     0% 58.77%  -165.82MB 50.02%  main.run.Zerologger.func6.1
         0     0% 58.77%  -204.32MB 61.64%  net/http.(*conn).serve
         0     0% 58.77%    -1.50MB  0.45%  net/http.(*connReader).backgroundRead
         0     0% 58.77%   -13.50MB  4.07%  net/http.(*response).WriteHeader
         0     0% 58.77%    -1.14MB  0.34%  net/http.HandlerFunc.ServeHTTP
         0     0% 58.77%       -6MB  1.81%  net/http.Header.Set
         0     0% 58.77%  -179.32MB 54.09%  net/http.serverHandler.ServeHTTP
         0     0% 58.77%    -1.14MB  0.34%  net/http/pprof.handler.ServeHTTP
         0     0% 58.77%    -9.50MB  2.87%  net/textproto.(*Reader).ReadMIMEHeader (inline)
         0     0% 58.77%       -3MB  0.91%  net/url.JoinPath
         0     0% 58.77%    -3.50MB  1.06%  net/url.Parse
         0     0% 58.77%    -2.50MB  0.75%  net/url.ParseRequestURI
         0     0% 58.77%    -1.14MB  0.34%  runtime/pprof.(*Profile).WriteTo
         0     0% 58.77%    -1.14MB  0.34%  runtime/pprof.(*profileBuilder).appendLocsForStack
         0     0% 58.77%    -1.14MB  0.34%  runtime/pprof.(*profileBuilder).emitLocation
         0     0% 58.77%    -1.14MB  0.34%  runtime/pprof.writeAlloc
         0     0% 58.77%    -1.14MB  0.34%  runtime/pprof.writeHeapInternal
         0     0% 58.77%    -1.14MB  0.34%  runtime/pprof.writeHeapProto
         0     0% 58.77%    -2.50MB  0.75%  time.AfterFunc
```
