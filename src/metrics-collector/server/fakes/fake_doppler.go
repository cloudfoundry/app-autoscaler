package fakes

import (
	"math/rand"
	. "metrics-collector/server"
	"mime/multipart"
	"net/http"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
)

const PATH_DOPPLER_CONTAINER_METRICS = "/apps/{appid}/containermetrics"
const FAKE_DOPPLER_ACCESS_TOKEN = "fake-access-token"
const FAKE_DOPPLER_URL = "wss://www.fake.com:4443"
const FAKE_APP_ID = "06ed2d79-e637-600d-8c6f-4b23b230c1d9"

func NewFakeDopplerHandler() http.Handler {
	r := mux.NewRouter()
	r.Methods("GET").Path(PATH_DOPPLER_CONTAINER_METRICS).HandlerFunc(handleContainerMetrics)
	return r
}

func handleContainerMetrics(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("Authorization")

	if token != TOKEN_TYPE_BEARER+" "+FAKE_DOPPLER_ACCESS_TOKEN {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	appId := mux.Vars(r)["appid"]
	if appId != FAKE_APP_ID {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	mp := multipart.NewWriter(w)
	defer mp.Close()

	w.Header().Set("Content-Type", `multipart/x-protobuf; boundary=`+mp.Boundary())

	origin := "fake-doppler"
	len := rand.Intn(11)

	for i := 0; i < len; i++ {
		index := int32(i)
		cpu := rand.Float64()
		memory := uint64(rand.Int63())
		disk := uint64(rand.Int63())
		cm := &events.ContainerMetric{
			ApplicationId: &appId,
			InstanceIndex: &index,
			CpuPercentage: &cpu,
			MemoryBytes:   &memory,
			DiskBytes:     &disk,
		}
		envelope := &events.Envelope{
			Origin:          &origin,
			EventType:       events.Envelope_ContainerMetric.Enum(),
			ContainerMetric: cm,
		}

		bytes, _ := proto.Marshal(envelope)
		partWriter, _ := mp.CreatePart(nil)
		partWriter.Write(bytes)
	}

}
