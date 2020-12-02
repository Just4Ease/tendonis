package tendonis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/Just4Ease/axon"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"io"
	"log"
	"net/http"
)

type (
	Server struct {
		eventStream axon.EventStore
		transports  []graphql.Transport
		exec        *executor.Executor
	}
)

func New(eventStream axon.EventStore, es graphql.ExecutableSchema) *Server {
	return &Server{
		eventStream: eventStream,
		exec:        executor.New(es),
	}
}

func NewDefaultServer(eventStream axon.EventStore, es graphql.ExecutableSchema) *Server {
	srv := New(eventStream, es)
	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	return srv
}

func (s *Server) SetErrorPresenter(f graphql.ErrorPresenterFunc) {
	s.exec.SetErrorPresenter(f)
}

func (s *Server) SetRecoverFunc(f graphql.RecoverFunc) {
	s.exec.SetRecoverFunc(f)
}

func (s *Server) SetQueryCache(cache graphql.Cache) {
	s.exec.SetQueryCache(cache)
}

func (s *Server) Use(extension graphql.HandlerExtension) {
	s.exec.Use(extension)
}

// AroundFields is a convenience method for creating an extension that only implements field middleware
func (s *Server) AroundFields(f graphql.FieldMiddleware) {
	s.exec.AroundFields(f)
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (s *Server) AroundOperations(f graphql.OperationMiddleware) {
	s.exec.AroundOperations(f)
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (s *Server) AroundResponses(f graphql.ResponseMiddleware) {
	s.exec.AroundResponses(f)
}

func (s *Server) getTransport(r *http.Request) graphql.Transport {
	for _, t := range s.transports {
		if t.Supports(r) {
			return t
		}
	}
	return nil
}

func (s *Server) Serve() {
	topic := fmt.Sprintf("%s::%s", "tendonis", s.eventStream.GetServiceName())
	err := s.eventStream.Subscribe(topic, func(event axon.Event) {
		var request axon.RequestPayload
		if err := json.Unmarshal(event.Data(), &request); err != nil {
			// TODO: Log this error.
			log.Printf("failed to parse ")
			return
		}

		err := s.eventStream.Reply(request.GetReplyAddress(), func() ([]byte, error) {
			ctx := graphql.StartOperationTrace(context.Background())
			defer func() {
				if err := recover(); err != nil {
					err := s.exec.PresentRecoveredError(ctx, err)
					resp := &graphql.Response{Errors: []*gqlerror.Error{err}} // TODO: see what to do with this guy.
					_, _ = json.Marshal(resp)
				}
			}()

			var params *graphql.RawParams
			start := graphql.Now()
			reader := bytes.NewReader(request.GetPayload())
			if err := jsonDecode(reader, &params); err != nil {
				//writeJsonErrorf(w, "json body could not be decoded: "+err.Error())
				log.Print(err, " Error decoding request payload")
				return nil, err
			}
			params.ReadTime = graphql.TraceTiming{
				Start: start,
				End:   graphql.Now(),
			}

			rc, err := s.exec.CreateOperationContext(ctx, params)
			if err != nil {
				//w.WriteHeader(statusFor(err))
				//resp := s.exec.DispatchError(graphql.WithOperationContext(blankCTX, rc), err)
				//writeJson(w, resp)
				return nil, err
			}
			responses, ctxProcessed := s.exec.DispatchOperation(graphql.WithOperationContext(ctx, rc), rc)
			rsp := responses(ctxProcessed)
			return json.Marshal(rsp)
		})
		log.Printf("failed to reply to request on: %s with the following error: %v", topic, err)
	})

	if err != nil {
		panic(err)
	}
}

func sendError(w http.ResponseWriter, code int, errors ...*gqlerror.Error) {
	w.WriteHeader(code)
	b, err := json.Marshal(&graphql.Response{Errors: errors})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(b)
}

type OperationFunc func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler

func (r OperationFunc) ExtensionName() string {
	return "InlineOperationFunc"
}

func (r OperationFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("OperationFunc can not be nil")
	}
	return nil
}

func (r OperationFunc) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return r(ctx, next)
}

type ResponseFunc func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response

func (r ResponseFunc) ExtensionName() string {
	return "InlineResponseFunc"
}

func (r ResponseFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("ResponseFunc can not be nil")
	}
	return nil
}

func (r ResponseFunc) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	return r(ctx, next)
}

type FieldFunc func(ctx context.Context, next graphql.Resolver) (res interface{}, err error)

func (f FieldFunc) ExtensionName() string {
	return "InlineFieldFunc"
}

func (f FieldFunc) Validate(schema graphql.ExecutableSchema) error {
	if f == nil {
		return fmt.Errorf("FieldFunc can not be nil")
	}
	return nil
}

func (f FieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	return f(ctx, next)
}

func jsonDecode(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(val)
}
