package main

import (
	"context"
	basicLambda "github.com/aws/aws-lambda-go/lambda"
	"../apimodel"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"os"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/ringoid/commons"
	"sort"
	"io"
	"strings"
	"crypto/sha1"
)

var anlogger *commons.Logger
var awsDeliveryStreamClient *firehose.Firehose
var deliveryStreamName string
var internalAuthFunctionName string
var awsKinesisClient *kinesis.Kinesis
var commonStreamName string
var clientLambda *lambda.Lambda

const maxMessageLengthInSymbols = 1000

func init() {
	var env string
	var ok bool
	var papertrailAddress string
	var err error
	var awsSession *session.Session

	env, ok = os.LookupEnv("ENV")
	if !ok {
		fmt.Printf("lambda-initialization : actions.go : env can not be empty ENV\n")
		os.Exit(1)
	}
	fmt.Printf("lambda-initialization : actions.go : start with ENV = [%s]\n", env)

	papertrailAddress, ok = os.LookupEnv("PAPERTRAIL_LOG_ADDRESS")
	if !ok {
		fmt.Printf("lambda-initialization : actions.go : env can not be empty PAPERTRAIL_LOG_ADDRESS\n")
		os.Exit(1)
	}
	fmt.Printf("lambda-initialization : actions.go : start with PAPERTRAIL_LOG_ADDRESS = [%s]\n", papertrailAddress)

	anlogger, err = commons.New(papertrailAddress, fmt.Sprintf("%s-%s", env, "actions"))
	if err != nil {
		fmt.Errorf("lambda-initialization : actions.go : error during startup : %v\n", err)
		os.Exit(1)
	}
	anlogger.Debugf(nil, "lambda-initialization : actions.go : logger was successfully initialized")

	internalAuthFunctionName, ok = os.LookupEnv("INTERNAL_AUTH_FUNCTION_NAME")
	if !ok {
		anlogger.Fatalf(nil, "lambda-initialization : actions.go : env can not be empty INTERNAL_AUTH_FUNCTION_NAME")
	}
	anlogger.Debugf(nil, "lambda-initialization : actions.go : start with INTERNAL_AUTH_FUNCTION_NAME = [%s]", internalAuthFunctionName)

	awsSession, err = session.NewSession(aws.NewConfig().
		WithRegion(commons.Region).WithMaxRetries(commons.MaxRetries).
		WithLogger(aws.LoggerFunc(func(args ...interface{}) { anlogger.AwsLog(args) })).WithLogLevel(aws.LogOff))
	if err != nil {
		anlogger.Fatalf(nil, "lambda-initialization : actions.go : error during initialization : %v", err)
	}
	anlogger.Debugf(nil, "lambda-initialization : actions.go : aws session was successfully initialized")

	deliveryStreamName, ok = os.LookupEnv("DELIVERY_STREAM")
	if !ok {
		anlogger.Fatalf(nil, "lambda-initialization : actions.go : env can not be empty DELIVERY_STREAM")
	}
	anlogger.Debugf(nil, "lambda-initialization : actions.go : start with DELIVERY_STREAM = [%s]", deliveryStreamName)

	commonStreamName, ok = os.LookupEnv("COMMON_STREAM")
	if !ok {
		anlogger.Fatalf(nil, "lambda-initialization : actions.go : env can not be empty COMMON_STREAM")
		os.Exit(1)
	}
	anlogger.Debugf(nil, "lambda-initialization : actions.go : start with DELIVERY_STREAM = [%s]", commonStreamName)

	awsKinesisClient = kinesis.New(awsSession)
	anlogger.Debugf(nil, "lambda-initialization : actions.go : kinesis client was successfully initialized")

	awsDeliveryStreamClient = firehose.New(awsSession)
	anlogger.Debugf(nil, "lambda-initialization : actions.go : firehose client was successfully initialized")

	clientLambda = lambda.New(awsSession)
	anlogger.Debugf(nil, "lambda-initialization : actions.go : lambda client was successfully initialized")
}

