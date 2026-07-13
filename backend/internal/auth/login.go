package auth

import (
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"net/url"
	"strings"

	"xjtu-housing-genius/internal/session"

	"github.com/go-resty/resty/v2"
)

const (
	housingBaseURL    = "http://housing.xjtu.edu.cn"
	housingAPIURL     = "http://housing2021.xjtu.edu.cn"
	casBaseURL        = "https://login.xjtu.edu.cn"
	oauthAuthorizeURL = "https://org.xjtu.edu.cn/openplatform/oauth/authorize"
)

const (
	oauthAppID       = "1676"
	oauthRedirectURI = "http://housing2021.xjtu.edu.cn/appsys/xjtuCASLogin/authenRedirectByDmwebV2"
	oauthScope       = "user_info"
	oauthState       = "home"
)

var failCount int

// stored state for captcha retry
var captchaCASURL string
var captchaExecution string
var captchaMFAState string
var captchaFpID string

// stored state for account choice
var accountChoiceExecution string
var accountChoices []map[string]string

func ResetFailCount() {
	failCount = 0
	captchaCASURL = ""
	captchaExecution = ""
	captchaMFAState = ""
	captchaFpID = ""
}
func IsCaptchaRequired() bool { return failCount >= 3 }

// isCaptchaPage 检查 CAS 是否主动显示验证码
func isCaptchaPage(body []byte) bool {
	s := string(body)
	if !strings.Contains(s, "fm1") || !strings.Contains(s, "execution") {
		return false
	}
	idx := strings.Index(s, "captcha.jpg")
	if idx < 0 {
		return false
	}
	start := idx - 400
	if start < 0 {
		start = 0
	}
	if strings.Contains(s[start:idx], "display:none") {
		return false
	}
	return true
}

// ── session alive check ──

func IsSessionAlive(client *resty.Client) bool {
	// 用 Token header 调用 housing 配置接口验证 session
	token := session.Get().Token
	if token == "" {
		return false
	}
	resp, err := client.R().
		SetHeader("Token", token).
		Post(housingAPIURL + "/appsys/sys/config/listAll")
	if err != nil {
		return false
	}
	if resp.StatusCode() == http.StatusOK {
		var j struct{ Code int `json:"code"` }
		if json.Unmarshal(resp.Body(), &j) == nil {
			return j.Code == 0
		}
	}
	return false
}

// ── Relogin ──

func ReloginIfNeeded(client *resty.Client) error {
	if IsSessionAlive(client) {
		return nil
	}
	s := session.Get()
	if err := FullLogin(client, s.Account, s.Password); err != nil {
		return err
	}
	if session.Get().Token != "" {
		client.SetHeader("Token", session.Get().Token)
	}
	ResetFailCount()
	session.SaveCookies(client)
	return nil
}

// ── full OAuth login (照抄 Course-Genius 状态机模式) ──

func FullLogin(client *resty.Client, account, password string) error {
	return FullLoginWithCaptcha(client, account, password, "")
}

