package web

import (
	"backup-x/entity"
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
)

//go:embed user_manage.html
var userManageEmbedFile embed.FS

type userManageData struct {
	Users       []entity.User
	CurrentUser string
}

// UserManagePage 用户管理页面
func UserManagePage(writer http.ResponseWriter, request *http.Request) {
	tmpl, err := template.ParseFS(userManageEmbedFile, "user_manage.html")
	if err != nil {
		log.Println(err)
		return
	}

	store, _ := entity.LoadUsers()
	sd := GetSession(request)
	currentUser := ""
	if sd != nil {
		currentUser = sd.Username
	}

	tmpl.Execute(writer, &userManageData{
		Users:       store.Users,
		CurrentUser: currentUser,
	})
}

// UserAdd 添加用户
func UserAdd(writer http.ResponseWriter, request *http.Request) {
	username := strings.TrimSpace(request.FormValue("username"))
	password := request.FormValue("password")
	role := strings.TrimSpace(request.FormValue("role"))

	if username == "" || password == "" {
		writer.Write([]byte("请输入用户名和密码"))
		return
	}

	if role != "admin" && role != "viewer" {
		role = "viewer"
	}

	err := entity.AddUser(username, password, role)
	if err != nil {
		writer.Write([]byte(err.Error()))
		return
	}

	log.Printf("添加用户: %s (%s)\n", username, role)
	writer.Write([]byte("ok"))
}

// UserUpdate 更新用户
func UserUpdate(writer http.ResponseWriter, request *http.Request) {
	username := strings.TrimSpace(request.FormValue("username"))
	password := request.FormValue("password")
	role := strings.TrimSpace(request.FormValue("role"))

	if username == "" {
		writer.Write([]byte("用户名不能为空"))
		return
	}

	if role != "admin" && role != "viewer" {
		role = "viewer"
	}

	err := entity.UpdateUser(username, password, role)
	if err != nil {
		writer.Write([]byte(err.Error()))
		return
	}

	log.Printf("更新用户: %s (%s)\n", username, role)
	writer.Write([]byte("ok"))
}

// UserDelete 删除用户
func UserDelete(writer http.ResponseWriter, request *http.Request) {
	username := strings.TrimSpace(request.FormValue("username"))

	if username == "" {
		writer.Write([]byte("用户名不能为空"))
		return
	}

	// 不允许删除自己
	sd := GetSession(request)
	if sd != nil && sd.Username == username {
		writer.Write([]byte("不能删除当前登录的用户"))
		return
	}

	// 检查是否是最后一个 admin
	store, _ := entity.LoadUsers()
	adminCount := 0
	for _, u := range store.Users {
		if u.Role == "admin" {
			adminCount++
		}
	}

	targetUser, _ := entity.GetUserByUsername(username)
	if targetUser != nil && targetUser.Role == "admin" && adminCount <= 1 {
		writer.Write([]byte("不能删除最后一个管理员"))
		return
	}

	err := entity.DeleteUser(username)
	if err != nil {
		writer.Write([]byte(err.Error()))
		return
	}

	log.Printf("删除用户: %s\n", username)
	writer.Write([]byte("ok"))
}
