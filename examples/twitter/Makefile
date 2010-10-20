run: clean goinstall example
	./example 
 
include $(GOROOT)/src/Make.inc

DEPS=../../server ../../web ../../oauth
TARG=example
GOFILES=\
    main.go\
    settings.go\

include $(GOROOT)/src/Make.cmd

goinstall:
	goinstall github.com/garyburd/twister/web
	goinstall github.com/garyburd/twister/server
	goinstall github.com/garyburd/twister/oauth

