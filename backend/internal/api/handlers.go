package api

import (
	"encoding/json"
	"log"
	"net/http"

	"xjtu-housing-genius/internal/auth"
	"xjtu-housing-genius/internal/bed"
	"xjtu-housing-genius/internal/config"
	"xjtu-housing-genius/internal/session"

	"github.com/go-resty/resty/v2"
)

type Server struct {
	client *resty.Client
	engine *bed.Engine
}

func NewServer() *Server {
	client := session.NewClient()
	return &Server{
		client: client,
		engine: bed.NewEngine(),
	}
}

// ── Login ──

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		Captcha  string `json:"captcha"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "参数错误"})
		return
	}
	if req.Account == "" || req.Password == "" {
		writeJSON(w, 400, map[string]string{"error": "账号密码不能为空"})
		return
	}

	st := session.Get()
	st.Account = req.Account
	st.Password = req.Password

	var client *resty.Client
	isRetry := req.Captcha != "" && s.client != nil
	log.Printf("[login] captcha=%q (len=%d) isRetry=%v", req.Captcha, len(req.Captcha), isRetry)
	if isRetry {
		client = s.client
	} else {
		client = session.NewClient()
	}

	err := auth.FullLoginWithCaptcha(client, req.Account, req.Password, req.Captcha)
	if err != nil {
		session.SaveCookiesFromHTTP(client.GetClient())
		s.client = client

		// 需要验证码
		if capErr, ok := err.(*auth.CaptchaNeededError); ok {
			resp := map[string]any{
				"captcha_required": true,
			}
			if capErr.Message != "" {
				resp["error"] = capErr.Message
			}
			writeJSON(w, 200, resp)
			return
		}

		// 需要 MFA
		if mfaErr, ok := err.(*auth.MFANeededError); ok {
			writeJSON(w, 200, map[string]any{
				"mfa_required": true,
				"state":        mfaErr.State,
			})
			return
		}

		// 需要选择账户身份
		if acErr, ok := err.(*auth.AccountChoiceNeededError); ok {
			writeJSON(w, 200, map[string]any{
				"account_choice_required": true,
				"choices":                 acErr.Choices,
			})
			return
		}

		// 登录成功但无权访问（如选宿时间未到）—— 仍保存 session 用于代理
		if adErr, ok := err.(*auth.AccessDeniedError); ok {
			session.SaveCookiesFromHTTP(client.GetClient())
			s.client = client
			writeJSON(w, 200, map[string]any{
				"success":       false,
				"access_denied": true,
				"error":         adErr.Message,
			})
			return
		}

		writeJSON(w, 200, map[string]string{"error": err.Error()})
		return
	}

	session.SaveCookiesFromHTTP(client.GetClient())
	s.client = client
	bed.LoadCollection(session.Get().StudentCode)

	writeJSON(w, 200, map[string]any{
		"success":     true,
		"studentCode": session.Get().StudentCode,
	})
}

// ── Account Choice ──

func (s *Server) HandleChooseAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountType string `json:"accountType"` // "undergraduate" or "postgraduate"
	}
	json.NewDecoder(r.Body).Decode(&req)

	client := session.NewClient()
	if err := auth.ChooseAccount(client, req.AccountType); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	session.SaveCookies(client)
	client.SetHeader("Token", session.Get().Token)
	s.client = client

	writeJSON(w, 200, map[string]any{
		"success":     true,
		"studentCode": session.Get().StudentCode,
	})
}

// ── Captcha ──

func (s *Server) HandleCaptchaImage(w http.ResponseWriter, r *http.Request) {
	img, err := auth.GetCaptchaImage(s.client)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(img)
}

// ── MFA ──

func (s *Server) HandleMFAInit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string `json:"method"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result, err := auth.InitMFA(s.client, req.Method)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) HandleMFASend(w http.ResponseWriter, r *http.Request) {
	if err := auth.SendMFACode(s.client); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) HandleMFAVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// 用 s.client（有 CAS cookies）验证 MFA 码
	if err := auth.VerifyMFACode(s.client, req.Code); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	// 保存 MFA 验证后的 cookies 到 session state
	session.SaveCookies(s.client)

	// 照抄 Course-Genius：创建新 client（从 session 恢复 cookies），完成登录
	client := session.NewClient()

	// Safety Verify flow
	if auth.IsSafetyVerifyFlow() {
		if err := auth.FinishSafetyVerifyLogin(client); err != nil {
			writeJSON(w, 500, map[string]string{"error": "二次认证失败: " + err.Error()})
			return
		}
		auth.ClearMFA()
		session.SaveCookies(client)
		client.SetHeader("Token", session.Get().Token)
		s.client = client
		writeJSON(w, 200, map[string]any{
			"success":     true,
			"studentCode": session.Get().StudentCode,
		})
		return
	}

	if err := auth.CompleteLoginAfterMFA(client); err != nil {
		if adErr, ok := err.(*auth.AccessDeniedError); ok {
			writeJSON(w, 200, map[string]any{
				"success":       false,
				"access_denied": true,
				"error":         adErr.Message,
			})
			return
		}
		writeJSON(w, 500, map[string]string{"error": "MFA验证通过但登录失败: " + err.Error()})
		return
	}

	auth.ClearMFA()
	session.SaveCookies(client)
	client.SetHeader("Token", session.Get().Token)
	s.client = client

	writeJSON(w, 200, map[string]any{
		"success":     true,
		"studentCode": session.Get().StudentCode,
	})
}

