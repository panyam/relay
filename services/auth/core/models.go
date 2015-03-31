package core

import (
	. "github.com/panyam/relay/services/msg/core"
	"time"
)

type Registration struct {
	Id               int64
	Username         string
	Team             *Team
	Created          time.Time
	AddressType      string
	Address          string
	VerificationData string
	Status           int
}

type UserLogin struct {
	Id          int64
	Source      string
	SourceId    string
	User        *User
	Credentials map[string]interface{}
	Created     time.Time
	Status      int
}