package entity

import (
	"backup-x/util"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

// User 用户
type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"` // 加密存储
	Role     string `yaml:"role"`     // admin / viewer
}

// UserStore 用户存储
type UserStore struct {
	Users      []User `yaml:"users"`
	EncryptKey string `yaml:"encryptkey"`
}

var userStoreCache *UserStore
var userStoreLock sync.Mutex

// getUserFilePath 获得用户文件路径
func getUserFilePath() string {
	_, err := os.Stat(parentSavePath)
	if err != nil {
		os.Mkdir(parentSavePath, 0750)
	}
	return parentSavePath + string(os.PathSeparator) + ".backup_x_users.yaml"
}

// LoadUsers 加载用户列表
func LoadUsers() (*UserStore, error) {
	userStoreLock.Lock()
	defer userStoreLock.Unlock()

	if userStoreCache != nil {
		return userStoreCache, nil
	}

	store := &UserStore{}
	filePath := getUserFilePath()

	_, err := os.Stat(filePath)
	if err != nil {
		// 文件不存在
		userStoreCache = store
		return store, nil
	}

	byt, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println("用户文件读取失败")
		return store, err
	}

	err = yaml.Unmarshal(byt, store)
	if err != nil {
		log.Println("反序列化用户文件失败", err)
		return store, err
	}

	userStoreCache = store
	return store, nil
}

// SaveUsers 保存用户列表
func (store *UserStore) SaveUsers() error {
	userStoreLock.Lock()
	defer userStoreLock.Unlock()

	byt, err := yaml.Marshal(store)
	if err != nil {
		log.Println(err)
		return err
	}

	err = ioutil.WriteFile(getUserFilePath(), byt, 0600)
	if err != nil {
		log.Println(err)
		return err
	}

	// 清空缓存
	userStoreCache = nil
	return nil
}

// ensureEncryptKey 确保 EncryptKey 存在
func (store *UserStore) ensureEncryptKey() error {
	if store.EncryptKey == "" {
		key, err := util.GenerateEncryptKey()
		if err != nil {
			return err
		}
		store.EncryptKey = key
	}
	return nil
}

// AddUser 添加用户
func AddUser(username, password, role string) error {
	store, err := LoadUsers()
	if err != nil {
		return err
	}

	// 检查用户名是否已存在
	for _, u := range store.Users {
		if u.Username == username {
			return errors.New("用户名已存在")
		}
	}

	if err := store.ensureEncryptKey(); err != nil {
		return err
	}

	// 加密密码
	encryptedPwd, err := util.EncryptByEncryptKey(store.EncryptKey, password)
	if err != nil {
		return errors.New("加密密码失败")
	}

	store.Users = append(store.Users, User{
		Username: username,
		Password: encryptedPwd,
		Role:     role,
	})

	return store.SaveUsers()
}

// DeleteUser 删除用户
func DeleteUser(username string) error {
	store, err := LoadUsers()
	if err != nil {
		return err
	}

	found := false
	newUsers := []User{}
	for _, u := range store.Users {
		if u.Username == username {
			found = true
			continue
		}
		newUsers = append(newUsers, u)
	}

	if !found {
		return errors.New("用户不存在")
	}

	store.Users = newUsers
	return store.SaveUsers()
}

// UpdateUser 更新用户（密码为空则不更新密码）
func UpdateUser(username, newPassword, newRole string) error {
	store, err := LoadUsers()
	if err != nil {
		return err
	}

	found := false
	for i, u := range store.Users {
		if u.Username == username {
			found = true
			if newPassword != "" {
				if err := store.ensureEncryptKey(); err != nil {
					return err
				}
				encryptedPwd, err := util.EncryptByEncryptKey(store.EncryptKey, newPassword)
				if err != nil {
					return errors.New("加密密码失败")
				}
				store.Users[i].Password = encryptedPwd
			}
			if newRole != "" {
				store.Users[i].Role = newRole
			}
			break
		}
	}

	if !found {
		return errors.New("用户不存在")
	}

	return store.SaveUsers()
}

// Authenticate 验证用户名密码，返回 User 和 error
func Authenticate(username, password string) (*User, error) {
	store, err := LoadUsers()
	if err != nil {
		return nil, err
	}

	for _, u := range store.Users {
		if u.Username == username {
			// 解密密码
			decryptedPwd, err := util.DecryptByEncryptKey(store.EncryptKey, u.Password)
			if err != nil {
				return nil, errors.New("解密密码失败")
			}
			if decryptedPwd == password {
				return &u, nil
			}
			return nil, errors.New("密码错误")
		}
	}

	return nil, errors.New("用户不存在")
}

// GetUserByUsername 根据用户名获取用户
func GetUserByUsername(username string) (*User, error) {
	store, err := LoadUsers()
	if err != nil {
		return nil, err
	}

	for _, u := range store.Users {
		if u.Username == username {
			return &u, nil
		}
	}

	return nil, errors.New("用户不存在")
}

// HasAnyUser 是否存在任何用户
func HasAnyUser() bool {
	store, err := LoadUsers()
	if err != nil {
		return false
	}
	return len(store.Users) > 0
}
