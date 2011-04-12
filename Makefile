all: install

DIRS=web server oauth websocket expvar pprof examples/demo examples/twitter examples/facebook examples/wiki
TEST=web oauth server websocket

clean.dirs: $(addsuffix .clean, $(DIRS))
install.dirs: $(addsuffix .install, $(DIRS))
test.dirs: $(addsuffix .test, $(TEST))

%.clean:
	+cd $* && gomake clean

%.install:
	+cd $* && gomake install

%.test:
	+cd $* && gomake test

clean: clean.dirs

install: install.dirs

test:	test.dirs

