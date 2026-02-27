package entity

import (
	"backup-x/util"
	"log"
)

// MigrateUsersFromConfig 从旧配置迁移用户到独立用户文件
// 如果旧配置有 Username/Password，迁移到 UserStore，角色为 admin
// 如果没有任何用户，创建默认 admin 账户
func MigrateUsersFromConfig() {
	// 已有用户则跳过
	if HasAnyUser() {
		return
	}

	conf, err := GetConfigCache()
	if err == nil && conf.Username != "" && conf.Password != "" {
		// 从旧配置迁移
		store := &UserStore{}
		if err := store.ensureEncryptKey(); err != nil {
			log.Println("迁移用户时生成 EncryptKey 失败:", err)
			return
		}

		// 旧密码已经用旧 EncryptKey 加密，需要先解密再用新 EncryptKey 加密
		plainPwd := conf.Password
		if conf.EncryptKey != "" {
			decrypted, err := util.DecryptByEncryptKey(conf.EncryptKey, conf.Password)
			if err == nil {
				plainPwd = decrypted
			}
		}

		encryptedPwd, err := util.EncryptByEncryptKey(store.EncryptKey, plainPwd)
		if err != nil {
			log.Println("迁移用户时加密密码失败:", err)
			return
		}

		store.Users = append(store.Users, User{
			Username: conf.Username,
			Password: encryptedPwd,
			Role:     "admin",
		})

		if err := store.SaveUsers(); err != nil {
			log.Println("迁移用户时保存失败:", err)
			return
		}

		log.Printf("已从旧配置迁移用户: %s (admin)\n", conf.Username)
		return
	}

	// 没有任何用户，创建默认账户
	log.Println("未找到用户，创建默认账户 admin/admin123")
	if err := AddUser("admin", "admin123", "admin"); err != nil {
		log.Println("创建默认账户失败:", err)
	}
}
