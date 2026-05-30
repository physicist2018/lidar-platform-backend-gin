package key

import "fmt"

func UserListKey(filterHash string) string {
	return fmt.Sprintf("user:list:%s", filterHash)
}

func UserByIDKey(id uint) string {
	return fmt.Sprintf("user:id:%d", id)
}

func UserByEmailKey(email string) string {
	return fmt.Sprintf("user:email:%s", email)
}
