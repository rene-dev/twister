all: install

DIRS=web server oauth websocket expvar examples/demo examples/twitter examples/facebook
TEST=web oauth server

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

