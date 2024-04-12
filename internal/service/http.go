package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	applicationv1 "github.com/mabaitar/gco/agent/gen/proto/application/v1"
	"github.com/mabaitar/gco/agent/internal/config"
	"github.com/mabaitar/gco/agent/internal/log"
	"github.com/mabaitar/gco/agent/internal/service/application"
	"github.com/mabaitar/gco/agent/pkg/control"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gorilla/mux"
)

type httpServer struct {
	controller *control.StateController
}

func StartHTTP(conf config.Http, controller *control.StateController) {
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
	router.HandleFunc("/api/v1/applications.list", serviceWrapper(&applicationv1.ListApplicationsRequest{}, appServer.ListApplications)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/applications.create", serviceWrapper(&applicationv1.CreateApplicationRequest{}, appServer.CreateApplication)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/applications.get", serviceWrapper(&applicationv1.GetApplicationRequest{}, appServer.GetApplication)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/applications.update", serviceWrapper(&applicationv1.UpdateApplicationRequest{}, appServer.UpdateApplication)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/applications.delete", serviceWrapper(&applicationv1.DeleteApplicationRequest{}, appServer.DeleteApplication)).Methods(http.MethodPost)

	// start listening for HTTP connections
	log.Infof("Started listening for HTTP connections on '%s'", conf.GetNetworkAddress())
	err = http.Serve(lis, router)
	if errors.Is(err, http.ErrServerClosed) {
		log.Info("HTTP server has been closed")
	} else {
		log.Errorf("Failed to serve HTTP server: %v", err)
		os.Exit(1)
	}

}

func serviceWrapper[ServiceRequest any, ServiceResponse any](sReq ServiceRequest, handler func(ctx context.Context, sReq ServiceRequest) (ServiceResponse, error)) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		// read request body
		bytes, err := io.ReadAll(req.Body)
		if err != nil {
			writeHttpError(res, err)
			return
		}

		// parse request body
		if len(bytes) > 0 {
			if err = json.Unmarshal(bytes, sReq); err != nil {
				writeHttpError(res, err)
				return
			}
		}

		// execute handler function
		ctx := context.Background()
		sRes, err := handler(ctx, sReq)
		if err != nil {
			writeHttpError(res, err)
			return
		}

		// format response
		body, err := json.Marshal(sRes)
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

	body := fmt.Sprintf(`{"message":"%s"}`, message)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(httpStatus)
	res.Write([]byte(body))
}
