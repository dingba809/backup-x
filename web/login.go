package web

import (
	"backup-x/entity"
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:embed login.html
var loginEmbedFile embed.FS

type loginData struct {
	Error string
}

type loginDetectNew struct {
	FailTimes    int
	LastFailTime time.Time
}

var ldNew = &loginDetectNew{}

// LoginPage 渲染登录页面
func LoginPage(writer http.ResponseWriter, request *http.Request) {
	// 已登录则跳转首页
	if sd := GetSession(request); sd != nil {
		http.Redirect(writer, request, "/", http.StatusFound)
		return
	}

	renderLoginPage(writer, "")
}

// LoginHandler 处理登录请求
func LoginHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Redirect(writer, request, "/login", http.StatusFound)
		return
	}

	// 防暴力破解：5次失败后延迟5分钟
	if ldNew.FailTimes >= 5 {
		if time.Since(ldNew.LastFailTime) < 5*time.Minute {
			log.Printf("%s 登录失败超过5次! 请等待5分钟后再试\n", request.RemoteAddr)
			renderLoginPage(writer, "登录失败次数过多，请等待5分钟后再试")
			return
		}
		ldNew.FailTimes = 0
	}

	username := strings.TrimSpace(request.FormValue("username"))
	password := request.FormValue("password")

	if username == "" || password == "" {
		renderLoginPage(writer, "请输入用户名和密码")
		return
	}

	user, err := entity.Authenticate(username, password)
	if err != nil {
		ldNew.FailTimes++
		ldNew.LastFailTime = time.Now()
		log.Printf("%s 登录失败! 用户名: %s\n", request.RemoteAddr, username)
		renderLoginPage(writer, "用户名或密码错误")
		return
	}

	// 登录成功
	ldNew.FailTimes = 0
	log.Printf("%s 用户 %s 登录成功\n", request.RemoteAddr, username)

	if err := CreateSession(writer, user.Username, user.Role); err != nil {
		renderLoginPage(writer, "创建会话失败，请重试")
		return
	}

	http.Redirect(writer, request, "/", http.StatusFound)
}

// LogoutHandler 登出
func LogoutHandler(writer http.ResponseWriter, request *http.Request) {
	sd := GetSession(request)
	if sd != nil {
		log.Printf("%s 用户 %s 登出\n", request.RemoteAddr, sd.Username)
	}
	DestroySession(writer, request)
	http.Redirect(writer, request, "/login", http.StatusFound)
}

func renderLoginPage(writer http.ResponseWriter, errMsg string) {
	tmpl, err := template.ParseFS(loginEmbedFile, "login.html")
	if err != nil {
		log.Println(err)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(writer, &loginData{Error: errMsg})
}
