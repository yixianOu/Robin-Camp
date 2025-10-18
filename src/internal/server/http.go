package server

import (
	"net/http"
	"strings"

	v1 "src/api/movie/v1"
	"src/internal/conf"
	"src/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// Custom response encoder to handle 201 status for resource creation
func customResponseEncoder(w http.ResponseWriter, r *http.Request, v interface{}) error {
	// Check if response has status code metadata
	type StatusResponse interface {
		HTTPStatus() int
	}

	if sr, ok := v.(StatusResponse); ok {
		w.WriteHeader(sr.HTTPStatus())
	} else {
		// Check if this is a CreateMovie response
		if strings.Contains(r.URL.Path, "/movies") && r.Method == "POST" && !strings.Contains(r.URL.Path, "/ratings") {
			w.WriteHeader(http.StatusCreated)
		}
	}

	// Use default encoder for the response body
	return khttp.DefaultResponseEncoder(w, r, v)
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, auth *conf.Auth, movieSvc *service.MovieService, logger log.Logger) *khttp.Server {
	var opts = []khttp.ServerOption{
		khttp.Middleware(
			recovery.Recovery(),
			AuthMiddleware(auth.Token),
			RaterIdMiddleware(),
		),
		khttp.ResponseEncoder(customResponseEncoder),
	}
	if c.Http.Network != "" {
		opts = append(opts, khttp.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, khttp.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, khttp.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := khttp.NewServer(opts...)
	v1.RegisterMovieServiceHTTPServer(srv, movieSvc)
	return srv
}
