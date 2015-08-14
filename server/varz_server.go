package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/emitter"
)

const varzNozzleRealm = "Varz Nozzle"

type VarzServer struct {
	port     int
	username string
	password string
	emitter  Emitter
}

type Emitter interface {
	Emit(job string, index int) *emitter.VarzMessage
}

type credentials struct {
	username string
	password string
}

func New(emitter Emitter, varzPort int, varzUser string, varzPass string) *VarzServer {
	return &VarzServer{
		port:     varzPort,
		username: varzUser,
		password: varzPass,
		emitter:  emitter,
	}
}

func (v *VarzServer) Start() error {
	http.Handle("/varz", v)
	err := http.ListenAndServe(fmt.Sprintf(":%d", v.port), nil)
	return err
}

func (v *VarzServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !v.validCredentials(req) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, varzNozzleRealm))
		fmt.Fprintf(w, "%d Unauthorized", http.StatusUnauthorized)
		return
	}

	message := v.emitter.Emit("job", 0)
	json, err := json.Marshal(message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(json)
}

func (v *VarzServer) validCredentials(req *http.Request) bool {
	creds, err := extractCredentials(req)
	if err != nil {
		return false
	}
	return creds.username == v.username && creds.password == v.password
}

func extractCredentials(req *http.Request) (*credentials, error) {
	authorizationHeader := req.Header.Get("Authorization")
	basicAuthParts := strings.Split(authorizationHeader, " ")
	if len(basicAuthParts) != 2 || basicAuthParts[0] != "Basic" {
		return nil, fmt.Errorf("Malformed authorization header: %s", authorizationHeader)
	}

	decodedUserAndPassword, err := base64.StdEncoding.DecodeString(basicAuthParts[1])
	if err != nil {
		return nil, err
	}

	userAndPassword := strings.Split(string(decodedUserAndPassword), ":")
	if len(userAndPassword) != 2 {
		return nil, fmt.Errorf("Malformed authorization header: %s", authorizationHeader)
	}

	return &credentials{username: userAndPassword[0], password: userAndPassword[1]}, nil
}
