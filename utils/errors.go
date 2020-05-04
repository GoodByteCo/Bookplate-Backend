package utils

import "fmt"

type UserExistError struct {
	email string
}

func (e UserExistError) Error() string {
	return fmt.Sprintf("User with %s email exists", e.email)
}

type NoUserError struct {
	email string
}

func (e NoUserError) Error() string {
	return fmt.Sprintf("No User with %s email exists", e.email)

}
