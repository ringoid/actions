package apimodel

import (
	"fmt"
)

type WarmUpRequest struct {
	WarmUpRequest bool `json:"warmUpRequest"`
}

func (req WarmUpRequest) String() string {
	return fmt.Sprintf("%#v", req)
}

type InternalGetUserIdReq struct {
	WarmUpRequest bool   `json:"warmUpRequest"`
	AccessToken   string `json:"accessToken"`
	BuildNum      int    `json:"buildNum"`
	IsItAndroid   bool   `json:"isItAndroid"`
}

func (req InternalGetUserIdReq) String() string {
	return fmt.Sprintf("%#v", req)
}

type InternalGetUserIdResp struct {
	BaseResponse
	UserId string `json:"userId"`
}

func (resp InternalGetUserIdResp) String() string {
	return fmt.Sprintf("%#v", resp)
}

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