func handler(ctx context.Context, request events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	lc, _ := lambdacontext.FromContext(ctx)

	userAgent := request.Headers["user-agent"]
	if strings.HasPrefix(userAgent, "ELB-HealthChecker") {
		return commons.NewServiceResponse("{}"), nil
	}

	if request.HTTPMethod != "POST" {
		return commons.NewWrongHttpMethodServiceResponse(), nil
	}
	sourceIp := request.Headers["x-forwarded-for"]

	anlogger.Debugf(lc, "actions.go : start handle request %v", request)

	appVersion, isItAndroid, ok, errStr := commons.ParseAppVersionFromHeaders(request.Headers, anlogger, lc)
	if !ok {
		anlogger.Errorf(lc, "actions.go : return %s to client", errStr)
		return commons.NewServiceResponse(errStr), nil
	}

	reqParam, ok, errStr := parseParams(request.Body, lc)
	if !ok {
		anlogger.Errorf(lc, "actions.go : return %s to client", errStr)
		return commons.NewServiceResponse(errStr), nil
	}

	userId, ok, _, errStr := commons.CallVerifyAccessToken(appVersion, isItAndroid, reqParam.AccessToken, internalAuthFunctionName, clientLambda, anlogger, lc)
	if !ok {
		anlogger.Errorf(lc, "actions.go : return %s to client", errStr)
		return commons.NewServiceResponse(errStr), nil
	}

	if ok, errStr = checkUserUserIds(reqParam, userId, lc); !ok {
		anlogger.Errorf(lc, "actions.go : return %s to client", errStr)
		return commons.NewServiceResponse(errStr), nil
	}

	sort.Slice(reqParam.Actions, func(i, j int) bool {
		return reqParam.Actions[i].ActionTime < reqParam.Actions[j].ActionTime
	})

	//todo:future place of optimization - we can use batch request model later
	for _, each := range reqParam.Actions {
		var event interface{}
		var partitionKey string
		var originPhotoId string
		if each.ActionType != commons.LocationActionType {
			originPhotoId, ok = commons.GetOriginPhotoId(userId, each.TargetPhotoId, anlogger, lc)
			if !ok {
				errStr := commons.InternalServerError
				anlogger.Errorf(lc, "actions.go :  userId [%s], return %s to client", userId, errStr)
				return commons.NewServiceResponse(errStr), nil
			}
		}
		switch each.ActionType {
		case commons.LikeActionType:
			event = commons.NewUserLikePhotoEvent(userId, each.TargetPhotoId, originPhotoId, each.TargetUserId, each.SourceFeed, sourceIp, each.LikeCount, each.ActionTime, "")
			//partitionKey = commons.GeneratePartitionKey(userId, each.TargetUserId)
			partitionKey = commons.GeneratePartitionKey(userId)
		case commons.ViewActionType:
			event = commons.NewUserViewPhotoEvent(userId, each.TargetPhotoId, originPhotoId, each.TargetUserId, each.SourceFeed, sourceIp, each.ViewCount, each.ViewTimeMillis, each.ActionTime, "")
			//partitionKey = commons.GeneratePartitionKey(userId, each.TargetUserId)
			partitionKey = commons.GeneratePartitionKey(userId)

		case commons.BlockActionType:
			event = commons.NewUserBlockOtherEvent(userId, each.TargetUserId, each.TargetPhotoId, originPhotoId, each.SourceFeed, sourceIp, each.ActionTime, each.BlockReasonNum, "")
			//partitionKey = commons.GeneratePartitionKey(userId, each.TargetUserId)
			partitionKey = commons.GeneratePartitionKey(userId)

		case commons.UnlikeActionType:
			event = commons.NewUserUnLikePhotoEvent(userId, each.TargetPhotoId, originPhotoId, each.TargetUserId, each.SourceFeed, sourceIp, each.ActionTime, "")
			//partitionKey = commons.GeneratePartitionKey(userId, each.TargetUserId)
			partitionKey = commons.GeneratePartitionKey(userId)

		case commons.MessageActionType:
			if len(each.Text) == 0 {
				anlogger.Errorf(lc, "actions.go : userId [%s], empty text in a message [%s]", userId, each.Text)
				errStr := commons.WrongRequestParamsClientError
				return commons.NewServiceResponse(errStr), nil
			}
			if len([]rune(each.Text)) > maxMessageLengthInSymbols {
				anlogger.Errorf(lc, "actions.go : too long [%d] text [%s] for userId [%s]", len([]rune(each.Text)), each.Text, userId)
				errStr := commons.WrongRequestParamsClientError
				anlogger.Errorf(lc, "actions.go :  userId [%s], return %s to client", userId, errStr)
				return commons.NewServiceResponse(errStr), nil
			}
			messageId, ok, errStr := generateMessageId(userId, each.Text, each.ActionTime, lc)
			if !ok {
				anlogger.Errorf(lc, "actions.go : userId [%s], return %s to client", userId, errStr)
				return commons.NewServiceResponse(errStr), nil
			}
			conversationId, ok, errStr := generateConversationId(userId, each.TargetUserId, lc)
			if !ok {
				anlogger.Errorf(lc, "actions.go : userId [%s], return %s to client", userId, errStr)
				return commons.NewServiceResponse(errStr), nil
			}
			event = commons.NewUserMsgEvent(messageId, conversationId, userId, each.TargetPhotoId, originPhotoId, each.TargetUserId, each.SourceFeed, sourceIp, each.Text, each.ActionTime)
			//partitionKey = commons.GeneratePartitionKey(userId, each.TargetUserId)
			partitionKey = commons.GeneratePartitionKey(userId)

		case commons.ViewChatActionType:
			event = commons.NewUserViewChantEvent(userId, each.TargetPhotoId, originPhotoId, each.TargetUserId, each.SourceFeed, sourceIp, each.ActionTime, each.ViewTimeMillis)
			//partitionKey = commons.GeneratePartitionKey(userId, each.TargetUserId)
			partitionKey = commons.GeneratePartitionKey(userId)
		case commons.LocationActionType:
			event = commons.NewUserChangeLocationEvent(userId, sourceIp, each.Lat, each.Lon, each.ActionTime)
			partitionKey = commons.GeneratePartitionKey(userId)
		default:
			anlogger.Errorf(lc, "actions.go : unsupported action type [%s] for userId [%s]", each.ActionType, userId)
			errStr := commons.InternalServerError
			anlogger.Errorf(lc, "actions.go :  userId [%s], return %s to client", userId, errStr)
			return commons.NewServiceResponse(errStr), nil
		}
		commons.SendAnalyticEvent(event, userId, deliveryStreamName, awsDeliveryStreamClient, anlogger, lc)
		ok, errStr = commons.SendCommonEvent(event, userId, commonStreamName, partitionKey, awsKinesisClient, anlogger, lc)
		if !ok {
			errStr := commons.InternalServerError
			anlogger.Errorf(lc, "actions.go : userId [%s], return %s to client", userId, errStr)
			return commons.NewServiceResponse(errStr), nil
		}
	}

	resp := apimodel.ActionResponse{}
	if len(reqParam.Actions) > 0 {
		resp.LastActionTime = reqParam.Actions[len(reqParam.Actions)-1].ActionTime
	}

	body, err := json.Marshal(resp)
	if err != nil {
		anlogger.Errorf(lc, "actions.go : error while marshaling resp [%v] object for userId [%s] : %v", resp, userId, err)
		anlogger.Errorf(lc, "actions.go : userId [%s], return %s to client", userId, commons.InternalServerError)
		return commons.NewServiceResponse(commons.InternalServerError), nil
	}
	anlogger.Debugf(lc, "actions.go : return successful resp [%s] for userId [%s]", string(body), userId)
	anlogger.Infof(lc, "actions.go : successfully handle [%d] actions for userId [%s] with lastActionTime [%v]", len(reqParam.Actions), userId, resp.LastActionTime)
	return commons.NewServiceResponse(string(body)), nil
}

