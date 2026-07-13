package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"xjtu-housing-genius/internal/bed"
)

func main() {
	bed.MockMode = true

	port := "18730"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	if p, err := strconv.Atoi(port); err == nil {
		port = strconv.Itoa(p)
	}

	mux := http.NewServeMux()

	// ── Login ──
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Account  string `json:"account"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Account == "" || req.Password == "" {
			writeJSON(w, map[string]string{"error": "账号密码不能为空"})
			return
		}
		writeJSON(w, map[string]any{
			"success":     true,
			"studentCode": req.Account,
		})
	})

	// ── Session ──
	mux.HandleFunc("/api/session/check", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]bool{"alive": true})
	})

	mux.HandleFunc("/api/relogin", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})

	// ── MFA (no-op) ──
	mux.HandleFunc("/api/mfa/init", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"target": "138****0000", "type": "securephone"})
	})
	mux.HandleFunc("/api/mfa/send", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/mfa/verify", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"success": true})
	})

	// ── Bed APIs ──
	mux.HandleFunc("/api/bed/divideId", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(bed.MockDivideId())
	})

	mux.HandleFunc("/api/bed/tree", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(bed.MockBunkTree())
	})

	mux.HandleFunc("/api/bed/room-beds", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(bed.MockRoomBeds(r.URL.Query().Get("roomCode")))
	})

	mux.HandleFunc("/api/bed/check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(bed.MockCheckMyBed())
	})

	// ── Collection (real file-based) ──
	mux.HandleFunc("/api/bed/collection", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			bed.LoadCollection("mock-user")
			writeJSON(w, bed.GetCollection())
			return
		}
		var col bed.Collection
		json.NewDecoder(r.Body).Decode(&col)
		bed.SaveCollection(col, "mock-user")
		writeJSON(w, map[string]string{"status": "ok"})
	})

	// ── Grab (mock engine) ──
	var grabbing bool
	var grabLog []string
	var grabSuccess bool
	grabProgress := make(map[string]bed.BedProgress)

	mux.HandleFunc("/api/bed/grab/start", func(w http.ResponseWriter, r *http.Request) {
		col := bed.GetCollection()
		if len(col.Beds) == 0 {
			writeJSON(w, map[string]string{"error": "没有收藏"})
			return
		}
		grabbing = true
		grabSuccess = false
		grabLog = []string{}
		grabProgress = make(map[string]bed.BedProgress)
		var req struct {
			TotalConcurrency int `json:"totalConcurrency"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.TotalConcurrency == 0 {
			req.TotalConcurrency = col.TotalConcurrency
		}

		// 模拟抢床：3秒后成功
		for _, b := range col.Beds {
			grabProgress[b.BedCode] = bed.BedProgress{Total: 3, Done: 0}
		}
		go func() {
			for i := 1; i <= 3; i++ {
				time.Sleep(1 * time.Second)
				ts := time.Now().Format("15:04:05")
				for _, b := range col.Beds {
					p := grabProgress[b.BedCode]
					p.Done = i
					if i < 3 {
						p.Fail++
						grabLog = append(grabLog, fmt.Sprintf("[%s] %s 第%d轮: 已被抢", ts, b.BedName, i))
					} else {
						p.OK++
						grabSuccess = true
						grabLog = append(grabLog, fmt.Sprintf("[%s] ✅ %s: 抢床成功!", ts, b.BedName))
					}
					grabProgress[b.BedCode] = p
				}
			}
			grabbing = false
		}()

		writeJSON(w, map[string]string{"status": "started"})
	})

	mux.HandleFunc("/api/bed/grab/stop", func(w http.ResponseWriter, r *http.Request) {
		grabbing = false
		writeJSON(w, map[string]string{"status": "stopped"})
	})

	mux.HandleFunc("/api/bed/grab/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"running":    grabbing,
			"success":    grabSuccess,
			"successBed": "",
			"progress":   grabProgress,
			"log":        grabLog,
		})
	})

	// ── CORS ──
	handler := corsMiddleware(mux)

	addr := "127.0.0.1:" + port
	fmt.Printf("PORT=%s\n", port)
	fmt.Printf("Mock backend listening on http://%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Token")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
