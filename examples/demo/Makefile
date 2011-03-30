run: clean example
	./example -hostOverride=www.example.com
 
include $(GOROOT)/src/Make.inc

TARG=example
DEPS=../../server ../../websocket ../../pprof
GOFILES=\
    main.go\
    core.go\
    chat.go\
    multipart.go\

include $(GOROOT)/src/Make.cmd
