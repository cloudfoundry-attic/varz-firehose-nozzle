package server
import (
	"github.com/cloudfoundry/noaa/events"
	"net/http"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/auth"
	"fmt"
	"encoding/json"
)

type VarzServer struct {
	port int
	user string
	pass string
}

func New(emitter Emitter, varzPort int, varzUser string, varzPass string) *VarzServer {
	return &VarzServer{
		port: varzPort,
		user: varzUser,
		pass: varzPass,
	}
}


func (v *VarzServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

}

func (v *VarzServer) StartMonitoringEndpoints() error {
	mux := http.NewServeMux()
	auth := auth.NewBasicAuth("Realm", []string{v.user, v.pass})

	mux.HandleFunc("/varz", auth.Wrap(varzHandlerFor(v)))

	err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", v.port), mux)
	return err
}

func varzHandlerFor(v *VarzServer) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {

		message, err := v.emitter.Emit()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		}

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
}