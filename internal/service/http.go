package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mbaitar/gco/agent/internal/config"
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/service/application"
	"github.com/mbaitar/gco/agent/pkg/control"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/gorilla/mux"
)

type httpServer struct {
	controller *control.StateController
}

// StartHTTP registers the known routes and starts listening using the configured address.
// The server is gracefully shut down when the given context is cancelled.
func StartHTTP(ctx context.Context, conf config.Http, controller *control.StateController) {
	if !conf.Enabled {
		log.Debug("HTTP server has not been enabled")
		return
	}

	// bind network address
	lis, err := net.Listen("tcp", conf.GetNetworkAddress())
	if err != nil {
		log.Errorf("failed to listen: %v", err)
		os.Exit(1)
	}

	// create services
	appServer := application.NewServer(controller)

	// register routes
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/v1/applications.list", serviceWrapper(appServer.ListApplications)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/applications.create", serviceWrapper(appServer.CreateApplication)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/applications.get", serviceWrapper(appServer.GetApplication)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/applications.update", serviceWrapper(appServer.UpdateApplication)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/applications.delete", serviceWrapper(appServer.DeleteApplication)).Methods(http.MethodPost)

	server := &http.Server{Handler: router}

	// shut the server down gracefully when the context is cancelled
	go func() {
		<-ctx.Done()
		log.Debug("Gracefully shutting down HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Warnf("Failed to gracefully shut down HTTP server: %v", err)
		}
	}()

	// start listening for HTTP connections
	log.Infof("Started listening for HTTP connections on '%s'", conf.GetNetworkAddress())
	err = server.Serve(lis)
	if errors.Is(err, http.ErrServerClosed) {
		log.Info("HTTP server has been closed")
	} else {
		log.Errorf("Failed to serve HTTP server: %v", err)
		os.Exit(1)
	}

}

func serviceWrapper[ServiceRequest any, ServiceResponse any](handler func(ctx context.Context, sReq *ServiceRequest) (ServiceResponse, error)) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		// allocate a fresh request for every call, sharing one instance
		// would leak fields between requests
		sReq := new(ServiceRequest)

		// read request body
		bytes, err := io.ReadAll(req.Body)
		if err != nil {
			writeHttpError(res, err)
			return
		}

		// parse request body, protojson accepts both the proto and the
		// camelCase field names
		if len(bytes) > 0 {
			if msg, ok := any(sReq).(proto.Message); ok {
				err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(bytes, msg)
			} else {
				err = json.Unmarshal(bytes, sReq)
			}

			if err != nil {
				writeHttpError(res, err)
				return
			}
		}

		// execute handler function
		sRes, err := handler(req.Context(), sReq)
		if err != nil {
			writeHttpError(res, err)
			return
		}

		// format response, keeping the proto field names in the output
		var body []byte
		if msg, ok := any(sRes).(proto.Message); ok {
			body, err = protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
		} else {
			body, err = json.Marshal(sRes)
		}

		if err != nil {
			writeHttpError(res, err)
			return
		}

		// write response
		res.Header().Set("Content-Type", "application/json")
		res.Write(body)
	}
}

func writeHttpError(res http.ResponseWriter, error error) {
	message := error.Error()
	httpStatus := http.StatusInternalServerError

	if s, ok := status.FromError(error); ok {

		// extract status code
		switch s.Code() {
		case codes.InvalidArgument, codes.AlreadyExists:
			httpStatus = http.StatusBadRequest
		case codes.NotFound:
			httpStatus = http.StatusNotFound
		}

		// update message
		message = s.Message()
	}

	body, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		body = []byte(`{"message":"internal server error"}`)
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(httpStatus)
	res.Write(body)
}
