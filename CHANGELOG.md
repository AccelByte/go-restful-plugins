Release v2.0.0 (2019-04-24)
===========================
### BREAKING CHANGES
1. package: [logger-event](https://github.com/AccelByte/go-restful-plugins/tree/master/pkg/logger/event): 
    * Change  [event struct](https://github.com/AccelByte/go-restful-plugins/blob/master/pkg/logger/event/event.go) to match new log event standard.
    * Refactor log parameters. (Include ```Debug```, ```Info```, ```Warn```, ```Error```, ```Fatal```)
        * example: ```Info()``` from ```Info(request, 99, "get user info message")``` to ```Info(request, 99, 50, 3, "get user info message")``` , 50 is event and 3 is eventLevel.

