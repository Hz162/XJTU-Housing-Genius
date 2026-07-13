package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter() *chi.Mux {
	s := NewServer()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api", func(r chi.Router) {
		r.Post("/login", s.HandleLogin)
		r.Post("/account/choose", s.HandleChooseAccount)
		r.Get("/captcha", s.HandleCaptchaImage)
		r.Get("/session/check", s.HandleSessionCheck)
		r.Post("/relogin", s.HandleRelogin)
		r.Get("/config", s.HandleConfigGet)

		r.Route("/mfa", func(r chi.Router) {
			r.Post("/init", s.HandleMFAInit)
			r.Post("/send", s.HandleMFASend)
			r.Post("/verify", s.HandleMFAVerify)
		})

		r.Route("/bed", func(r chi.Router) {
			r.Get("/divideId", s.HandleBedDivideId)
			r.Get("/tree", s.HandleBedTree)
			r.Get("/room-beds", s.HandleBedRoomBeds)
			r.Get("/check", s.HandleBedCheck)
			r.Get("/collection", s.HandleBedCollectionGet)
			r.Post("/collection", s.HandleBedCollectionSave)
			r.Get("/collect-list", s.HandleBedCollectSyncList)
			r.Post("/collect-save", s.HandleBedCollectSyncSave)
			r.Post("/collect-delete", s.HandleBedCollectSyncDelete)
			r.Get("/grab/status", s.HandleBedGrabStatus)
			r.Post("/grab/start", s.HandleBedGrabStart)
			r.Post("/grab/stop", s.HandleBedGrabStop)
			r.Get("/room-assign", s.HandleBedRoomAssign)
		})
	})

	return r
}
