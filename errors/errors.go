package errors

import "fmt"

type UserExistError struct {
	Email string
}

func (e UserExistError) Error() string {
	return fmt.Sprintf("User with %s email exists", e.Email)
}

type NoUserError struct {
	Email string
}

func (e NoUserError) Error() string {
	return fmt.Sprintf("No User with %s email exists", e.Email)

}

type PasskeyExists struct{}

func (e PasskeyExists) Error() string {
	return fmt.Sprintf("Passkey exist within 24 hours")
}
