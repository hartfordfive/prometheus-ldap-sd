package handler

import (
	//"encoding/json"
	"fmt"
	"net/http"

	//"strings"

	//"github.com/gorilla/mux"
	"github.com/hartfordfive/prometheus-ldap-sd-server/config"
	//"github.com/hartfordfive/prometheus-ldap-sd-server/lib"
	//"github.com/hartfordfive/prometheus-ldap-sd-server/logger"
	"github.com/hartfordfive/prometheus-ldap-sd-server/store"
	// "github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/hartfordfive/prometheus-ldap-sd-server/logger"
)

// var (
// 	metricTargetGroupUpdates = promauto.NewCounter(prometheus.CounterOpts{
// 		Name: "ldap_sd_target_group_updates",
// 		Help: "Number of times a target group has been updated.",
// 	})
// )

var HealthHandler = func(w http.ResponseWriter, req *http.Request) {
	/*
		TO COMPLETE:
		This handler should only return OK if the underlying datastore is ready to accept connections
	*/
	fmt.Fprint(w, "OK")
}

var ShowTargetsHandler = func(w http.ResponseWriter, req *http.Request) {
	//logger.Logger.Debug("Target listing requested", zap.String("remote_addr", req.RemoteAddr))
	w.Header().Set("Content-Type", "application/json")
	targetGroup := req.URL.Query().Get("targetGroup")
	dataStore := store.StoreInstance
	res, err := dataStore.Serialize(targetGroup)
	if err != nil {
		fmt.Fprint(w, "[]\n")
		return
	}
	fmt.Fprintf(w, "%s\n", res)
}

var ShowConfigHandler = func(w http.ResponseWriter, req *http.Request) {
	logger.Logger.Info("Debug config requested")
	w.Header().Set("Content-Type", "text/yaml")
	printCnf, err := config.GlobalConfig.Serialize()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s\n", printCnf)
}
