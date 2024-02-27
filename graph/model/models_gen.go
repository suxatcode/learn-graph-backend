// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

type CreateEntityResult struct {
	ID     string  `json:"ID"`
	Status *Status `json:"Status,omitempty"`
}

type CreateUserResult struct {
	Login *LoginResult `json:"login"`
}

type Edge struct {
	ID     string  `json:"id"`
	From   string  `json:"from"`
	To     string  `json:"to"`
	Weight float64 `json:"weight"`
}

type Graph struct {
	Nodes []*Node `json:"nodes,omitempty"`
	Edges []*Edge `json:"edges,omitempty"`
}

type LoginAuthentication struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResult struct {
	Success  bool    `json:"success"`
	Token    string  `json:"token"`
	UserID   string  `json:"userID"`
	UserName string  `json:"userName"`
	Message  *string `json:"message,omitempty"`
}

type Mutation struct {
}

type Node struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Resources   *string `json:"resources,omitempty"`
}

type NodeEdit struct {
	User           string       `json:"user"`
	Type           NodeEditType `json:"type"`
	NewDescription string       `json:"newDescription"`
	NewResources   *string      `json:"newResources,omitempty"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

type Query struct {
}

type Status struct {
	Message string `json:"Message"`
}

type Text struct {
	Translations []*Translation `json:"translations"`
}

type Translation struct {
	Language string `json:"language"`
	Content  string `json:"content"`
}

type NodeEditType string

const (
	NodeEditTypeCreate NodeEditType = "create"
	NodeEditTypeEdit   NodeEditType = "edit"
)

var AllNodeEditType = []NodeEditType{
	NodeEditTypeCreate,
	NodeEditTypeEdit,
}

func (e NodeEditType) IsValid() bool {
	switch e {
	case NodeEditTypeCreate, NodeEditTypeEdit:
		return true
	}
	return false
}

func (e NodeEditType) String() string {
	return string(e)
}

func (e *NodeEditType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = NodeEditType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid NodeEditType", str)
	}
	return nil
}

func (e NodeEditType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
