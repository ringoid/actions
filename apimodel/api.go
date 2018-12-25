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
	SourceFeed      string `json:"sourceFeed"`
	ActionType      string `json:"actionType"`
	TargetPhotoId   string `json:"targetPhotoId"`
	TargetUserId    string `json:"targetUserId"`
	Text            string `json:"text"`
	LikeCount       int    `json:"likeCount"`
	ViewCount       int    `json:"viewCount"`
	ViewTimeSec     int    `json:"viewTimeSec"`
	OpenChatCount   int    `json:"openChatCount"`
	OpenChatTimeSec int    `json:"openChatTimeSec"`
	BlockReasonNum  int    `json:"blockReasonNum"`
	ActionTime      int    `json:"actionTime"`
}

func (req Action) String() string {
	return fmt.Sprintf("%#v", req)
}
