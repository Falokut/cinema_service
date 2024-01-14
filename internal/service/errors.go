package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Falokut/grpc_errors"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrInternal        = errors.New("internal error")
	ErrInvalidArgument = errors.New("invalid input data")
)

var errorCodes = map[error]codes.Code{
	ErrNotFound:        codes.NotFound,
	ErrInvalidArgument: codes.InvalidArgument,
	ErrInternal:        codes.Internal,
}

type errorHandler struct {
	logger *logrus.Logger
}

func IsContextError(msg string) bool {
	switch msg {
	case context.Canceled.Error(), context.DeadlineExceeded.Error():
		return true
	default:
		if strings.Contains(msg, "context canceled") {
			return true
		}

		if strings.Contains(msg, context.DeadlineExceeded.Error()) {
			return true
		}

		return false
	}
}

func newErrorHandler(logger *logrus.Logger) errorHandler {
	return errorHandler{
		logger: logger,
	}
}

func (e *errorHandler) createErrorResponceWithSpan(span opentracing.Span, err error, developerMessage string) error {
	if err == nil {
		return nil
	}
	if IsContextError(developerMessage) {
		err = context.Canceled
		span.SetTag("grpc.status", codes.Canceled)
		ext.LogError(span, err)
	} else {
		span.SetTag("grpc.status", grpc_errors.GetGrpcCode(err))
		ext.LogError(span, err)
	}

	return e.createErrorResponce(err, developerMessage)
}

func (e *errorHandler) createErrorResponce(err error, developerMessage string) error {
	if errors.Is(err, context.Canceled) || IsContextError(developerMessage) {
		err = status.Error(codes.Canceled, developerMessage)
		e.logger.Error(err)
		return err
	}

	var msg string
	if len(developerMessage) == 0 {
		msg = err.Error()
	} else {
		msg = fmt.Sprintf("%s. error: %v", developerMessage, err)
	}

	err = status.Error(grpc_errors.GetGrpcCode(err), msg)
	e.logger.Error(err)
	return err
}

func init() {
	grpc_errors.RegisterErrors(errorCodes)
}
