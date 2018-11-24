package apimodel

import (
	"fmt"
)

type ActionReq struct {
	AccessToken string   `json:"accessToken"`
	Actions     []Action `json:"actions"`
}

func (req ActionReq) String() string {
	return fmt.Sprintf("%#v", req)
}

type Action struct {
	SourceFeed    string `json:"sourceFeed"`
	ActionType    string `json:"actionType"`
	TargetPhotoId string `json:"targetPhotoId"`
	TargetUserId  string `json:"targetUserId"`
	LikeCount     int    `json:"likeCount"`
	ViewCount     int    `json:"viewCount"`
	ViewTimeSec   int    `json:"viewTimeSec"`
	ActionTime    int    `json:"actionTime"`
}

func (req Action) String() string {
	return fmt.Sprintf("%#v", req)
}
