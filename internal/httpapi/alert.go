package httpapi

import (
	"log"
	"net/http"
	"time"
)

func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	if err := s.store.SyncAlerts(r.Context(), u.ID, time.Now().UTC()); err != nil {
		log.Printf("alert sync failed: %v", err)
	}

	alerts, err := s.store.ListAlerts(r.Context(), u.ID)
	if err != nil {
		serverError(w, err)
		return
	}

	resp := make([]alertResponse, 0, len(alerts))
	for _, a := range alerts {
		resp = append(resp, newAlertResponse(a))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) markAllRead(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if err := s.store.MarkAllRead(r.Context(), u.ID); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "All alerts marked as read."})
}
