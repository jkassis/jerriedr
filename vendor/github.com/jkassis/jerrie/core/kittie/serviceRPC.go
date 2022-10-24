package kittie

import (
	"bufio"
	"bytes"
	context "context"
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/google/uuid"
	"github.com/jkassis/jerrie/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrJSONParseFailed is what it says
	ErrJSONParseFailed = errors.New("JSON parsing failed")
	// ErrRouteNotSpecified is what it says
	ErrRouteNotSpecified = errors.New("route not specified")
	// ErrRouteNotFound is what it says
	ErrRouteNotFound = errors.New("route not found")
)

// RPCReq is a single RPC
type RPCReq struct {
	UUID uuid.UUID
	Fn   string
	Body *json.RawMessage
}

var (
	binaryencflag = byte('b')
	newline       = byte('\n')
	nilBytes      = []byte{}
)

// RPCReqNew returns a new RPCReq
func RPCReqNew(fn string, body []byte) *RPCReq {
	UUID, _ := uuid.NewRandom()
	raw := json.RawMessage(body)
	return &RPCReq{
		UUID: UUID,
		Fn:   fn,
		Body: &raw,
	}
}

// MarshalBinary dehydrates to []byte
func (j *RPCReq) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte(binaryencflag)
	b.Write([]byte(j.UUID[:]))
	b.Write([]byte(j.Fn))
	b.WriteByte(newline)
	b.Write([]byte(*j.Body))
	return b.Bytes(), nil
}

// UnmarshalBinary hydrates from []byte
func (j *RPCReq) UnmarshalBinary(bs []byte) error {
	// UUID
	j.UUID.UnmarshalBinary(bs[1:17])

	// Fn
	bb := bufio.NewReader(bytes.NewBuffer(bs[17:]))
	FN, _, err := bb.ReadLine()
	if err != nil {
		return err
	}
	j.Fn = string(FN)

	body, err := ioutil.ReadAll(bb)
	if err != nil {
		return err
	}
	if body == nil || len(body) == 0 {
		j.Body = nil

	} else {
		raw := json.RawMessage(body)
		j.Body = &raw
	}

	return nil
}

var rpcStatusCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "rpcs_handled",
	Help: "The total number of rpcs processed by status code",
}, []string{"fn", "status"})

// ServiceRPC handles RPC requests
type ServiceRPC struct {
	ServiceRouter
}

// Init is a no-op here, but allows shadowing for types that embed this.
func (service *ServiceRPC) Init() {
	service.Routes = map[string]Handler{
		"api/{tail:.+}": service.API,
	}
}

// API does one RPC
func (service *ServiceRPC) API(ctx context.Context, req []byte) (res []byte, err error) {
	// Look for empty requests
	if len(req) == 0 {
		err = errors.New("request was empty")
		core.Log.Error(err)
		return []byte(err.Error()), err
	}

	// Unmarshal
	var jsonRPC RPCReq
	if req[0] == 'b' {
		// it's binary encoded
		if err = jsonRPC.UnmarshalBinary(req); err != nil {
			return nil, err
		}
	} else {
		// it's json encoded
		// if err = json.NewDecoder(r).Decode(&jsonRPC); err != nil {
		if err = json.Unmarshal(req, &jsonRPC); err != nil {
			rpcStatusCounter.WithLabelValues("serviceRPC.API", "err").Inc()
			log.Error(err)
			return nil, ErrJSONParseFailed
		}
	}

	// Did we get a function name??
	if jsonRPC.Fn == "" {
		rpcStatusCounter.WithLabelValues("serviceRPC.API", "err").Inc()
		core.Log.Error(ErrRouteNotSpecified.Error() + " in \n" + string(req))
		return nil, ErrRouteNotSpecified
	}
	core.Log.Tracef("ServiceRPC: handling %s", jsonRPC.Fn)

	// Do we have a handler?
	handler := service.Routes[jsonRPC.Fn]
	if handler == nil {
		rpcStatusCounter.WithLabelValues(jsonRPC.Fn, "err").Inc()
		routes := ""
		for k := range service.Routes {
			routes += k + ","
		}
		core.Log.Error(ErrRouteNotFound.Error() + " for " + jsonRPC.Fn + " have these routes: " + routes)
		return []byte(ErrRouteNotFound.Error()), err
	}

	// Got a handler. run it.
	var body []byte
	if jsonRPC.Body == nil {
		body = nilBytes
	} else {
		body = []byte(*jsonRPC.Body)
	}

	res, err = handler(ctx, body)
	if err != nil {
		rpcStatusCounter.WithLabelValues(jsonRPC.Fn, "err").Inc()
		core.Log.Error(err)
		return []byte(err.Error()), err
	}

	// success
	rpcStatusCounter.WithLabelValues(jsonRPC.Fn, "ok").Inc()
	return res, err
}

// Log dumps a jsonRPC to the logs
func (service *ServiceRPC) Log(jsonRPC *RPCReq) {
	if prettyJSON, err := json.MarshalIndent(jsonRPC.Body, "", "\t"); err == nil {
		core.Log.Trace("Got this API Request:")
		core.Log.Trace(string(prettyJSON))
	} else {
		core.Log.Error("error parsing api jsonRPC: ", err)
	}
}