func FullLoginWithCaptcha(client *resty.Client, account, password, captcha string) error {
	httpClient := client.GetClient()

	var casURL, execution, mfaState, fpID string
	var encPwd string

	stdlog.Printf("[captcha] FullLogin: captcha=%q (len=%d) captchaCASURL set=%v", captcha, len(captcha), captchaCASURL != "")

	// ── 验证码重试分支：复用已保存的 CAS session/execution ──
	if captcha != "" && captchaCASURL != "" {
		stdlog.Println("[captcha] retry: reusing stored execution")
		casURL = captchaCASURL
		execution = captchaExecution
		fpID = captchaFpID
		encPwd, _ = EncryptPassword(password)

		if mfaNeed, mfaState2, err := detectMFA(client, account, encPwd, fpID); err == nil {
			mfaState = mfaState2
			if mfaNeed {
				currentMFA = &MFAInfo{State: mfaState}
				return &MFANeededError{State: mfaState, Reason: "需要MFA验证"}
			}
		}
	} else {
		// ── 全新登录 ──
		fpID, _ = GetFingerprint()
		session.SetFpVisitorID(fpID)

		// Step 1: GET OAuth URL → CAS redirect
		oauthURL := buildOAuthURL()
		resp, err := httpClient.Get(oauthURL)
		if err != nil {
			return fmt.Errorf("访问认证平台失败: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		casURL = resp.Request.URL.String()
		execution = extractExecution(body)
		stdlog.Printf("[oauth] Step 1: CAS URL=%s, execution=%s", casURL[:min(80, len(casURL))], execution[:min(40, len(execution))])

		if !strings.Contains(casURL, "login.xjtu.edu.cn") {
			if strings.Contains(casURL, "housing") {
				stdlog.Println("[oauth] 已有有效会话")
				return nil
			}
			return fmt.Errorf("未跳转到CAS登录页: %s", casURL)
		}

		if execution == "" {
			return fmt.Errorf("无法获取CAS登录凭证(execution)")
		}

		// Step 2: GET public key
		if pubResp, err := httpClient.Get(casBaseURL + "/cas/jwt/publicKey"); err == nil {
			pubBody, _ := io.ReadAll(pubResp.Body)
			pubResp.Body.Close()
			SetPubKey(string(pubBody))
		}

		encPwd, _ = EncryptPassword(password)

		// Step 3: MFA detect
		if mfaNeed, mfaState2, err := detectMFA(client, account, encPwd, fpID); err == nil {
			mfaState = mfaState2
			if mfaNeed {
				currentMFA = &MFAInfo{State: mfaState}
				return &MFANeededError{State: mfaState, Reason: "需要MFA验证"}
			}
		}

		// Store CAS session for captcha retry
		stdlog.Println("[captcha] storing CAS session")
		captchaCASURL = casURL
		captchaExecution = execution
		captchaMFAState = mfaState
		captchaFpID = fpID
	}

	// captcha pre-check
	if failCount >= 3 && captcha == "" {
		stdlog.Printf("[captcha] pre-check: failCount=%d, returning captcha_required", failCount)
		return &CaptchaNeededError{}
	}

	// Step 4: POST CAS login (照抄 postCASRaw)
	return postCASRaw(httpClient, casURL, account, encPwd, execution, mfaState, fpID, captcha, "")
}

// postCASRaw — 照抄 Course-Genius
func postCASRaw(httpClient *http.Client, casURL, account, encPwd, execution, mfaState, fpID, captcha, trustAgent string) error {
	form := url.Values{
		"username":    {account},
		"password":    {encPwd},
		"captcha":     {captcha},
		"currentMenu": {"1"},
		"failN":       {fmt.Sprintf("%d", failCount)},
		"mfaState":    {mfaState},
		"execution":   {execution},
		"_eventId":    {"submit"},
		"geolocation": {""},
		"fpVisitorId": {fpID},
		"trustAgent":  {trustAgent},
		"submit1":     {"Login1"},
	}

	req, err := http.NewRequest("POST", casURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建CAS请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CAS登录请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		failCount++
		return fmt.Errorf("登录失败：用户名或密码错误")
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	isCASPage := strings.Contains(string(body), "fm1") && strings.Contains(string(body), "execution")
	alertMsg := extractAlertMessage(body)

	if isCASPage {
		newExec := extractExecution(body)
		if newExec != "" {
			captchaExecution = newExec
			captchaCASURL = resp.Request.URL.String()
		}
	}

	captchaLikely := failCount >= 3 || captcha != "" || alertMsg == ""

	if isCASPage && captchaLikely {
		failCount++
		stdlog.Printf("[captcha] captcha required, alert=%q, failCount=%d", alertMsg, failCount)
		reason := ""
		if captcha != "" {
			reason = "验证码错误，请重试"
			if alertMsg != "" {
				reason = alertMsg
				if strings.Contains(alertMsg, "reCAPTCHA") {
					reason = "验证码错误，请重试"
				}
			}
		} else if alertMsg != "" {
			reason = alertMsg
			if strings.Contains(alertMsg, "reCAPTCHA") {
				reason = "验证码错误，请重试"
			}
		}
		return &CaptchaNeededError{Message: reason}
	}

	if alertMsg != "" {
		failCount++
		return fmt.Errorf("登录失败: %s", alertMsg)
	}

	// Check for account choice
	choices := extractAccountChoices(body)
	if choices != nil {
		failCount++
		return &AccountChoiceNeededError{Choices: choices}
	}

	finalURL := resp.Request.URL.String()
	stdlog.Printf("[oauth] final URL: %s", finalURL[:min(120, len(finalURL))])

	// Check for message= in URL (access denied — 但仍保存 session，后续可代理API)
	if strings.Contains(finalURL, "message=") {
		msg := extractURLMessage(finalURL)
		if msg != "访问被拒绝" {
			// 保存 session cookies（CAS已通过，housing 已设置 cookies）
			session.SaveCookiesFromHTTP(httpClient)
			return &AccessDeniedError{Message: msg}
		}
	}

	// 检测登录类型：casredirect（CAS直接登录）vs xjtuoauthlogin（OAuth登录）
	if strings.Contains(finalURL, "casredirect") {
		// CAS 直接登录：从URL提取 employeeNo 和 gsessionId
		employeeNo := extractURLParam(finalURL, "employeeNo")
		gsessionId := extractURLParam(finalURL, "gsessionId")
		stdlog.Printf("[oauth] casredirect: employeeNo=%s gsessionId=%s", employeeNo, gsessionId[:min(16, len(gsessionId))])
		return bedAuthenLogin(httpClient, employeeNo, gsessionId)
	}

	// OAuth 登录：调用 /xjtuCasLogin/caslogin
	return exchangeTokenFromHTTP(httpClient)
}

// bedAuthenLogin 调用 /bed/bedAuthenLogin（casredirect 页面的登录方式）
func bedAuthenLogin(httpClient *http.Client, employeeNo, gsessionId string) error {
	// 设置 xjtu_gsessionId cookie
	if gsessionId != "" {
		housingURL, _ := url.Parse(housingAPIURL)
		httpClient.Jar.SetCookies(housingURL, []*http.Cookie{
			{Name: "xjtu_gsessionId", Value: gsessionId},
		})
	}

	username := employeeNo
	if username == "" {
		username = session.Get().Account
	}

	loginURL := fmt.Sprintf("%s/appsys/bed/bedAuthenLogin?username=%s", housingAPIURL, username)
	stdlog.Printf("[bedAuthen] POST %s", loginURL[:min(80, len(loginURL))])
	resp, err := httpClient.Post(loginURL, "application/x-www-form-urlencoded", strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("bedAuthenLogin失败: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var j struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Token string `json:"token"`
	}
	json.Unmarshal(body, &j)

	stdlog.Printf("[bedAuthen] response: code=%d msg=%s token=%s", j.Code, j.Msg, j.Token[:min(16, len(j.Token))])

	if j.Code != 0 {
		if j.Msg != "" {
			return &AccessDeniedError{Message: j.Msg}
		}
		return fmt.Errorf("bedAuthenLogin失败: code=%d", j.Code)
	}

	if j.Token != "" {
		session.SetToken(j.Token)
	}
	session.SetStudentCode(username)
	ResetFailCount()
	session.SaveCookiesFromHTTP(httpClient)
	return nil
}

// exchangeTokenFromHTTP 调用 /xjtuCasLogin/caslogin 换取 token（OAuth 流程）
func exchangeTokenFromHTTP(httpClient *http.Client) error {
	resp, err := httpClient.Post(housingAPIURL+"/appsys/xjtuCasLogin/caslogin",
		"application/x-www-form-urlencoded", strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("换取token失败: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var j struct {
		Code     int    `json:"code"`
		Msg      string `json:"msg"`
		Token    string `json:"token"`
		Username string `json:"username"`
	}
	json.Unmarshal(body, &j)

	if j.Code != 0 {
		if j.Msg != "" {
			return &AccessDeniedError{Message: j.Msg}
		}
		return fmt.Errorf("换取token失败: code=%d", j.Code)
	}

	if j.Token != "" {
		session.SetToken(j.Token)
	}
	if j.Username != "" {
		session.SetStudentCode(j.Username)
	}

	ResetFailCount()
	session.SaveCookiesFromHTTP(httpClient)
	return nil
}

// ── MFA completion（照抄 Course-Genius CompleteMFALogin → followAndRegister）──

// CompleteLoginAfterMFA  MFA验证后访问 OAuth URL，CAS 自动通过并重定向到 housing
func CompleteLoginAfterMFA(client *resty.Client) error {
	return followOAuthAndExchange(client, "")
}

// followOAuthAndExchange 照抄 followAndRegister 模式：
// GET OAuth URL → CAS → POST CAS（MFA已验证）→ housing → 换token
func followOAuthAndExchange(client *resty.Client, startURL string) error {
	httpClient := client.GetClient()
	s := session.Get()

	// Step 1: GET OAuth URL → CAS 登录页
	oauthURL := buildOAuthURL()
	resp, err := httpClient.Get(oauthURL)
	if err != nil {
		return fmt.Errorf("MFA后访问OAuth失败: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	casURL := resp.Request.URL.String()
	stdlog.Printf("[mfa] after GET OAuth: %s", casURL[:min(80, len(casURL))])

	// 如果不在CAS → 说明自动通过了，检查结果
	if !strings.Contains(casURL, "login.xjtu.edu.cn") {
		if strings.Contains(casURL, "housing") {
			stdlog.Println("[mfa] CAS自动通过，直接换token")
			return exchangeTokenFromHTTP(httpClient)
		}
		if strings.Contains(casURL, "message=") {
			msg := extractURLMessage(casURL)
			return &AccessDeniedError{Message: msg}
		}
	}

	// Step 2: 提取新 execution
	execution := extractExecution(body)
	if execution == "" {
		return fmt.Errorf("MFA后无法获取execution")
	}
	stdlog.Printf("[mfa] execution=%s", execution[:min(40, len(execution))])

	// Step 3: 获取公钥
	if pubResp, err := httpClient.Get(casBaseURL + "/cas/jwt/publicKey"); err == nil {
		pubBody, _ := io.ReadAll(pubResp.Body)
		pubResp.Body.Close()
		SetPubKey(string(pubBody))
	}

	// Step 4: 加密密码
	encPwd, _ := EncryptPassword(s.Password)

	// Step 5: detectMFA（拿 mfaState，CAS应该知道MFA已完成）
	fpID := s.FpVisitorID
	if fpID == "" {
		fpID, _ = GetFingerprint()
	}
	mfaState := ""
	if mfaNeed, mfaState2, _ := detectMFA(client, s.Account, encPwd, fpID); true {
		mfaState = mfaState2
		stdlog.Printf("[mfa] detectMFA: need=%v state=%s", mfaNeed, mfaState2)
	}

	// Step 6: POST CAS（MFA已完成，CAS应该通过）
	stdlog.Println("[mfa] POST CAS login...")
	return postCASRaw(httpClient, casURL, s.Account, encPwd, execution, mfaState, fpID, "", "")

	// Exchange token
	token, username, errMsg, err := ExchangeOAuthToken(client)
	if err != nil {
		return fmt.Errorf("换取登录token失败: %w", err)
	}
	if errMsg != "" {
		return &AccessDeniedError{Message: errMsg}
	}
	if token != "" {
		session.SetToken(token)
		client.SetHeader("Token", token)
	}
	if username != "" {
		session.SetStudentCode(username)
	}

	ResetFailCount()
	session.SaveCookies(client)
	return nil
}

// ── Token exchange ──

func ExchangeOAuthToken(client *resty.Client) (token string, username string, errMsg string, err error) {
	resp, err := client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		Post(housingAPIURL + "/appsys/xjtuCasLogin/caslogin")
	if err != nil {
		return "", "", "", fmt.Errorf("获取登录token失败: %w", err)
	}

	var j struct {
		Code     int    `json:"code"`
		Msg      string `json:"msg"`
		Token    string `json:"token"`
		Username string `json:"username"`
	}
	json.Unmarshal(resp.Body(), &j)

	if j.Code != 0 {
		return "", "", j.Msg, nil
	}
	return j.Token, j.Username, "", nil
}

// ── Captcha ──

func GetCaptchaImage(client *resty.Client) ([]byte, error) {
	resp, err := client.R().Get(casBaseURL + "/cas/captcha.jpg")
	if err != nil {
		return nil, fmt.Errorf("获取验证码失败: %w", err)
	}
	return resp.Body(), nil
}

// ── Account choice（照抄 Course-Genius）──

func ChooseAccount(client *resty.Client, accountType string) error {
	if len(accountChoices) == 0 || accountChoiceExecution == "" {
		return fmt.Errorf("没有待处理的账户选择")
	}

	matchKeyword := "研究" // default postgraduate
	if accountType == "undergraduate" {
		matchKeyword = "本科"
	}

	var selectedLabel string
	for _, c := range accountChoices {
		if strings.Contains(c["name"], matchKeyword) {
			selectedLabel = c["label"]
			break
		}
	}
	if selectedLabel == "" && len(accountChoices) > 0 {
		selectedLabel = accountChoices[0]["label"]
	}

	fpID := session.Get().FpVisitorID

	resp, err := client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"execution":   accountChoiceExecution,
			"_eventId":    "submit",
			"geolocation": "",
			"fpVisitorId": fpID,
			"trustAgent":  "true",
			"username":    selectedLabel,
			"useDefault":  "false",
		}).
		Post(casBaseURL + "/cas/login")
	if err != nil {
		return fmt.Errorf("账户选择请求失败: %w", err)
	}

	accountChoiceExecution = ""
	accountChoices = nil

	if msg := extractAlertMessage(resp.Body()); msg != "" {
		return fmt.Errorf("账户选择失败: %s", msg)
	}

	return followOAuthAndExchange(client, "")
}

// ── Helpers ──

func buildOAuthURL() string {
	return fmt.Sprintf("%s?appId=%s&redirectUri=%s&responseType=code&scope=%s&state=%s",
		oauthAuthorizeURL, oauthAppID, oauthRedirectURI, oauthScope, oauthState)
}

func detectMFA(client *resty.Client, account, encPwd, fpID string) (need bool, state string, err error) {
	data := map[string]string{
		"username":    account,
		"password":    encPwd,
		"fpVisitorId": fpID,
	}
	resp, err := client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(data).
		Post(casBaseURL + "/cas/mfa/detect")
	if err != nil {
		return false, "", err
	}
	var j struct {
		Code int `json:"code"`
		Data struct {
			Need  bool   `json:"need"`
			State string `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &j); err != nil {
		return false, "", err
	}
	return j.Data.Need, j.Data.State, nil
}

// ── Error types ──

type MFANeededError struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

func (e *MFANeededError) Error() string { return e.Reason }

type CaptchaNeededError struct {
	Message string `json:"message"`
}

func (e *CaptchaNeededError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "需要验证码"
}

type AccountChoiceNeededError struct {
	Choices []map[string]string `json:"choices"`
}

func (e *AccountChoiceNeededError) Error() string { return "需要选择账户身份" }

type AccessDeniedError struct {
	Message string `json:"message"`
}

func (e *AccessDeniedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "访问被拒绝"
}

// ── HTML parsing ──

func extractExecution(html []byte) string {
	return extractInputValue(html, "execution")
}

func extractInputValue(html []byte, name string) string {
	s := string(html)
	search := fmt.Sprintf(`name="%s"`, name)
	idx := strings.Index(s, search)
	if idx < 0 {
		return ""
	}
	valIdx := strings.Index(s[idx:], `value="`)
	if valIdx < 0 {
		return ""
	}
	valIdx += len(`value="`)
	end := strings.Index(s[idx+valIdx:], `"`)
	if end < 0 {
		return ""
	}
	return s[idx+valIdx : idx+valIdx+end]
}

func extractAlertMessage(htmlContent []byte) string {
	s := string(htmlContent)
	idx := strings.Index(s, "el-alert")
	if idx < 0 {
		return ""
	}
	titleIdx := strings.Index(s[idx:], `title="`)
	if titleIdx < 0 {
		return ""
	}
	titleIdx += len(`title="`)
	end := strings.Index(s[idx+titleIdx:], `"`)
	if end < 0 {
		return ""
	}
	return s[idx+titleIdx : idx+titleIdx+end]
}

func extractAccountChoices(htmlContent []byte) []map[string]string {
	s := string(htmlContent)
	if !strings.Contains(s, "account-wrap") {
		return nil
	}
	var choices []map[string]string
	for {
		wrapIdx := strings.Index(s, "account-wrap")
		if wrapIdx < 0 {
			break
		}
		s = s[wrapIdx:]
		nameStart := strings.Index(s, `class="name"`)
		if nameStart < 0 {
			break
		}
		nameStart = strings.Index(s[nameStart:], ">") + nameStart + 1
		nameEnd := strings.Index(s[nameStart:], "<")
		name := strings.TrimSpace(s[nameStart : nameStart+nameEnd])

		labelIdx := strings.Index(s, `label="`)
		if labelIdx < 0 {
			break
		}
		labelIdx += len(`label="`)
		labelEnd := strings.Index(s[labelIdx:], `"`)
		label := s[labelIdx : labelIdx+labelEnd]

		choices = append(choices, map[string]string{"name": name, "label": label})
		s = s[nameStart+nameEnd:]
		if len(s) < 100 {
			break
		}
	}
	return choices
}

// extractURLParam 从URL的query string或fragment中提取参数
func extractURLParam(rawURL, param string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	// 先查主 query string
	if v := u.Query().Get(param); v != "" {
		return v
	}
	// 再查 fragment 中的 query string
	if u.Fragment != "" {
		frag := u.Fragment
		if idx := strings.Index(frag, "?"); idx >= 0 {
			if vals, err := url.ParseQuery(frag[idx+1:]); err == nil {
				return vals.Get(param)
			}
		}
	}
	return ""
}

func extractURLMessage(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "访问被拒绝"
	}
	if u.Fragment != "" {
		frag := u.Fragment
		if idx := strings.Index(frag, "?"); idx >= 0 {
			if vals, err := url.ParseQuery(frag[idx+1:]); err == nil {
				if msg := vals.Get("message"); msg != "" {
					if decoded, err := url.QueryUnescape(msg); err == nil {
						return decoded
					}
					return msg
				}
			}
		}
	}
	return "访问被拒绝"
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