// ── Session ──

func (s *Server) HandleSessionCheck(w http.ResponseWriter, r *http.Request) {
	alive := auth.IsSessionAlive(s.client)
	writeJSON(w, 200, map[string]bool{"alive": alive})
}

func (s *Server) HandleRelogin(w http.ResponseWriter, r *http.Request) {
	client := session.NewClient()
	if err := auth.ReloginIfNeeded(client); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	session.SaveCookies(client)
	s.client = client
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// ── Config ──

func (s *Server) HandleConfigGet(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	writeJSON(w, 200, cfg)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ── Bed ──

func (s *Server) HandleBedDivideId(w http.ResponseWriter, r *http.Request) {
	personsn := r.URL.Query().Get("personsn")
	if personsn == "" {
		personsn = session.Get().StudentCode
	}
	body, err := bed.ProxyGet(s.client, "/appdm/freshman/resident/getDivideCountDown",
		map[string]string{"personsn": personsn, "status": "PC"}, session.Get().Token)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) HandleBedTree(w http.ResponseWriter, r *http.Request) {
	divideId := r.URL.Query().Get("divideId")
	body, err := bed.ProxyGet(s.client, "/appdm/freshman/divide/getBunkTreeByDivideId",
		map[string]string{"modelId": "dm", "type": "ROOM", "divideId": divideId}, session.Get().Token)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	// 真实API返回 {places: [...]}，提取places数组
	var wrapper struct {
		Code   int  `json:"code"`
		Places []any `json:"places"`
	}
	w.Header().Set("Content-Type", "application/json")
	if json.Unmarshal(body, &wrapper) == nil && wrapper.Places != nil {
		json.NewEncoder(w).Encode(wrapper.Places)
	} else {
		w.Write(body)
	}
}

func (s *Server) HandleBedRoomBeds(w http.ResponseWriter, r *http.Request) {
	divideId := r.URL.Query().Get("divideId")
	roomCode := r.URL.Query().Get("roomCode")
	body, err := bed.ProxyPost(s.client, "/appdm/freshman/divide/getBedInfoByDivideId",
		map[string]string{"modelId": "dm", "roomCode": roomCode, "divideId": divideId}, "query", session.Get().Token)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Write(body)
}

func (s *Server) HandleBedCheck(w http.ResponseWriter, r *http.Request) {
	personsn := r.URL.Query().Get("personsn")
	divideId := r.URL.Query().Get("divideId")
	body, err := bed.ProxyGet(s.client, "/appdm/freshman/bunk/checkMyBed",
		map[string]string{"personsn": personsn, "divideId": divideId}, session.Get().Token)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) HandleBedGrabStart(w http.ResponseWriter, r *http.Request) {
	personsn := r.URL.Query().Get("personsn")
	if personsn == "" {
		personsn = session.Get().StudentCode
	}
	divideId := r.URL.Query().Get("divideId")

	s.engine.SetClient(s.client, personsn, divideId)

	var req struct {
		TotalConcurrency int `json:"totalConcurrency"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.TotalConcurrency == 0 {
		req.TotalConcurrency = bed.GetCollection().TotalConcurrency
	}

	if err := s.engine.Start(req.TotalConcurrency); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "started"})
}

func (s *Server) HandleBedCollectionGet(w http.ResponseWriter, r *http.Request) {
	studentCode := session.Get().StudentCode
	if studentCode == "" {
		studentCode = r.URL.Query().Get("studentCode")
	}
	bed.LoadCollection(studentCode)
	writeJSON(w, 200, bed.GetCollection())
}

func (s *Server) HandleBedCollectionSave(w http.ResponseWriter, r *http.Request) {
	var col bed.Collection
	if err := json.NewDecoder(r.Body).Decode(&col); err != nil {
		writeJSON(w, 400, map[string]string{"error": "参数错误"})
		return
	}
	for _, b := range col.Beds {
		if err := bed.ValidatePriority(b.Priority); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
	}
	studentCode := session.Get().StudentCode
	if err := bed.SaveCollection(col, studentCode); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// ── 服务器端收藏同步 (原网页 saveBed / getBedCollectList / deleteBedCollect) ──

func (s *Server) HandleBedCollectSyncSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Personsn     string `json:"personsn"`
		BedPlaceCode string `json:"bedPlaceCode"`
		DivideId     string `json:"divideId"`
		BeddingInfo  string `json:"beddingInfo"`
		BedCodes     string `json:"bedCodes"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Personsn == "" {
		req.Personsn = session.Get().StudentCode
	}
	if req.DivideId == "" {
		req.DivideId = r.URL.Query().Get("divideId")
	}
	log.Printf("[collect] saveBed: personsn=%s divideId=%s bedPlaceCode=%s bedCodes=%s",
		req.Personsn, req.DivideId, req.BedPlaceCode, req.BedCodes)

	body, err := bed.ProxyPostJSON(s.client, "/appdm/freshman/collect/saveBed",
		map[string]interface{}{
			"personsn":     req.Personsn,
			"bedPlaceCode": req.BedPlaceCode,
			"divideId":     req.DivideId,
			"beddingInfo":  req.BeddingInfo,
			"bedCodes":     req.BedCodes,
		}, session.Get().Token)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) HandleBedCollectSyncList(w http.ResponseWriter, r *http.Request) {
	personsn := r.URL.Query().Get("personsn")
	if personsn == "" {
		personsn = session.Get().StudentCode
	}
	divideId := r.URL.Query().Get("divideId")
	modelId := r.URL.Query().Get("modelId")
	if modelId == "" {
		modelId = "dm"
	}
	log.Printf("[collect] getBedCollectList: personsn=%s divideId=%s", personsn, divideId)

	body, err := bed.ProxyPost(s.client, "/appdm/freshman/collect/getBedCollectList",
		map[string]string{"personsn": personsn, "divideId": divideId, "modelId": modelId},
		"query", session.Get().Token)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) HandleBedCollectSyncDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Id      string `json:"id"`
		BedCode string `json:"bedCode"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	log.Printf("[collect] deleteBedCollect: id=%s bedCode=%s", req.Id, req.BedCode)

	body, err := bed.ProxyPost(s.client, "/appdm/freshman/collect/deleteBedCollect",
		map[string]string{"id": req.Id, "bedCode": req.BedCode},
		"form", session.Get().Token)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) HandleBedGrabStop(w http.ResponseWriter, r *http.Request) {
	s.engine.Stop()
	writeJSON(w, 200, map[string]string{"status": "stopped"})
}

func (s *Server) HandleBedGrabStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.engine.Status())
}

func (s *Server) HandleBedRoomAssign(w http.ResponseWriter, r *http.Request) {
	divideId := r.URL.Query().Get("divideId")
	roomCodes := r.URL.Query().Get("roomCodes")
	body, err := bed.ProxyGet(s.client, "/appdm/freshman/bunk/queryAssignBedsByRoom",
		map[string]string{"divideId": divideId, "roomCodes": roomCodes}, session.Get().Token)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
