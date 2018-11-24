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
	"sort"
	"github.com/ringoid/commons"
)

var anlogger *commons.Logger
var awsDeliveryStreamClient *firehose.Firehose
var deliveryStreamName string
var internalAuthFunctionName string
var awsKinesisClient *kinesis.Kinesis
var commonStreamName string
var clientLambda *lambda.Lambda

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

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	lc, _ := lambdacontext.FromContext(ctx)

	anlogger.Debugf(lc, "actions.go : start handle request %v", request)

	if commons.IsItWarmUpRequest(request.Body, anlogger, lc) {
		return events.APIGatewayProxyResponse{}, nil
	}

	appVersion, isItAndroid, ok, errStr := commons.ParseAppVersionFromHeaders(request.Headers, anlogger, lc)
	if !ok {
		anlogger.Errorf(lc, "actions.go : return %s to client", errStr)
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: errStr}, nil
	}

	reqParam, ok, errStr := parseParams(request.Body, lc)
	if !ok {
		anlogger.Errorf(lc, "actions.go : return %s to client", errStr)
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: errStr}, nil
	}

	userId, ok, _, errStr := commons.CallVerifyAccessToken(appVersion, isItAndroid, reqParam.AccessToken, internalAuthFunctionName, clientLambda, anlogger, lc)
	if !ok {
		anlogger.Errorf(lc, "actions.go : return %s to client", errStr)
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: errStr}, nil
	}

	if ok, errStr = checkUserUserIds(reqParam, userId, lc); !ok {
		anlogger.Errorf(lc, "actions.go : return %s to client", errStr)
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: errStr}, nil
	}
	//todo:future place of optimization - we can use batch request model later
	for _, each := range reqParam.Actions {
		var event interface{}
		var partitionKey string
		originPhotoId, ok := commons.GetOriginPhotoId(userId, each.TargetPhotoId, anlogger, lc)
		if !ok && each.ActionType != commons.BlockActionType {
			errStr := commons.InternalServerError
			anlogger.Errorf(lc, "actions.go :  userId [%s], return %s to client", userId, errStr)
			return events.APIGatewayProxyResponse{StatusCode: 200, Body: errStr}, nil
		}
		switch each.ActionType {
		case commons.LikeActionType:
			event = commons.NewUserLikePhotoEvent(userId, each.TargetPhotoId, originPhotoId, each.TargetUserId, each.SourceFeed, each.LikeCount, each.ActionTime, "")
			partitionKey = generatePartitionKey(userId, each.TargetUserId)
		case commons.ViewActionType:
			event = commons.NewUserViewPhotoEvent(userId, each.TargetPhotoId, originPhotoId, each.TargetUserId, each.SourceFeed, each.ViewCount, each.ViewTimeSec, each.ActionTime, "")
			partitionKey = generatePartitionKey(userId, each.TargetUserId)
		case commons.BlockActionType:
			event = commons.NewUserBlockOtherEvent(userId, each.TargetUserId, each.SourceFeed, each.ActionTime, "")
			partitionKey = generatePartitionKey(userId, each.TargetUserId)
		case commons.UnlikeActionType:
			event = commons.NewUserUnLikePhotoEvent(userId, each.TargetPhotoId, originPhotoId, each.TargetUserId, each.SourceFeed, each.ActionTime, "")
			partitionKey = generatePartitionKey(userId, each.TargetUserId)
		default:
			anlogger.Errorf(lc, "actions.go : unsupported action type [%s] for userId [%s]", each.ActionType, userId)
			errStr := commons.InternalServerError
			anlogger.Errorf(lc, "actions.go :  userId [%s], return %s to client", userId, errStr)
			return events.APIGatewayProxyResponse{StatusCode: 200, Body: errStr}, nil
		}
		commons.SendAnalyticEvent(event, userId, deliveryStreamName, awsDeliveryStreamClient, anlogger, lc)
		ok, errStr = commons.SendCommonEvent(event, userId, commonStreamName, partitionKey, awsKinesisClient, anlogger, lc)
		if !ok {
			errStr := commons.InternalServerError
			anlogger.Errorf(lc, "actions.go : userId [%s], return %s to client", userId, errStr)
			return events.APIGatewayProxyResponse{StatusCode: 200, Body: errStr}, nil
		}
	}

	resp := commons.BaseResponse{}
	body, err := json.Marshal(resp)
	if err != nil {
		anlogger.Errorf(lc, "actions.go : error while marshaling resp [%v] object for userId [%s] : %v", resp, userId, err)
		anlogger.Errorf(lc, "actions.go : userId [%s], return %s to client", userId, commons.InternalServerError)
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: commons.InternalServerError}, nil
	}
	anlogger.Debugf(lc, "actions.go : return successful resp [%s] for userId [%s]", string(body), userId)
	anlogger.Infof(lc, "actions.go : successfully handle all actions for userId [%s]", userId)
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(body)}, nil
}

func generatePartitionKey(arr ... string) string {
	sort.Strings(arr)
	if len(arr) == 0 {
		return ""
	}
	key := arr[0]
	for index, each := range arr {
		if index != 0 {
			key += "_" + each
		}
	}
	return key
}

func checkUserUserIds(req *apimodel.ActionReq, userId string, lc *lambdacontext.LambdaContext) (bool, string) {
	anlogger.Debugf(lc, "actions.go : check that legal userIds were used, req %v for userId [%s]", req, userId)
	for _, each := range req.Actions {
		if each.TargetUserId == userId {
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

	if req.Actions == nil {
		anlogger.Errorf(lc, "actions.go : actions required param is nil, req %v", req)
		return nil, false, commons.WrongRequestParamsClientError
	}

	for _, each := range req.Actions {
		if len(each.SourceFeed) == 0 {
			anlogger.Errorf(lc, "actions.go : sourceFeed required param is nil, req %v", req)
			return nil, false, commons.WrongRequestParamsClientError
		}
		if _, ok := commons.FeedNames[each.SourceFeed]; !ok {
			anlogger.Errorf(lc, "actions.go : sourceFeed contains unsupported value [%s]", each.SourceFeed)
			return nil, false, commons.WrongRequestParamsClientError
		}
		if each.ActionType == "" || each.TargetUserId == "" {
			anlogger.Errorf(lc, "actions.go : one of the action's required param is nil, action %v", each)
			return nil, false, commons.WrongRequestParamsClientError
		}
		if each.ActionType == commons.LikeActionType ||
			each.ActionType == commons.ViewActionType ||
			each.ActionType == commons.UnlikeActionType {
			if each.TargetPhotoId == "" {
				anlogger.Errorf(lc, "actions.go : one of the action's required param is nil, action %v", each)
				return nil, false, commons.WrongRequestParamsClientError
			}
		}
		if _, ok := commons.ActionNames[each.ActionType]; !ok {
			anlogger.Errorf(lc, "actions.go : unsupported action type [%s]", each.ActionType)
			return nil, false, commons.WrongRequestParamsClientError
		}
	}

	anlogger.Debugf(lc, "actions.go : successfully parse request string [%s] to %v", params, req)
	return &req, true, ""
}

func main() {
	basicLambda.Start(handler)
}
