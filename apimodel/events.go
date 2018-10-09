package apimodel

import (
	"time"
	"fmt"
)

const (
	LikeActionType = "LIKE"
	ViewActionType = "VIEW"
)

type UserLikePhotoEvent struct {
	UserId                string `json:"userId"`
	PhotoId               string `json:"photoId"`
	OriginPhotoId         string `json:"originPhotoId"`
	TargetUserId          string `json:"targetUserId"`
	LikeCount             int    `json:"likeCount"`
	Source                string `json:"source"`
	LikedAt               int    `json:"likedAt"`
	UnixTime              int64  `json:"unixTime"`
	EventType             string `json:"eventType"`
	InternalServiceSource string `json:"internalServiceSource"`
}

func (event UserLikePhotoEvent) String() string {
	return fmt.Sprintf("%#v", event)
}

func NewUserLikePhotoEvent(userId, photoId, originPhotoId, targetUserId, source string, likeCount, likedAt int, serviceName string) *UserLikePhotoEvent {
	return &UserLikePhotoEvent{
		UserId:                userId,
		PhotoId:               photoId,
		OriginPhotoId:         originPhotoId,
		TargetUserId:          targetUserId,
		LikeCount:             likeCount,
		LikedAt:               likedAt,
		Source:                source,
		UnixTime:              time.Now().Unix(),
		EventType:             "ACTION_USER_LIKE_PHOTO",
		InternalServiceSource: serviceName,
	}
}

type UserViewPhotoEvent struct {
	UserId                string `json:"userId"`
	PhotoId               string `json:"photoId"`
	OriginPhotoId         string `json:"originPhotoId"`
	TargetUserId          string `json:"targetUserId"`
	ViewCount             int    `json:"viewCount"`
	ViewTimeSec           int    `json:"viewTimeSec"`
	ViewAt                int    `json:"viewAt"`
	Source                string `json:"source"`
	UnixTime              int64  `json:"unixTime"`
	EventType             string `json:"eventType"`
	InternalServiceSource string `json:"internalServiceSource"`
}

func (event UserViewPhotoEvent) String() string {
	return fmt.Sprintf("%#v", event)
}

func NewUserViewPhotoEvent(userId, photoId, originPhotoId, targetUserId, source string, viewCount, viewTimeSec, viewAt int, serviceName string) *UserViewPhotoEvent {
	return &UserViewPhotoEvent{
		UserId:                userId,
		PhotoId:               photoId,
		OriginPhotoId:         originPhotoId,
		TargetUserId:          targetUserId,
		ViewCount:             viewCount,
		ViewTimeSec:           viewTimeSec,
		ViewAt:                viewAt,
		Source:                source,
		UnixTime:              time.Now().Unix(),
		EventType:             "ACTION_USER_VIEW_PHOTO",
		InternalServiceSource: serviceName,
	}
}
