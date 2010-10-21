run: clean goinstall example
	./example 
 
include $(GOROOT)/src/Make.inc

DEPS=../../server ../../oauth
TARG=example
GOFILES=\
    main.go\
    settings.go\

include $(GOROOT)/src/Make.cmd
