package web

import (
	"backup-x/entity"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed writing.html
var writingEmbedFile embed.FS

const VersionEnv = "BACKUP_X_VERSION"

type writtingData struct {
	entity.Config
	Version     string
	CurrentUser string
	CurrentRole string
	Hours       []int
}

// WritingConfig 填写配置信息
func WritingConfig(writer http.ResponseWriter, request *http.Request) {
	tmpl, err := template.ParseFS(writingEmbedFile, "writing.html")
	if err != nil {
		log.Println(err)
		return
	}

	// 获取当前用户信息
	sd := GetSession(request)
	currentUser := ""
	currentRole := ""
	if sd != nil {
		currentUser = sd.Username
		currentRole = sd.Role
	}

	var hours []int
	for i := 0; i < 24; i++ {
		hours = append(hours, i)
	}

	conf, err := entity.GetConfigCache()
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err == nil {
		// Pad to 16 items as in original behavior
		for len(conf.BackupConfig) < 16 {
			conf.BackupConfig = append(conf.BackupConfig, entity.BackupConfig{SaveDays: 30, SaveDaysS3: 60, StartTime: 1, Period: 1440, BackupType: 0})
		}
		tmpl.Execute(writer, &writtingData{Config: conf, Version: os.Getenv(VersionEnv), CurrentUser: currentUser, CurrentRole: currentRole, Hours: hours})
		return
	}

	// default config
	backupConf := []entity.BackupConfig{}
	for i := 0; i < 16; i++ {
		backupConf = append(backupConf, entity.BackupConfig{SaveDays: 30, SaveDaysS3: 60, StartTime: 1, Period: 1440, BackupType: 0})
	}
	conf = entity.Config{
		BackupConfig: backupConf,
	}

	tmpl.Execute(writer, &writtingData{Config: conf, Version: os.Getenv(VersionEnv), CurrentUser: currentUser, CurrentRole: currentRole, Hours: hours})
}
