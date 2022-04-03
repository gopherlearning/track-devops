package handlers

import (
	"net/http"
	"strings"

	"github.com/gopherlearning/track-devops/internal/repositories"
)

// Handler ...
type ClassicHandler struct {
	s repositories.Repository
}

// NewHandler создаёт новый экземпляр обработчика запросов, привязанный к хранилищу
func NewHandler(s repositories.Repository) *ClassicHandler {
	return &ClassicHandler{s: s}
}

func (h *ClassicHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}
	// if r.Header.Get("Content-Type") != "text/plain" {
	// 	http.Error(w, "Only text/plain content are allowed!", http.StatusBadRequest)
	// 	return
	// }
	u := strings.Split(r.RequestURI, "/")
	// // c, ok := u[1]

	// if !ok || c != "update" {
	// 	w.WriteHeader(http.StatusNotFound)
	// 	return
	// }
	target := strings.Split(r.RemoteAddr, ":")[0]
	if err := h.s.Update(target, u[1], u[2], u[3]); err != nil {
		switch err {
		case repositories.ErrWrongMetricURL:
			w.WriteHeader(http.StatusNotFound)
		case repositories.ErrWrongMetricValue:
			w.WriteHeader(http.StatusBadRequest)
		case repositories.ErrWrongMetricType:
			w.WriteHeader(http.StatusNotImplemented)
		case repositories.ErrWrongValueInStorage:
			w.WriteHeader(http.StatusNotImplemented)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)

}
