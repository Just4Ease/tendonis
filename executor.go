package tendonis

//type Stream struct {
//	rootPath    string
//	eventStream axon.EventStore
//}
//
//func NewStreamTransport(eventStream axon.EventStore) *Stream {
//
//	s := &Stream{
//		rootPath:    "x-tendonis/",
//		eventStream: eventStream,
//	}
//
//	var _ graphql.Transport = s
//	return s
//}

//func (s Stream) Supports(r *http.Request) bool {
//	if r.Header.Get("Upgrade") != "" {
//		return false
//	}
//
//	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
//	if err != nil {
//		return false
//	}
//
//	return r.Method == "POST" && mediaType == "application/json"
//}

//func (s Stream) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
//	w.Header().Set("Content-Type", "application/json")
//
//	var params *graphql.RawParams
//	start := graphql.Now()
//	if err := jsonDecode(r.Body, &params); err != nil {
//		transport.writeJsonErrorf(w, "json body could not be decoded: "+err.Error())
//		return
//	}
//	params.ReadTime = graphql.TraceTiming{
//		Start: start,
//		End:   graphql.Now(),
//	}
//
//	rc, err := exec.CreateOperationContext(r.Context(), params)
//	if err != nil {
//		w.WriteHeader(statusFor(err))
//		resp := exec.DispatchError(graphql.WithOperationContext(r.Context(), rc), err)
//		writeJson(w, resp)
//		return
//	}
//	ctx := graphql.WithOperationContext(r.Context(), rc)
//	responses, ctx := exec.DispatchOperation(ctx, rc)
//	writeJson(w, responses(ctx))
//}
