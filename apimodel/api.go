package apimodel

import (
	"fmt"
	"github.com/ringoid/commons"
)

type ActionReq struct {
	AccessToken string   `json:"accessToken"`
	Actions     []Action `json:"actions"`
}

func (req ActionReq) String() string {
	return fmt.Sprintf("%#v", req)
}

type Action struct {
	SourceFeed     string  `json:"sourceFeed"`
	ActionType     string  `json:"actionType"`
	TargetPhotoId  string  `json:"targetPhotoId"`
	TargetUserId   string  `json:"targetUserId"`
	Text           string  `json:"text"`
	LikeCount      int     `json:"likeCount"`
	ViewCount      int     `json:"viewCount"`
	ViewTimeMillis int64   `json:"viewTimeMillis"`
	BlockReasonNum int     `json:"blockReasonNum"`
	Lat            float64 `json:"lat"`
	Lon            float64 `json:"lon"`
	ActionTime     int64   `json:"actionTime"`
}

func (req Action) String() string {
	return fmt.Sprintf("%#v", req)
}

type ActionResponse struct {
	commons.BaseResponse
	LastActionTime int64 `json:"lastActionTime"`
}

func (resp ActionResponse) String() string {
	return fmt.Sprintf("%#v", resp)
}
