package app

import (
	"log/slog"
	"net/http"
	"time"
)

//counterfeiter:generate . TimeWaster
type TimeWaster interface {
	Sleep(sleepTime time.Duration)
}

type Sleeper struct{}

var _ TimeWaster = Sleeper{}

func ResponseTimeTests(logger *slog.Logger, mux *http.ServeMux, timeWaster TimeWaster) {
	mux.HandleFunc("GET /responsetime/slow/{delayInMS}", func(w http.ResponseWriter, r *http.Request) {
		milliseconds, err := parsePositiveInt64(r, "delayInMS")
		if err != nil {
			respondWithErrorf(logger, w, http.StatusBadRequest, "%s", err.Error())
			return
		}
		duration := time.Duration(milliseconds) * time.Millisecond
		timeWaster.Sleep(duration)
		respondOk(logger, w, JSONResponse{"duration": duration.String()})
	})

	mux.HandleFunc("GET /responsetime/fast", func(w http.ResponseWriter, r *http.Request) {
		respondOk(logger, w, JSONResponse{"fast": true})
	})
}

func (Sleeper) Sleep(sleepTime time.Duration) {
	time.Sleep(sleepTime)
}
