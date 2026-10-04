package api

import (
	"net/http"

	"homebase/internal/sched"
	"homebase/internal/syncer"
)

func (s *server) handlePushKey(w http.ResponseWriter, r *http.Request) {
	key, err := s.Keys.PublicKey()
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, map[string]string{"publicKey": key})
}

func (s *server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	var req syncer.SubscribeInput
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.Sync.Subscribe(r.Context(), req); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Endpoint string `json:"endpoint"`
		ID       string `json:"id"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.Sync.Unsubscribe(r.Context(), req.Endpoint, req.ID); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleDevices(w http.ResponseWriter, r *http.Request) {
	list, err := s.Sync.Devices(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, list)
}

func (s *server) handlePushTest(w http.ResponseWriter, r *http.Request) {
	sent, failed := sched.Deliver(r.Context(), s.Sync, s.Sender, s.Log, sched.TestMessage())
	writeJSON(w, s.Log, http.StatusOK, map[string]int{"sent": sent, "failed": failed})
}

func (s *server) handleSaveReminders(w http.ResponseWriter, r *http.Request) {
	var req []syncer.ReminderInput
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	list, err := s.Sync.SaveReminders(r.Context(), req)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, list)
}

func (s *server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Sync.GetSettings(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, st)
}

func (s *server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var req syncer.Settings
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	st, err := s.Sync.SaveSettings(r.Context(), req)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, st)
}
