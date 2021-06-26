package healthz

import (
	"net/http"
	"strconv"
)

func Start(port int) {
	http.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(nil)
	}))
	_ = http.ListenAndServe("0.0.0.0:"+strconv.Itoa(port), nil)
}
