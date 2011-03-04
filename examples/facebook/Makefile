run: example
	./example 
 
include $(GOROOT)/src/Make.inc

DEPS=$(GOROOT)/src/pkg/github.com/garyburd/twister/server $(GOROOT)/src/pkg/github.com/garyburd/twister/oauth
TARG=example
GOFILES=\
    main.go\
    template.go

include $(GOROOT)/src/Make.cmd