func checkUserUserIds(req *apimodel.ActionReq, userId string, lc *lambdacontext.LambdaContext) (bool, string) {
	anlogger.Debugf(lc, "actions.go : check that legal userIds were used, req %v for userId [%s]", req, userId)
	for _, each := range req.Actions {
		if each.TargetUserId == userId && each.ActionType != commons.LocationActionType {
			anlogger.Errorf(lc, "actions.go : error, use the same targetUserId [%s] and userId [%s] for action %v", each.TargetUserId, userId, each)
			return false, commons.WrongRequestParamsClientError
		}
	}
	anlogger.Debugf(lc, "actions.go : successfully check that legal userIds were used, req %v for userId [%s]", req, userId)
	return true, ""
}

func parseParams(params string, lc *lambdacontext.LambdaContext) (*apimodel.ActionReq, bool, string) {
	anlogger.Debugf(lc, "actions.go : parse request body %s", params)
	var req apimodel.ActionReq
	err := json.Unmarshal([]byte(params), &req)
	if err != nil {
		anlogger.Errorf(lc, "actions.go : error marshaling required params from the string [%s] : %v", params, err)
		return nil, false, commons.InternalServerError
	}

	if req.AccessToken == "" {
		anlogger.Errorf(lc, "actions.go : accessToken is empty", req)
		return nil, false, commons.WrongRequestParamsClientError
	}

	if len(req.Actions) == 0 {
		anlogger.Errorf(lc, "actions.go : actions are empty, req %v", req)
		return nil, false, commons.WrongRequestParamsClientError
	}

	for _, each := range req.Actions {
		if each.ActionType == "" {
			anlogger.Errorf(lc, "actions.go : one of the action's required param is nil, action %v", each)
			return nil, false, commons.WrongRequestParamsClientError
		}
		if _, ok := commons.ActionNames[each.ActionType]; !ok {
			anlogger.Errorf(lc, "actions.go : unsupported action type [%s]", each.ActionType)
			return nil, false, commons.WrongRequestParamsClientError
		}
		if each.SourceFeed == "" && each.ActionType != commons.LocationActionType {
			anlogger.Errorf(lc, "actions.go : sourceFeed required param is nil, req %v", req)
			return nil, false, commons.WrongRequestParamsClientError
		}
		if _, ok := commons.FeedNames[each.SourceFeed]; !ok && each.ActionType != commons.LocationActionType {
			anlogger.Errorf(lc, "actions.go : sourceFeed contains unsupported value [%s]", each.SourceFeed)
			return nil, false, commons.WrongRequestParamsClientError
		}
		if (each.TargetUserId == "" || each.TargetPhotoId == "") && each.ActionType != commons.LocationActionType {
			anlogger.Errorf(lc, "actions.go : one of the action's required param is nil, action %v", each)
			return nil, false, commons.WrongRequestParamsClientError
		}
		if each.LikeCount < 0 || each.ViewCount < 0 || each.ActionTime <= 0 || each.ViewTimeMillis < 0 || each.BlockReasonNum < 0 {
			anlogger.Errorf(lc, "actions.go : some of numeric param < 0")
			return nil, false, commons.WrongRequestParamsClientError
		}

		switch each.ActionType {
		case commons.LikeActionType:
			if each.LikeCount == 0 {
				anlogger.Errorf(lc, "actions.go : likeCount is 0 with action type %s", commons.LikeActionType)
				return nil, false, commons.WrongRequestParamsClientError
			}
		case commons.ViewActionType:
			if each.ViewCount == 0 || each.ViewTimeMillis == 0 {
				anlogger.Errorf(lc, "actions.go : (swallow this error coz ios bug) viewCount or viewTimeMillis is 0 with action type %s", commons.ViewActionType)
				//todo:need comment this coz a bug on ios
				//return nil, false, commons.WrongRequestParamsClientError
			}
		case commons.MessageActionType:
			if each.Text == "" {
				anlogger.Errorf(lc, "actions.go : text is empty with action type %s", commons.MessageActionType)
				return nil, false, commons.WrongRequestParamsClientError
			}
		case commons.ViewChatActionType:
			if each.SourceFeed != commons.FeedNameWhoLikedMe &&
				each.SourceFeed != commons.FeedNameMatches &&
				each.SourceFeed != commons.FeedNameMessages {
				anlogger.Errorf(lc, "actions.go : wrong source [%s] for action type %s", each.SourceFeed, commons.ViewChatActionType)
				return nil, false, commons.WrongRequestParamsClientError
			}
		case commons.LocationActionType:
			var defaultValue float64
			if each.Lat == defaultValue || each.Lon == defaultValue {
				anlogger.Errorf(lc, "actions.go : wrong source lat [%v] or lon [%v] for action type %s", each.Lat, each.Lon, commons.LocationActionType)
				return nil, false, commons.WrongRequestParamsClientError
			}
		}
	}

	anlogger.Debugf(lc, "actions.go : successfully parse request string [%s] to %v", params, req)
	return &req, true, ""
}
func generateConversationId(sourceUserId, targetUserId string, lc *lambdacontext.LambdaContext) (string, bool, string) {
	anlogger.Debugf(lc, "actions.go : generate conversation id for source user id [%s] and target user id [%s]",
		sourceUserId, targetUserId)
	arr := []string{sourceUserId, targetUserId}
	sort.Strings(arr)
	str := strings.Join(arr, "_")
	sha := sha1.New()
	_, err := io.WriteString(sha, str)
	if err != nil {
		anlogger.Errorf(lc, "actions.go : error while write base string for conversation id sha1 algo, base string [%s] : %v", str, err)
		return "", false, commons.InternalServerError
	}
	resultConversationId := fmt.Sprintf("%x", sha.Sum(nil))
	anlogger.Debugf(lc, "actions.go : successfully generate conversation id [%s] for source user id [%s] and target user id [%s]",
		resultConversationId, sourceUserId, targetUserId)
	return resultConversationId, true, ""
}

func generateMessageId(sourceUserId, text string, actionTime int64, lc *lambdacontext.LambdaContext) (string, bool, string) {
	sha := sha1.New()
	_, err := io.WriteString(sha, sourceUserId)
	if err != nil {
		anlogger.Errorf(lc, "actions.go : error while write source user id string to sha1 algo, source user id [%s] : %v", sourceUserId, err)
		return "", false, commons.InternalServerError
	}
	_, err = io.WriteString(sha, fmt.Sprintf("%v", actionTime))
	if err != nil {
		anlogger.Errorf(lc, "actions.go : error while write action time string to sha1 algo, actionTime [%v] : %v", actionTime, err)
		return "", false, commons.InternalServerError
	}
	_, err = io.WriteString(sha, text)
	if err != nil {
		anlogger.Errorf(lc, "actions.go : error while write text string to sha1 algo, text [%s] : %v", text, err)
		return "", false, commons.InternalServerError
	}

	resultMessageId := fmt.Sprintf("%x", sha.Sum(nil))
	anlogger.Debugf(lc, "actions.go : successfully generate message id [%s]", resultMessageId)
	return resultMessageId, true, ""
}

func main() {
	basicLambda.Start(handler)
}
