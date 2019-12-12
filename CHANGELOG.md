Release v3.1.3 (2019-12-12)
===========================
- Add `WriteErrorWithEventID` in `response` package
- Update `Write` in `response` package

Release v3.1.2 (2019-10-21)
===========================
- Update auth function to return error response based on standard
- Update auth error response field name to camelCase

Release v3.1.1 (2019-10-21)
===========================
- Update ip extraction on `logger/common` package

Release v3.1.0 (2019-10-18)
===========================
- Add `trace` package
- Update `response` package for Error Code Standards
- Update ExtractDefault in `util` package

Release v3.0.9 (2019-10-02)
===========================
- Update `iam-go-sdk` to v1.1.2

Release v3.0.8 (2019-09-23)
===========================
- Added `response` package
- Added `util` package

Release v3.0.7 (2019-09-19)
===========================
- Updated go module to denote v3 

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

