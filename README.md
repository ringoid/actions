# Actions Service

### STAGE API ENDPOINT IS `` https://yk3a6opus2.execute-api.eu-west-1.amazonaws.com``
### PROD API ENDPOINT IS ````


### Action url

* url ``https://{API ENDPOINT}/Prod/actions``

POST request

Headers:

* x-ringoid-android-buildnum : 1       //int, x-ringoid-ios-buildnum in case of iOS
* Content-Type : application/json

Body:

    {
        "accessToken":"adfsdfsdfsdfsdfs",
        "sourceFeed":"new_faces", // who_liked_me, matches, messages (messages_inbox, messages_starred, messages_sent)
        "actions":[ACTION_OBJECT, ACTION_OBJECT]
    }
    
    all parameters are required
    
 Response Body:
 
    {
        "errorCode":"",
        "errorMessage":""
    }
    
Possible errorCodes:

* InternalServerError
* InvalidAccessTokenClientError
* WrongRequestParamsClientError
* TooOldAppVersionClientError

## Possible ACTION_OBJECTS

1. LIKE

    {
        "actionType":"LIKE",
        "targetPhotoId":"640x480_ksjdhfkjhhsh",
        "targetUserId":"skdfkjhkjsdhf",
        "likeCount":12,
        "actionTime":12342342354 //unix time
    }

2. VIEW

    {
       "actionType":"VIEW",
       "targetPhotoId":"640x480_ksjdhfkjhhsh",
       "targetUserId":"skdfkjhkjsdhf",
       "viewCount":5,
       "viewTimeSec":45,
       "actionTime":12342342354 //unix time
    }

3. BLOCK

    {
       "actionType":"BLOCK",
       "targetUserId":"skdfkjhkjsdhf",
       "actionTime":12342342354 //unix time
    }

4. UNLIKE

    {
        "actionType":"UNLIKE",
        "targetPhotoId":"640x480_ksjdhfkjhhsh",
        "targetUserId":"skdfkjhkjsdhf",
        "actionTime":12342342354 //unix time
    }


## Analytics Events

1. ACTION_USER_LIKE_PHOTO

* userId - string
* photoId - string
* originPhotoId - string
* targetUserId - string
* likeCount - int
* likedAt - int
* source - string
* unixTime - int
* eventType - string (ACTION_USER_LIKE_PHOTO)
* internalServiceSource - string

2. ACTION_USER_VIEW_PHOTO

* userId - string
* photoId - string
* originPhotoId - string
* targetUserId - string
* viewCount - int
* viewTimeSec - int
* viewAt - int
* source - string
* unixTime - int
* eventType - string (ACTION_USER_VIEW_PHOTO)
* internalServiceSource - string

3. ACTION_USER_BLOCK_OTHER

* userId - string
* targetUserId - string
* blockedAt - int
* source - string
* unixTime - int
* eventType - string (ACTION_USER_BLOCK_OTHER)
* internalServiceSource - string

4. ACTION_USER_UNLIKE_PHOTO

* userId - string
* photoId - string
* originPhotoId - string
* targetUserId - string
* unLikedAt - int
* source - string
* unixTime - int
* eventType - string (ACTION_USER_UNLIKE_PHOTO)
* internalServiceSource - string
