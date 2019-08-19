Release v3.0.6 (2019-08-19)
===========================
- Updated `iam-go-sdk` to v1.1.1 

Release v3.0.5 (2019-07-27)
===========================
- Fixed event codes

Release v3.0.4 (2019-07-27)
===========================
- Fixed `aud` and `scope` validation error response  

Release v3.0.3 (2019-07-25)
===========================
- Separating `aud` and `scope` validation  

Release v3.0.1 (2019-07-24)
===========================
- Fixed `aud` validation error message  

Release v3.0.0 (2019-07-23)
===========================
- Added `aud` and `scope` token fields validation    

Release v2.0.1 (2019-04-26)
===========================
- Export Trace `TraceIDKey` and Session `SessionIDKey` request header constants publicly

Release v2.0.0 (2019-04-24)
===========================
### BREAKING CHANGES
1. package: [logger-event](https://github.com/AccelByte/go-restful-plugins/tree/master/pkg/logger/event): 
    * Change  [event struct](https://github.com/AccelByte/go-restful-plugins/blob/master/pkg/logger/event/event.go) to match new log event standard.
    * Refactor log parameters. (Include ```Debug```, ```Info```, ```Warn```, ```Error```, ```Fatal```)
        * example: ```Info()``` from ```Info(request, 99, "get user info message")``` to ```Info(request, 99, 50, 3, "get user info message")``` , 50 is eventType and 3 is eventLevel.

